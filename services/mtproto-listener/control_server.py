"""Small local-only control API for the already-running listener, proxied
through the Go API at /telegram-login/* (same prefix weblogin_server.py
uses - they never run at the same time, so sharing the port is safe).

Currently just exposes a way to trigger a history backfill after the
fact, e.g. for channels that were registered before this endpoint
existed, or to re-run with a different limit.
"""

import asyncio
import logging

from aiohttp import web
from telethon import TelegramClient

from api_client import ApiClient
from backfill import backfill_all
from config import Config

logger = logging.getLogger("control_server")

CONTROL_PORT = 8091


def check_key(request) -> bool:
    return bool(Config.INTERNAL_API_KEY) and request.query.get("key") == Config.INTERNAL_API_KEY


def make_backfill_handler(client: TelegramClient):
    async def handle_backfill(request):
        if not check_key(request):
            return web.Response(status=403, text="Invalid or missing ?key=")
        try:
            limit = int(request.query.get("limit", "300"))
        except ValueError:
            limit = 300

        async def run():
            api = ApiClient(Config.API_BASE_URL, Config.INTERNAL_API_KEY)
            try:
                results = await backfill_all(client, api, limit)
                logger.info("backfill finished: %s", results)
            finally:
                await api.close()

        asyncio.create_task(run())
        return web.Response(
            text=f"Backfill started (up to {limit} messages per channel). "
            "This runs in the background - check the Render Logs tab for progress."
        )

    return handle_backfill


async def start_control_server(client: TelegramClient):
    app = web.Application()
    app.router.add_get("/telegram-login/backfill", make_backfill_handler(client))

    runner = web.AppRunner(app)
    await runner.setup()
    site = web.TCPSite(runner, "127.0.0.1", CONTROL_PORT)
    await site.start()
    logger.info("control server listening on 127.0.0.1:%s", CONTROL_PORT)
