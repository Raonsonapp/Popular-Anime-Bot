"""Turns a single Telegram message into a catalog entry. Shared between
the live listener (main.py, for new/edited messages) and the history
backfill (backfill.py, for messages that existed before the channel was
registered).
"""

import logging

from telethon import TelegramClient
from telethon.tl.types import DocumentAttributeVideo

from api_client import ApiClient
from config import Config
from parser import parse_post
from translator import translate_to_persian

logger = logging.getLogger("processor")


def is_video_message(message) -> bool:
    if message.video:
        return True
    if message.document:
        for attr in message.document.attributes:
            if isinstance(attr, DocumentAttributeVideo):
                return True
    return False


async def resolve_storage_message_id(client: TelegramClient, channel: dict, message) -> int:
    """Returns the message id in the storage channel that the bot will later
    copyMessage from. If the source channel *is* the storage channel (the
    simple single-channel setup: you post directly into your own private
    channel and the bot is admin there too), no forward is needed - the
    message is already exactly where it needs to be."""
    if channel["telegram_channel_id"] == Config.STORAGE_CHANNEL_ID:
        return message.id

    forwarded = await client.forward_messages(Config.STORAGE_CHANNEL_ID, message)
    if isinstance(forwarded, list):
        forwarded = forwarded[0]
    return forwarded.id


async def handle_episode(client: TelegramClient, api: ApiClient, channel: dict, message):
    text = message.message or ""
    parsed = parse_post(text)
    if not parsed.title:
        logger.warning("could not extract a title, skipping message %s", message.id)
        await api.create_import_log(channel["id"], message.id, "skipped", detail="no title extracted")
        return

    storage_message_id = await resolve_storage_message_id(client, channel, message)

    title_persian = translate_to_persian(parsed.title, channel["source_language"], Config.TRANSLATE_ENABLED)

    anime = await api.upsert_anime(
        {
            "source_channel_id": channel["id"],
            "title_original": parsed.title,
            "title_persian": title_persian,
            "year": parsed.year or 0,
            "auto_publish": Config.AUTO_PUBLISH,
        }
    )

    episode = await api.upsert_episode(
        {
            "anime_id": anime["id"],
            "episode_number": parsed.episode_number or 0,
            "quality": parsed.quality,
            "storage_chat_id": Config.STORAGE_CHANNEL_ID,
            "storage_message_id": storage_message_id,
            "source_channel_id": channel["id"],
            "source_message_id": message.id,
        }
    )

    await api.create_import_log(
        channel["id"], message.id, "new", detail=f"episode {parsed.episode_number}",
        anime_id=anime["id"], episode_id=episode["id"],
    )
    logger.info("imported episode %s of anime '%s' (id=%s)", parsed.episode_number, title_persian, anime["id"])


async def handle_announcement(client: TelegramClient, api: ApiClient, channel: dict, message):
    text = message.message or ""
    parsed = parse_post(text)
    if not parsed.title:
        return

    poster_chat_id = None
    poster_message_id = None
    if message.photo:
        poster_chat_id = Config.STORAGE_CHANNEL_ID
        poster_message_id = await resolve_storage_message_id(client, channel, message)

    title_persian = translate_to_persian(parsed.title, channel["source_language"], Config.TRANSLATE_ENABLED)
    synopsis_persian = translate_to_persian(parsed.synopsis, channel["source_language"], Config.TRANSLATE_ENABLED)
    genres_persian = [
        translate_to_persian(g, channel["source_language"], Config.TRANSLATE_ENABLED) for g in parsed.genres
    ]

    payload = {
        "source_channel_id": channel["id"],
        "title_original": parsed.title,
        "title_persian": title_persian,
        "synopsis_persian": synopsis_persian,
        "synopsis_original": parsed.synopsis,
        "year": parsed.year or 0,
        "studio_name": parsed.studio or "",
        "genre_names_english": parsed.genres,
        "genre_names_persian": genres_persian,
        "auto_publish": Config.AUTO_PUBLISH,
    }
    if poster_chat_id:
        payload["poster_storage_chat_id"] = poster_chat_id
        payload["poster_storage_message_id"] = poster_message_id

    anime = await api.upsert_anime(payload)
    await api.create_import_log(channel["id"], message.id, "new", detail="anime announcement", anime_id=anime["id"])
    logger.info("imported/updated anime announcement '%s' (id=%s)", title_persian, anime["id"])


def should_process(message) -> str | None:
    """Classifies a message as 'episode', 'announcement', or None (skip)."""
    if is_video_message(message):
        return "episode"
    if message.photo or (message.message and len(message.message) > 40):
        return "announcement"
    return None
