import logging

from deep_translator import GoogleTranslator

logger = logging.getLogger("translator")


def translate_to_persian(text: str, source_language: str, enabled: bool = True) -> str:
    """Best-effort translation to Persian. Falls back to the original text
    on any failure (offline, rate-limited, empty input, already Persian)."""
    if not enabled or not text or not text.strip():
        return text

    if source_language == "fa":
        return text

    lang_map = {"ru": "russian", "en": "english"}
    source = lang_map.get(source_language, "auto")

    try:
        return GoogleTranslator(source=source, target="persian").translate(text)
    except Exception:
        logger.warning("translation failed, falling back to original text", exc_info=True)
        return text
