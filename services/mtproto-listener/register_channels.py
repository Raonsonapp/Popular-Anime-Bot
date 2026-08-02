"""One-time helper to register a list of public Telegram channels as
source_channels, without having to manually look up each channel's
numeric id.

Run this standalone AFTER `login.py` (the userbot session must already
exist), e.g. via a VPS shell:

    python register_channels.py

(On Render's free tier, which has no Shell access, `register_all` below
runs automatically right after a successful login via weblogin_server.py
instead - you don't need to run this file directly there.)

It resolves each @username via the already-authenticated Telethon
session, joins the channel (so the account actually receives new-message
updates from it - Telegram only pushes those to channels you're a member
of), and registers it with the core API. Safe to re-run: channels already
registered are skipped.

Edit the CHANNELS list below to add/remove channels.
"""

import asyncio
import logging

from telethon import TelegramClient
from telethon.tl.functions.channels import JoinChannelRequest
from telethon.utils import get_peer_id

from api_client import ApiClient
from config import Config

logger = logging.getLogger("register_channels")

# (username, source_language) - source_language is "fa" for these since
# they're already Farsi-dubbed anime channels (no RU/EN->FA translation
# needed for their captions).
CHANNELS = [
    ("anime_doble_31", "fa"),
    ("Animehangoutll", "fa"),
    ("Animemifarsi", "fa"),
    ("Anime_Dubble", "fa"),
    ("Animenayzz", "fa"),
    ("animeloveyoukiramuddin", "fa"),
    ("Anime_Duble_Farsi", "fa"),
    ("Farsi_dub", "fa"),
]


async def register_all(client: TelegramClient, api: ApiClient) -> list[str]:
    """Resolves, joins, and registers every channel in CHANNELS using an
    already-connected+authorized client. Returns a list of human-readable
    result lines (one per channel), for callers that want to show them."""
    existing = {ch["telegram_channel_id"] for ch in await api.list_source_channels(active_only=False)}
    results = []

    for username, lang in CHANNELS:
        try:
            entity = await client.get_entity(username)
            channel_id = get_peer_id(entity)

            if channel_id in existing:
                msg = f"already registered, skipping: @{username} (id={channel_id})"
                logger.info(msg)
                results.append(msg)
                continue

            try:
                await client(JoinChannelRequest(entity))
            except Exception:
                logger.warning("could not join @%s (may already be a member) - continuing", username)

            title = getattr(entity, "title", username)
            await api.create_source_channel(channel_id, username, title, lang)
            msg = f"registered @{username} -> id={channel_id} title={title!r}"
            logger.info(msg)
            results.append(msg)
        except Exception as e:
            msg = f"FAILED @{username}: {e}"
            logger.exception("failed to register @%s", username)
            results.append(msg)

    return results


async def main():
    logging.basicConfig(level=logging.INFO, format="%(asctime)s %(levelname)s %(name)s: %(message)s")
    if not Config.API_ID or not Config.API_HASH:
        raise SystemExit("TELEGRAM_API_ID and TELEGRAM_API_HASH are required")

    client = TelegramClient(Config.SESSION_NAME, Config.API_ID, Config.API_HASH)
    await client.start()

    api = ApiClient(Config.API_BASE_URL, Config.INTERNAL_API_KEY)
    await register_all(client, api)

    await client.disconnect()
    await api.close()


if __name__ == "__main__":
    asyncio.run(main())
