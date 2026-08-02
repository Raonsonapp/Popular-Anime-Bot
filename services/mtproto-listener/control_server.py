"""Combined web-based login + control API for the MTProto listener,
reachable through the Go API's reverse proxy at /telegram-login/*
(see cmd/api/main.go).

Runs for the entire lifetime of the process. Unlike an earlier version of
this setup (a separate weblogin_server.py that exited once login
succeeded, requiring main.py to take over on the next restart), this
server never stops: it serves the login endpoints until the account is
authorized, and the status/backfill/register-storage endpoints for as
long as the process runs afterward. A successful login saves the
Telethon StringSession to Postgres via ApiClient.save_listener_session,
so a future redeploy on a PaaS with ephemeral disk (Render's free tier)
reconnects using the saved session instead of requiring this flow again.
"""

import asyncio
import logging

from aiohttp import web
from telethon import TelegramClient
from telethon.errors import PhoneCodeExpiredError, PhoneCodeInvalidError, SessionPasswordNeededError

from api_client import ApiClient
from backfill import backfill_all
from config import Config
from register_channels import register_all

logger = logging.getLogger("control_server")

CONTROL_PORT = 8091

_phone_code_hash: str | None = None
_phone_number: str | None = None


def check_key(request) -> bool:
    return bool(Config.INTERNAL_API_KEY) and request.query.get("key") == Config.INTERNAL_API_KEY


def page(body: str, status: int = 200) -> web.Response:
    return web.Response(
        status=status,
        content_type="text/html",
        text=f"<html><body style='font-family:sans-serif;padding:24px;font-size:18px;line-height:1.6'>{body}</body></html>",
    )


def make_index_handler(client: TelegramClient):
    async def handle_index(request):
        key = request.query.get("key", "YOUR_INTERNAL_API_KEY")
        if await client.is_user_authorized():
            return page(
                "✅ Already logged in - the listener is active and the session is saved "
                f"permanently, no need to repeat this. See <code>/telegram-login/status?key={key}</code>."
            )
        return page(
            "<b>Telegram login helper</b><br><br>"
            "1) Send the login code to your phone:<br>"
            f"<code>/telegram-login/send-code?key={key}&phone=992XXXXXXXXX</code><br><br>"
            "2) Then confirm the code Telegram sent you:<br>"
            f"<code>/telegram-login/sign-in?key={key}&code=12345</code>"
        )

    return handle_index


def make_send_code_handler(client: TelegramClient):
    async def handle_send_code(request):
        global _phone_code_hash, _phone_number
        if not check_key(request):
            return page("❌ Invalid or missing ?key=", status=403)

        phone = request.query.get("phone", "").strip()
        if not phone:
            return page("Missing ?phone= (digits only, e.g. phone=992XXXXXXXXX)", status=400)
        if not phone.startswith("+"):
            phone = "+" + phone

        try:
            sent = await client.send_code_request(phone)
            _phone_code_hash = sent.phone_code_hash
            _phone_number = phone
            key = request.query.get("key")
            return page(
                f"✅ Code sent to {phone}. Check your Telegram app, then open:<br>"
                f"<code>/telegram-login/sign-in?key={key}&code=YOUR_CODE</code>"
            )
        except Exception as e:
            logger.exception("send_code_request failed")
            return page(f"❌ Failed to send code: {e}", status=500)

    return handle_send_code


def make_sign_in_handler(client: TelegramClient, api: ApiClient, on_authorized):
    async def handle_sign_in(request):
        if not check_key(request):
            return page("❌ Invalid or missing ?key=", status=403)
        if _phone_code_hash is None:
            return page("❌ Call send-code first (see /telegram-login/).", status=400)

        code = request.query.get("code", "").strip()
        password = request.query.get("password")
        key = request.query.get("key")

        try:
            if password:
                await client.sign_in(password=password)
            elif code:
                await client.sign_in(phone=_phone_number, code=code, phone_code_hash=_phone_code_hash)
            else:
                return page("Missing ?code= (or ?password= if you have 2FA enabled).", status=400)

            me = await client.get_me()

            await api.save_listener_session(client.session.save())
            results = await register_all(client, api)
            await on_authorized()

            channels_html = "<br>".join(results) or "(no channels configured in register_channels.py)"
            return page(
                f"✅ Logged in as {me.first_name} (@{me.username}). The session is now saved "
                "permanently in your database - future redeploys will reconnect automatically, "
                "no need to log in again.<br><br>"
                f"<b>Source channels:</b><br>{channels_html}<br><br>"
                "The listener is now active."
            )
        except SessionPasswordNeededError:
            return page(
                "🔒 This account has a 2FA password. Open:<br>"
                f"<code>/telegram-login/sign-in?key={key}&password=YOUR_2FA_PASSWORD</code>"
            )
        except (PhoneCodeInvalidError, PhoneCodeExpiredError):
            return page("❌ Invalid or expired code - start over from /telegram-login/.", status=400)
        except Exception as e:
            logger.exception("sign_in failed")
            return page(f"❌ Sign-in failed: {e}", status=500)

    return handle_sign_in


