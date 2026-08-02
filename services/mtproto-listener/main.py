import asyncio
import logging

from telethon import TelegramClient, events
from telethon.tl.types import DocumentAttributeVideo

from api_client import ApiClient
from config import Config
from parser import parse_post
from translator import translate_to_persian

logging.basicConfig(level=logging.INFO, format="%(asctime)s %(levelname)s %(name)s: %(message)s")
logger = logging.getLogger("mtproto-listener")

# telegram_channel_id -> source_channel dict (id, source_language, ...)
active_channels: dict[int, dict] = {}


def is_video_message(message) -> bool:
    if message.video:
        return True
    if message.document:
        for attr in message.document.attributes:
            if isinstance(attr, DocumentAttributeVideo):
                return True
    return False


async def refresh_channels(api: ApiClient):
    while True:
        try:
            channels = await api.list_source_channels(active_only=True)
            active_channels.clear()
            for ch in channels:
                active_channels[ch["telegram_channel_id"]] = ch
            logger.info("watching %d source channel(s)", len(active_channels))
        except Exception:
            logger.exception("failed to refresh source channel list")
        await asyncio.sleep(Config.CHANNEL_REFRESH_SECONDS)


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


def build_handlers(client: TelegramClient, api: ApiClient):
    @client.on(events.NewMessage(incoming=True))
    async def on_new_message(event):
        channel = active_channels.get(event.chat_id)
        if not channel:
            return
        try:
            if is_video_message(event.message):
                await handle_episode(client, api, channel, event.message)
            elif event.message.photo or (event.message.message and len(event.message.message) > 40):
                await handle_announcement(client, api, channel, event.message)
            await api.update_source_channel_cursor(channel["id"], event.message.id)
        except Exception:
            logger.exception("failed to process new message %s in channel %s", event.message.id, event.chat_id)
            await api.create_import_log(channel["id"], event.message.id, "error", detail="exception while processing")

    @client.on(events.MessageEdited(incoming=True))
    async def on_edited_message(event):
        channel = active_channels.get(event.chat_id)
        if not channel:
            return
        try:
            if is_video_message(event.message):
                await handle_episode(client, api, channel, event.message)
            else:
                await handle_announcement(client, api, channel, event.message)
        except Exception:
            logger.exception("failed to process edited message %s in channel %s", event.message.id, event.chat_id)

    @client.on(events.MessageDeleted())
    async def on_deleted_message(event):
        if event.chat_id is None:
            return
        channel = active_channels.get(event.chat_id)
        if not channel:
            return
        for message_id in event.deleted_ids:
            try:
                await api.delete_episode_by_source(channel["id"], message_id)
                await api.create_import_log(channel["id"], message_id, "deleted")
            except Exception:
                logger.exception("failed to mark message %s deleted", message_id)


async def start_health_server(port: str):
    """Minimal HTTP server so this can run as a Render (or similar PaaS)
    free-tier "web service", which requires something bound to $PORT.
    Not used for docker-compose/VPS deployments."""
    from aiohttp import web

    app = web.Application()
    app.router.add_get("/healthz", lambda request: web.Response(text="ok"))
    runner = web.AppRunner(app)
    await runner.setup()
    site = web.TCPSite(runner, "0.0.0.0", int(port))
    await site.start()
    logger.info("health server listening on port %s", port)


async def main():
    if not Config.API_ID or not Config.API_HASH:
        raise SystemExit("TELEGRAM_API_ID and TELEGRAM_API_HASH are required")
    if not Config.STORAGE_CHANNEL_ID:
        raise SystemExit("STORAGE_CHANNEL_ID is required")

    api = ApiClient(Config.API_BASE_URL, Config.INTERNAL_API_KEY)
    client = TelegramClient(Config.SESSION_NAME, Config.API_ID, Config.API_HASH)

    build_handlers(client, api)

    if Config.PORT:
        await start_health_server(Config.PORT)

    await client.start()
    logger.info("MTProto listener connected")

    refresh_task = asyncio.create_task(refresh_channels(api))

    try:
        await client.run_until_disconnected()
    finally:
        refresh_task.cancel()
        await api.close()


if __name__ == "__main__":
    asyncio.run(main())
