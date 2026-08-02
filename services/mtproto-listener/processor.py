"""Turns a single Telegram message into a catalog entry. Shared between
the live listener (main.py, for new/edited messages) and the history
backfill (backfill.py, for messages that existed before the channel was
registered).
"""

import asyncio
import logging

from telethon import TelegramClient
from telethon.tl.types import DocumentAttributeVideo

from api_client import ApiClient
from config import Config
from deep_link_fetcher import (
    episode_number_from_delivered,
    extract_episode_deep_links,
    fetch_batch_files,
    fetch_episode_file,
    relay_to_storage,
)
from parser import is_adult_content, is_removed_placeholder, is_subtitle_only, parse_post
from translator import translate_to_persian

BETWEEN_LINKED_EPISODES_DELAY = 3  # be gentle - each one drives a second bot

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
    if is_removed_placeholder(text):
        await api.create_import_log(channel["id"], message.id, "skipped", detail="message removed by Telegram (copyright), nothing to import")
        return

    parsed = parse_post(text)
    if not parsed.title:
        logger.warning("could not extract a title, skipping message %s", message.id)
        await api.create_import_log(channel["id"], message.id, "skipped", detail="no title extracted")
        return

    if is_adult_content(parsed, text):
        logger.info("skipping adult-flagged message %s ('%s')", message.id, parsed.title)
        await api.create_import_log(channel["id"], message.id, "skipped", detail="adult content marker matched")
        return

    if is_subtitle_only(text):
        logger.info("skipping subtitle-only (non-dubbed) message %s ('%s')", message.id, parsed.title)
        await api.create_import_log(channel["id"], message.id, "skipped", detail="Persian-subtitle-only, not dubbed")
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
    if is_removed_placeholder(text):
        await api.create_import_log(channel["id"], message.id, "skipped", detail="message removed by Telegram (copyright), nothing to import")
        return

    parsed = parse_post(text)
    if not parsed.title:
        return

    if is_adult_content(parsed, text):
        logger.info("skipping adult-flagged message %s ('%s')", message.id, parsed.title)
        await api.create_import_log(channel["id"], message.id, "skipped", detail="adult content marker matched")
        return

    if is_subtitle_only(text):
        logger.info("skipping subtitle-only (non-dubbed) message %s ('%s')", message.id, parsed.title)
        await api.create_import_log(channel["id"], message.id, "skipped", detail="Persian-subtitle-only, not dubbed")
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

    if Config.FETCH_LINKED_EPISODES:
        await fetch_linked_episodes(client, api, channel, message, anime["id"], parsed.quality)


async def _store_linked_episode(
    api: ApiClient, client: TelegramClient, channel: dict, message, anime_id: int, quality: str,
    episode_number: int, bot_username: str, file_message,
):
    """Shared tail end for both a single-episode link and one file out of
    a batch link: dub/subtitle-check the delivered file, relay it into the
    storage channel, and upsert the episode."""
    delivered_text = file_message.message or ""
    if is_subtitle_only(delivered_text):
        logger.info(
            "discarding linked episode %s via @%s - delivered file is subtitle-only, not dubbed",
            episode_number, bot_username,
        )
        await api.create_import_log(
            channel["id"], message.id, "skipped",
            detail=f"linked episode {episode_number} via @{bot_username} was subtitle-only, discarded",
        )
        return

    storage_message_id = await relay_to_storage(client, file_message)
    episode = await api.upsert_episode(
        {
            "anime_id": anime_id,
            "episode_number": episode_number,
            "quality": quality,
            "storage_chat_id": Config.STORAGE_CHANNEL_ID,
            "storage_message_id": storage_message_id,
            "source_channel_id": channel["id"],
            # Synthetic but stable/unique per episode so re-processing this
            # same announcement updates rather than duplicates.
            "source_message_id": message.id * 1000 + episode_number,
        }
    )
    await api.create_import_log(
        channel["id"], message.id, "new",
        detail=f"linked episode {episode_number} via @{bot_username}",
        anime_id=anime_id, episode_id=episode["id"],
    )
    logger.info("fetched linked episode %s for anime %s via @%s", episode_number, anime_id, bot_username)