def make_backfill_handler(client: TelegramClient):
    async def handle_backfill(request):
        if not check_key(request):
            return web.Response(status=403, text="Invalid or missing ?key=")
        if not await client.is_user_authorized():
            return web.Response(status=400, text="Not logged in yet - complete /telegram-login/ first.")

        raw_limit = request.query.get("limit", "300").strip().lower()
        if raw_limit in ("0", "all", "none", "unlimited"):
            limit = None  # Telethon's iter_messages(limit=None) walks the entire history.
        else:
            try:
                limit = int(raw_limit)
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
        limit_desc = "the entire history" if limit is None else f"up to {limit} messages"
        return web.Response(
            text=f"Backfill started ({limit_desc} per channel). This can take a while for "
            "channels with a lot of history - it runs in the background regardless. "
            "Check the Render Logs tab for progress."
        )

    return handle_backfill


def make_status_handler(client: TelegramClient):
    async def handle_status(request):
        if not check_key(request):
            return web.Response(status=403, text="Invalid or missing ?key=")

        authorized = await client.is_user_authorized()
        lines = [f"Logged in: {'YES' if authorized else 'NO - complete /telegram-login/ first'}"]
        if not authorized:
            return web.Response(text="\n".join(lines))

        api = ApiClient(Config.API_BASE_URL, Config.INTERNAL_API_KEY)
        try:
            channels = await api.list_source_channels(active_only=False)
        finally:
            await api.close()

        storage_registered = any(ch["telegram_channel_id"] == Config.STORAGE_CHANNEL_ID for ch in channels)
        lines.append(f"Storage channel id: {Config.STORAGE_CHANNEL_ID}")
        lines.append(
            "Storage channel registered as a source: " + ("YES" if storage_registered else "NO (see /register-storage below)")
        )
        lines.append("")
        lines.append(f"Registered source channels ({len(channels)}):")
        for ch in channels:
            active = "active" if ch.get("is_active") else "inactive"
            lines.append(f"  - {ch['title']} (id={ch['telegram_channel_id']}, {ch['source_language']}, {active})")

        return web.Response(text="\n".join(lines))

    return handle_status


def make_register_storage_handler(client: TelegramClient):
    async def handle_register_storage(request):
        if not check_key(request):
            return web.Response(status=403, text="Invalid or missing ?key=")
        if not await client.is_user_authorized():
            return web.Response(status=400, text="Not logged in yet - complete /telegram-login/ first.")
        if not Config.STORAGE_CHANNEL_ID:
            return web.Response(status=400, text="STORAGE_CHANNEL_ID is not set")

        lang = request.query.get("lang", "fa")

        api = ApiClient(Config.API_BASE_URL, Config.INTERNAL_API_KEY)
        try:
            existing = await api.list_source_channels(active_only=False)
            if any(ch["telegram_channel_id"] == Config.STORAGE_CHANNEL_ID for ch in existing):
                return web.Response(text="Already registered - nothing to do. New posts in it should already be picked up.")

            entity = await client.get_entity(Config.STORAGE_CHANNEL_ID)
            title = getattr(entity, "title", "Storage channel")
            await api.create_source_channel(Config.STORAGE_CHANNEL_ID, None, title, lang)
        finally:
            await api.close()

        return web.Response(
            text=f"✅ Registered your storage channel ('{title}') as a source too "
            f"(language={lang}). New posts in it will now be picked up within "
            "~60 seconds, no forwarding needed since it's already the storage channel."
        )

    return handle_register_storage


async def start_control_server(client: TelegramClient, api: ApiClient, on_authorized):
    """on_authorized is an async callback invoked exactly once, right after
    a fresh web login succeeds, so the caller can start the live listener
    (register event handlers, kick off the channel-list refresher) without
    needing to restart the process."""
    app = web.Application()
    app.router.add_get("/telegram-login/", make_index_handler(client))
    app.router.add_get("/telegram-login/send-code", make_send_code_handler(client))
    app.router.add_get("/telegram-login/sign-in", make_sign_in_handler(client, api, on_authorized))
    app.router.add_get("/telegram-login/backfill", make_backfill_handler(client))
    app.router.add_get("/telegram-login/status", make_status_handler(client))
    app.router.add_get("/telegram-login/register-storage", make_register_storage_handler(client))

    runner = web.AppRunner(app)
    await runner.setup()
    site = web.TCPSite(runner, "127.0.0.1", CONTROL_PORT)
    await site.start()
    logger.info("control server listening on 127.0.0.1:%s", CONTROL_PORT)
