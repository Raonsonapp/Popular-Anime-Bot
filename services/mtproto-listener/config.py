import os

from dotenv import load_dotenv

load_dotenv()


def _int(name: str, default: int = 0) -> int:
    value = os.getenv(name)
    return int(value) if value else default


def _bool(name: str, default: bool = False) -> bool:
    value = os.getenv(name)
    if value is None:
        return default
    return value.strip().lower() in ("1", "true", "yes", "on")


class Config:
    # https://my.telegram.org credentials for the *userbot* MTProto session.
    # Never commit these - keep them only in a local .env / secret store.
    API_ID = _int("TELEGRAM_API_ID")
    API_HASH = os.getenv("TELEGRAM_API_HASH", "")
    SESSION_NAME = os.getenv("SESSION_NAME", "/data/userbot.session")

    # Private channel (bot must be admin there) episodes/posters get forwarded
    # into. The bot service delivers content to end users via copyMessage
    # against this channel, so it never needs raw Telethon file references.
    STORAGE_CHANNEL_ID = _int("STORAGE_CHANNEL_ID")

    API_BASE_URL = os.getenv("API_BASE_URL", "http://api:8080")
    INTERNAL_API_KEY = os.getenv("INTERNAL_API_KEY", "")

    CHANNEL_REFRESH_SECONDS = _int("CHANNEL_REFRESH_SECONDS", 60)
    AUTO_PUBLISH = _bool("AUTO_PUBLISH", True)
    TRANSLATE_ENABLED = _bool("TRANSLATE_ENABLED", True)

    # Only used when deployed as a PaaS "web service" (e.g. Render's free
    # tier) that requires something listening on $PORT. Ignored for
    # docker-compose/VPS deployments.
    PORT = os.getenv("PORT", "")