async def fetch_linked_episodes(client: TelegramClient, api: ApiClient, channel: dict, message, anime_id: int, quality: str):
    """Some channels hide each episode behind a link into a separate file
    delivery bot instead of attaching the video to the post (see
    deep_link_fetcher.py). Handles two shapes: a link naming one specific
    episode, and a single "batch" link (e.g. "E01_E20") whose one /start
    streams back many files at once - each identified by its own filename."""
    all_links = extract_episode_deep_links(message)
    if not all_links:
        logger.info("message %s: no delivery-bot deep links found (poster/metadata-only post)", message.id)
        return

    single_links = [link for link in all_links if link["episode_number"] is not None]
    batch_links = [link for link in all_links if link["episode_number"] is None and link.get("is_batch")]
    unresolved = [link for link in all_links if link["episode_number"] is None and not link.get("is_batch")]
    if unresolved:
        logger.info(
            "message %s: found %d deep link(s) with no readable episode number: %s",
            message.id, len(unresolved), [l["label"] for l in unresolved],
        )
    if not single_links and not batch_links:
        return
    logger.info(
        "message %s: found %d single + %d batch linked episode(s), fetching...",
        message.id, len(single_links), len(batch_links),
    )

    for link in single_links:
        try:
            # If the channel already labels this exact episode's link as
            # subtitle-only (e.g. "Eposide_1 - زیرنویس فارسی"), skip it
            # without ever pressing start - no point fetching what we'd
            # discard anyway.
            if is_subtitle_only(link.get("label", "")):
                await api.create_import_log(
                    channel["id"], message.id, "skipped",
                    detail=f"linked episode {link['episode_number']} labeled subtitle-only, not fetched",
                )
                continue

            file_message = await fetch_episode_file(
                client, link["bot_username"], link["start_param"], link["episode_number"]
            )
            if not file_message:
                await api.create_import_log(
                    channel["id"], message.id, "skipped",
                    detail=f"linked episode {link['episode_number']} via @{link['bot_username']} did not arrive",
                )
                continue

            await _store_linked_episode(
                api, client, channel, message, anime_id, quality,
                link["episode_number"], link["bot_username"], file_message,
            )
        except Exception:
            logger.exception("failed to fetch linked episode %s via @%s", link.get("episode_number"), link.get("bot_username"))

        await asyncio.sleep(BETWEEN_LINKED_EPISODES_DELAY)

    for link in batch_links:
        try:
            if is_subtitle_only(link.get("label", "")):
                await api.create_import_log(
                    channel["id"], message.id, "skipped",
                    detail=f"batch link '{link['label']}' labeled subtitle-only, not fetched",
                )
                continue

            file_messages = await fetch_batch_files(client, link["bot_username"], link["start_param"])
            if not file_messages:
                await api.create_import_log(
                    channel["id"], message.id, "skipped",
                    detail=f"batch link '{link['label']}' via @{link['bot_username']} returned no files",
                )
                continue

            logger.info(
                "message %s: batch link '%s' via @%s returned %d file(s)",
                message.id, link["label"], link["bot_username"], len(file_messages),
            )
            for file_message in file_messages:
                episode_number = episode_number_from_delivered(file_message)
                if episode_number is None:
                    logger.info(
                        "batch file from @%s had no recognizable episode number, skipping: %s",
                        link["bot_username"], getattr(file_message.file, "name", None) or file_message.message,
                    )
                    continue
                await _store_linked_episode(
                    api, client, channel, message, anime_id, quality,
                    episode_number, link["bot_username"], file_message,
                )
        except Exception:
            logger.exception("failed to fetch batch link '%s' via @%s", link.get("label"), link.get("bot_username"))

        await asyncio.sleep(BETWEEN_LINKED_EPISODES_DELAY)


def should_process(message) -> str | None:
    """Classifies a message as 'episode', 'announcement', or None (skip)."""
    if is_video_message(message):
        return "episode"
    if message.photo or (message.message and len(message.message) > 40):
        return "announcement"
    return None
