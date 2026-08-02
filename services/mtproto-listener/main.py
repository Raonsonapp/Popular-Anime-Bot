import asyncio
import logging

from telethon import TelegramClient, events

from api_client import ApiClient
from config import Config
from control_server import start_control_server
from processor import handle_announcement, handle_episode, should_process

logging.basicConfig(level=logging.INFO, format="%(asctime)s %(levelname)s %(name)s: %(message)s")
logger = logging.getLogger("mtproto-listener")

# telegram_channel_id -> source_channel dict (id, source_language, ...)
active_channels: dict[int, dict] = {}


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


def build_handlers(client: TelegramClient, api: ApiClient):
    @client.on(events.NewMessage(incoming=True))
    async def on_new_message(event):
        channel = active_channels.get(event.chat_id)
        if not channel:
            return
        try:
            kind = should_process(event.message)
            if kind == "episode":
                await handle_episode(client, api, channel, event.message)
            elif kind == "announcement":
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
            kind = should_process(event.message)
            if kind == "episode":
                await handle_episode(client, api, channel, event.message)
            elif kind == "announcement":
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

    await start_control_server(client)

    refresh_task = asyncio.create_task(refresh_channels(api))

    try:
        await client.run_until_disconnected()
    finally:
        refresh_task.cancel()
        await api.close()


if __name__ == "__main__":
    asyncio.run(main())
