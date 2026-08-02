"""Web-based Telegram login, for deployments with no interactive terminal
(e.g. Render's free tier, which has no Shell access).

entrypoint.sh runs this instead of main.py whenever no session file exists
yet. It exposes a few GET endpoints - reachable through the Go API's
reverse proxy at /telegram-login/... - that a human can complete entirely
by opening links in a phone browser, no CLI needed. Once login succeeds,
it writes the session file and exits; entrypoint.sh's loop then starts the
real main.py automatically.
"""

import asyncio
import logging

from aiohttp import web
from telethon import TelegramClient
from telethon.errors import PhoneCodeExpiredError, PhoneCodeInvalidError, SessionPasswordNeededError

from api_client import ApiClient
from config import Config
from register_channels import register_all

logging.basicConfig(level=logging.INFO, format="%(asctime)s %(levelname)s %(name)s: %(message)s")
logger = logging.getLogger("weblogin")

LOGIN_PORT = 8091

client: TelegramClient = None
phone_code_hash: str = None
phone_number: str = None
login_done = asyncio.Event()


def check_key(request) -> bool:
    return bool(Config.INTERNAL_API_KEY) and request.query.get("key") == Config.INTERNAL_API_KEY


def page(body: str, status: int = 200) -> web.Response:
    return web.Response(
        status=status,
        content_type="text/html",
        text=f"<html><body style='font-family:sans-serif;padding:24px;font-size:18px;line-height:1.6'>{body}</body></html>",
    )


async def handle_index(request):
    key = request.query.get("key", "YOUR_INTERNAL_API_KEY")
    return page(
        "<b>Telegram login helper</b><br><br>"
        "1) Send the login code to your phone:<br>"
        f"<code>/telegram-login/send-code?key={key}&phone=992XXXXXXXXX</code><br><br>"
        "2) Then confirm the code Telegram sent you:<br>"
        f"<code>/telegram-login/sign-in?key={key}&code=12345</code>"
    )


async def handle_send_code(request):
    global client, phone_code_hash, phone_number
    if not check_key(request):
        return page("❌ Invalid or missing ?key=", status=403)

    phone = request.query.get("phone", "").strip()
    if not phone:
        return page("Missing ?phone= (digits only, e.g. phone=992XXXXXXXXX)", status=400)
    if not phone.startswith("+"):
        phone = "+" + phone

    try:
        client = TelegramClient(Config.SESSION_NAME, Config.API_ID, Config.API_HASH)
        await client.connect()
        sent = await client.send_code_request(phone)
        phone_code_hash = sent.phone_code_hash
        phone_number = phone
        key = request.query.get("key")
        return page(
            f"✅ Code sent to {phone}. Check your Telegram app, then open:<br>"
            f"<code>/telegram-login/sign-in?key={key}&code=YOUR_CODE</code>"
        )
    except Exception as e:
        logger.exception("send_code_request failed")
        return page(f"❌ Failed to send code: {e}", status=500)


async def handle_sign_in(request):
    global client
    if not check_key(request):
        return page("❌ Invalid or missing ?key=", status=403)
    if client is None or phone_code_hash is None:
        return page("❌ Call send-code first (see /telegram-login/).", status=400)

    code = request.query.get("code", "").strip()
    password = request.query.get("password")
    key = request.query.get("key")

    try:
        if password:
            await client.sign_in(password=password)
        elif code:
            await client.sign_in(phone=phone_number, code=code, phone_code_hash=phone_code_hash)
        else:
            return page("Missing ?code= (or ?password= if you have 2FA enabled).", status=400)

        me = await client.get_me()

        api = ApiClient(Config.API_BASE_URL, Config.INTERNAL_API_KEY)
        results = await register_all(client, api)
        await api.close()

        await client.disconnect()
        login_done.set()

        channels_html = "<br>".join(results) or "(no channels configured in register_channels.py)"
        return page(
            f"✅ Logged in as {me.first_name} (@{me.username}).<br><br>"
            f"<b>Source channels:</b><br>{channels_html}<br><br>"
            "The listener will start automatically within ~10 seconds."
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


async def main():
    if not Config.API_ID or not Config.API_HASH:
        raise SystemExit("TELEGRAM_API_ID and TELEGRAM_API_HASH are required")

    app = web.Application()
    app.router.add_get("/telegram-login/", handle_index)
    app.router.add_get("/telegram-login/send-code", handle_send_code)
    app.router.add_get("/telegram-login/sign-in", handle_sign_in)

    runner = web.AppRunner(app)
    await runner.setup()
    site = web.TCPSite(runner, "127.0.0.1", LOGIN_PORT)
    await site.start()
    logger.info("web login helper listening on 127.0.0.1:%s", LOGIN_PORT)

    await login_done.wait()
    await runner.cleanup()


if __name__ == "__main__":
    asyncio.run(main())
