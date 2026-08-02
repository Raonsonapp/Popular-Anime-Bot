"""One-time interactive login for the MTProto userbot session.

Run this once before starting the listener service for the first time:

    docker compose run --rm mtproto-listener python login.py

(or `python login.py` directly if you have the venv/deps locally). It will
prompt for your phone number, the login code Telegram sends you, and your
2FA password if you have one enabled, then writes the session file to
SESSION_NAME (see .env.example) so the long-running `mtproto-listener`
service can connect afterwards without any further interaction.
"""

import asyncio

from telethon import TelegramClient

from config import Config


async def main():
    if not Config.API_ID or not Config.API_HASH:
        raise SystemExit("TELEGRAM_API_ID and TELEGRAM_API_HASH are required (see .env.example)")

    client = TelegramClient(Config.SESSION_NAME, Config.API_ID, Config.API_HASH)
    await client.start()
    me = await client.get_me()
    print(f"Logged in as {me.first_name} (@{me.username}). Session saved to {Config.SESSION_NAME}")
    await client.disconnect()


if __name__ == "__main__":
    asyncio.run(main())
