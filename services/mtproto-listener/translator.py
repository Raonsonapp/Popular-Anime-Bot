import logging

from deep_translator import GoogleTranslator

logger = logging.getLogger("translator")


def translate_to_tajik(text: str, source_language: str, enabled: bool = True) -> str:
    """Best-effort translation into Tajik (Cyrillic script) - this is a
    Tajik bot/channel, not a Farsi one. Even Farsi-sourced text needs this:
    Farsi and Tajik are the same spoken language but written in different
    scripts (Perso-Arabic vs Cyrillic), so passing Farsi through unchanged
    would leave the wrong alphabet in an otherwise-Tajik catalog. Falls
    back to the original text on any failure (offline, rate-limited,
    empty input, already Tajik)."""
    if not enabled or not text or not text.strip():
        return text
    if source_language == "tg":
        return text

    lang_map = {"ru": "russian", "en": "english", "fa": "persian"}
    source = lang_map.get(source_language, "auto")

    try:
        return GoogleTranslator(source=source, target="tajik").translate(text)
    except Exception:
        logger.warning("translation failed, falling back to original text", exc_info=True)
        return text
