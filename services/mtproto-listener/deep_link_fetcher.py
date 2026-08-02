"""Some Farsi anime channels never attach the actual video file to their
channel post at all - the post is just a poster + caption where each
episode is a hidden hyperlink (text like "Eposide_3" that actually links
to a completely different bot, e.g. t.me/Anime_loveri?start=xxxx). Tapping
it opens that bot, which replies with the real video file in DM (often
auto-deleting it again a short time later), sometimes only after you join
a "sponsor" channel it names first.

This module drives that flow programmatically with the same userbot
session already used to watch the source channels, so those episodes make
it into the catalog as real, playable episodes instead of poster-only
"announcements". The user explicitly asked for this (including
auto-joining sponsor channels if a bot demands it) rather than settling
for metadata-only imports.
"""

import io
import logging
import re

from telethon import TelegramClient
from telethon.tl.functions.channels import JoinChannelRequest
from telethon.tl.types import MessageEntityTextUrl, MessageEntityUrl

from config import Config

logger = logging.getLogger("deep_link_fetcher")

# Two ways a deep link shows up in the wild: a normal t.me/<domain mirror>
# URL, or Telegram Desktop/some bots' own "tg://resolve?domain=...&start=..."
# scheme - both need to be recognized, or extraction silently finds nothing.
DEEP_LINK_RE = re.compile(
    r"(?:t(?:elegram)?\.me|telegram\.dog)/(?P<user1>[A-Za-z0-9_]{5,32})/?\?start=(?P<param1>[A-Za-z0-9_\-]+)"
    r"|tg://resolve\?domain=(?P<user2>[A-Za-z0-9_]{5,32})&start=(?P<param2>[A-Za-z0-9_\-]+)",
    re.IGNORECASE,
)
EPISODE_NUMBER_RE = re.compile(r"(\d{1,4})")
SPONSOR_CHANNEL_RE = re.compile(r"t\.me/([A-Za-z0-9_]{5,32})/?$", re.IGNORECASE)

RESPONSE_TIMEOUT = 25  # seconds to wait for the delivery bot to reply
MAX_JOIN_RETRIES = 2


def _parse_deep_link(url: str):
    m = DEEP_LINK_RE.search(url or "")
    if not m:
        return None
    bot_username = m.group("user1") or m.group("user2")
    start_param = m.group("param1") or m.group("param2")
    return bot_username, start_param


def extract_episode_deep_links(message) -> list[dict]:
    """Returns [{"episode_number": int|None, "bot_username": str, "start_param": str}, ...]
    for every deep link the post points at a delivery bot with - whether
    it's hidden behind hyperlinked caption text or an inline button
    attached under the post (both are common, depending on the channel)."""
    links = []
    seen = set()

    for entity, entity_text in message.get_entities_text():
        if isinstance(entity, MessageEntityTextUrl):
            url = entity.url
        elif isinstance(entity, MessageEntityUrl):
            url = entity_text
        else:
            continue
        parsed = _parse_deep_link(url)
        if not parsed:
            continue
        bot_username, start_param = parsed
        if (bot_username, start_param) in seen:
            continue
        seen.add((bot_username, start_param))
        ep_match = EPISODE_NUMBER_RE.search(entity_text)
        links.append(
            {
                "episode_number": int(ep_match.group(1)) if ep_match else None,
                "bot_username": bot_username,
                "start_param": start_param,
            }
        )

    for row in message.buttons or []:
        for button in row:
            if not button.url:
                continue
            parsed = _parse_deep_link(button.url)
            if not parsed:
                continue
            bot_username, start_param = parsed
            if (bot_username, start_param) in seen:
                continue
            seen.add((bot_username, start_param))
            ep_match = EPISODE_NUMBER_RE.search(button.text or "")
            links.append(
                {
                    "episode_number": int(ep_match.group(1)) if ep_match else None,
                    "bot_username": bot_username,
                    "start_param": start_param,
                }
            )

    return links


def _sponsor_channels_requested(response) -> list[str]:
    """The near-universal pattern for these gates: an inline URL button
    linking to the channel you must join, usually next to a 'try again'
    button. Detected from the button URLs, not the (highly variable)
    prompt wording."""
    channels = []
    for row in response.buttons or []:
        for button in row:
            url = getattr(button, "url", None)
            if not url:
                continue
            m = SPONSOR_CHANNEL_RE.search(url)
            if m:
                channels.append(m.group(1))
    return channels


async def _attempt(client: TelegramClient, bot_entity, start_param: str):
    async with client.conversation(bot_entity, timeout=RESPONSE_TIMEOUT) as conv:
        await conv.send_message(f"/start {start_param}")
        while True:
            resp = await conv.get_response()
            if resp.video or resp.document:
                return "file", resp
            sponsor_channels = _sponsor_channels_requested(resp)
            if sponsor_channels:
                return "join", sponsor_channels
            # Some bots send a "please wait"/"processing" text first - keep
            # waiting for the real reply instead of giving up on it.


async def fetch_episode_file(client: TelegramClient, bot_username: str, start_param: str):
    """Simulates tapping a t.me/<bot>?start=<param> deep link and returns
    the video/document message the bot sends back, joining any sponsor
    channel(s) it demands along the way. Returns None on timeout or if it
    keeps demanding channels we already joined (broken/private channel)."""
    try:
        bot_entity = await client.get_entity(bot_username)
    except Exception:
        logger.warning("could not resolve delivery bot @%s", bot_username)
        return None

    joined: set[str] = set()
    for _ in range(MAX_JOIN_RETRIES + 1):
        try:
            kind, payload = await _attempt(client, bot_entity, start_param)
        except TimeoutError:
            logger.warning("@%s: no file for start=%s within %ss", bot_username, start_param, RESPONSE_TIMEOUT)
            return None

        if kind == "file":
            return payload

        new_channels = [c for c in payload if c not in joined]
        if not new_channels:
            logger.warning("@%s keeps asking to join %s - giving up", bot_username, payload)
            return None
        for channel in new_channels:
            joined.add(channel)
            try:
                await client(JoinChannelRequest(channel))
                logger.info("joined sponsor channel @%s to unlock @%s delivery", channel, bot_username)
            except Exception:
                logger.warning("could not join sponsor channel @%s", channel)

    logger.warning("@%s: gave up after %d join attempt(s)", bot_username, MAX_JOIN_RETRIES)
    return None


async def relay_to_storage(client: TelegramClient, message) -> int:
    """Copies a fetched episode file into the storage channel so the bot
    can serve it later via copyMessage, same as any other episode. Falls
    back to download+re-upload if the delivery bot disabled forwarding
    (common - it's how they keep the file inside their own bot)."""
    try:
        forwarded = await client.forward_messages(Config.STORAGE_CHANNEL_ID, message)
        return forwarded[0].id if isinstance(forwarded, list) else forwarded.id
    except Exception:
        logger.info("forwarding blocked, falling back to download+re-upload")
        buf = io.BytesIO()
        await client.download_media(message, file=buf)
        buf.seek(0)
        buf.name = getattr(message.file, "name", None) or "episode.mp4"
        sent = await client.send_file(Config.STORAGE_CHANNEL_ID, buf, caption=message.text or "")
        return sent.id
