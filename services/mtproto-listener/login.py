"""One-time interactive login for the MTProto userbot session.

Run this once before starting the listener service for the first time:

    docker compose run --rm mtproto-listener python login.py

(or `python login.py` directly if you have the venv/deps locally). It will
prompt for your phone number, the login code Telegram sends you, and your
2FA password if you have one enabled, then saves the resulting Telethon
StringSession to Postgres (via the API's internal /listener-session route)
so the long-running `mtproto-listener` service can pick it up on its next
start without any further interaction - this also means the session
survives redeploys/restarts even on hosts with no persistent disk.
"""

import asyncio

from telethon import TelegramClient
from telethon.sessions import StringSession

from api_client import ApiClient
from config import Config


async def main():
    if not Config.API_ID or not Config.API_HASH:
        raise SystemExit("TELEGRAM_API_ID and TELEGRAM_API_HASH are required (see .env.example)")

    client = TelegramClient(StringSession(), Config.API_ID, Config.API_HASH)
    await client.start()
    me = await client.get_me()

    api = ApiClient(Config.API_BASE_URL, Config.INTERNAL_API_KEY)
    try:
        await api.save_listener_session(client.session.save())
    finally:
        await api.close()

    print(f"Logged in as {me.first_name} (@{me.username}). Session saved to the database.")
    await client.disconnect()


if __name__ == "__main__":
    asyncio.run(main())
