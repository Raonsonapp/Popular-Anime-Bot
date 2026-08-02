"""Imports a source channel's existing message history, not just messages
that arrive after it was registered. Uses the same per-message logic as
the live listener (processor.py).
"""

import asyncio
import logging

from telethon import TelegramClient

from api_client import ApiClient
from processor import handle_announcement, handle_episode, should_process

logger = logging.getLogger("backfill")

DEFAULT_LIMIT = 300


async def backfill_channel(client: TelegramClient, api: ApiClient, channel: dict, limit: int | None = DEFAULT_LIMIT) -> int:
    """Walks a channel's message history (oldest of the fetched batch
    first) and imports anything that looks like an episode or anime
    announcement. Returns how many messages were successfully processed."""
    count = 0
    async for message in client.iter_messages(channel["telegram_channel_id"], limit=limit, reverse=True):
        try:
            kind = should_process(message)
            if kind == "episode":
                await handle_episode(client, api, channel, message)
                count += 1
            elif kind == "announcement":
                await handle_announcement(client, api, channel, message)
                count += 1
        except Exception:
            logger.exception(
                "backfill: failed on message %s in channel %s", message.id, channel["telegram_channel_id"]
            )
        await asyncio.sleep(0.5)  # be gentle with Telegram's rate limits
    return count


async def backfill_all(client: TelegramClient, api: ApiClient, limit: int | None = DEFAULT_LIMIT) -> dict[str, str]:
    channels = await api.list_source_channels(active_only=True)
    results = {}
    for channel in channels:
        logger.info("backfilling %s (limit=%s)...", channel["title"], limit)
        try:
            n = await backfill_channel(client, api, channel, limit)
            results[channel["title"]] = f"{n} messages imported"
            logger.info("backfilled %s: %d messages imported", channel["title"], n)
        except Exception as e:
            results[channel["title"]] = f"FAILED: {e}"
            logger.exception("backfill failed for %s", channel["title"])
    return results
