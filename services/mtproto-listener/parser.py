"""Heuristic, regex-based extraction of anime metadata from a Telegram
channel post's caption/text.

This is intentionally simple: real-world channels format posts wildly
differently, so this covers the common patterns (title on the first line,
"Episode N", quality tags, a "Genre:" line) and leaves everything else as
free text. Treat it as an MVP baseline - the roadmap calls for a proper
NLP-based extractor down the line.
"""

import re
from dataclasses import dataclass, field

# Text-based signal for adult/18+ content. This only catches content that
# is self-labeled (genre tags, warning emoji, etc.) - it can't inspect
# image/video pixels, so it's a first line of defense, not a guarantee.
ADULT_CONTENT_MARKERS = (
    "🔞",
    "18+",
    "r18",
    "r-18",
    "nsfw",
    "hentai",
    "хентай",
    "ecchi",
    "эччи",
    "porn",
    "порно",
    "xxx",
    "erotic",
    "эротик",
    "explicit content",
)

# The user only wants Persian-DUBBED anime (audio track), not
# subtitled-only releases. Some posts self-label which one they are (e.g.
# "زیرنویس فارسی چسبیده" - hardsubbed Persian subtitle, no dub at all).
SUBTITLE_MARKERS = ("زیرنویس", "زیر نویس", "زیرنوشت", "hardsub", "softsub", "subtitle", "sub:")
DUB_MARKERS = ("دوبله", "دوبلاژ", "dubbed", "dub:")
# "بدون زیرنویس" = "without subtitles" - a post saying this is boasting
# it's PURE dub with no subtitles at all, so a plain substring match on
# SUBTITLE_MARKERS would otherwise misread this as subtitle-only and wrongly
# discard genuinely dubbed content.
NEGATED_SUBTITLE_RE = re.compile(r"(?:بدون|بی|no|without)\s*(?:هیچ\s*)?(?:زیرنویس|زیر\s*نویس|subtitle)", re.IGNORECASE)

# Many channels post a yes/no checklist line per property (dub, censorship,
# etc.) marked with an emoji rather than the word "no"/"without" - e.g. a
# "🚫 دوبله بدون سانسور" line means that dub does NOT apply to this release
# (compare a "✅" in front of the same phrase on a release that does have
# it). A plain substring check for "دوبله" can't tell these apart and would
# wrongly treat the 🚫 case as confirmation of a dub.
NEGATIVE_MARK_RE = re.compile(r"[🚫❌⛔🚷🔴👎✖]")
POSITIVE_MARK_RE = re.compile(r"[✅✔☑🟢👍]")

# Telegram itself injects this placeholder text (in the viewer's own
# client, not the actual message) when a message was taken down over a
# copyright complaint - there's no real content left in it at all. Without
# this check, that boilerplate gets parsed as if it were the anime's
# title, importing junk "anime" entries for messages that no longer
# contain anything.
REMOVED_MESSAGE_MARKERS = (
    "couldn't be displayed on your device due to copyright",
    "message was deleted",
    "this message is unavailable",
)

QUALITY_RE = re.compile(r"\b(480p|720p|1080p|2160p|4k)\b", re.IGNORECASE)
EPISODE_RE = re.compile(
    r"(?:episode|epi?sode|\bep\b|قسمت|серия|эпизод)\s*[\-:#]?\s*(\d{1,4})", re.IGNORECASE
)
YEAR_RE = re.compile(r"\b(19|20)\d{2}\b")
GENRE_LINE_RE = re.compile(r"(?:genre|жанр|ژانر)\s*[:：]\s*(.+)", re.IGNORECASE)
STUDIO_LINE_RE = re.compile(r"(?:studio|студия|استودیو)\s*[:：]\s*(.+)", re.IGNORECASE)
BRACKET_TAG_RE = re.compile(r"[\[\(].*?[\]\)]")


@dataclass
class ParsedPost:
    title: str = ""
    is_episode: bool = False
    episode_number: int | None = None
    quality: str = "720p"
    year: int | None = None
    studio: str | None = None
    genres: list[str] = field(default_factory=list)
    synopsis: str = ""


def _first_line(text: str) -> str:
    for line in text.splitlines():
        line = line.strip()
        if line:
            return line
    return ""


def _clean_title(line: str) -> str:
    cleaned = BRACKET_TAG_RE.sub("", line)
    cleaned = EPISODE_RE.sub("", cleaned)
    cleaned = QUALITY_RE.sub("", cleaned)
    return re.sub(r"\s{2,}", " ", cleaned).strip(" -–:|")


def is_adult_content(parsed: ParsedPost, raw_text: str) -> bool:
    """Best-effort check for self-labeled adult/18+ content, so it never
    gets imported into the catalog. Checks the title, genres, and the full
    raw caption (covers warning lines the structured fields don't capture)."""
    haystack = " ".join([parsed.title, " ".join(parsed.genres), raw_text or ""]).lower()
    return any(marker in haystack for marker in ADULT_CONTENT_MARKERS)


def is_removed_placeholder(raw_text: str) -> bool:
    """True if this is Telegram's own copyright-takedown placeholder text,
    not real post content - there's nothing to import here."""
    haystack = (raw_text or "").lower()
    return any(marker in haystack for marker in REMOVED_MESSAGE_MARKERS)


def _line_dub_status(line: str) -> str | None:
    """'has_dub' / 'no_dub' / None (this line says nothing about dubbing),
    accounting for a "🚫 دوبله ..." checklist-style line meaning that dub
    does NOT apply - not proof that it does, which a bare word match would
    otherwise assume."""
    if not any(marker.lower() in line.lower() for marker in DUB_MARKERS):
        return None
    if NEGATIVE_MARK_RE.search(line):
        return "no_dub"
    return "has_dub"


def is_subtitle_only(raw_text: str) -> bool:
    """True if the post is not Persian-dubbed - either it explicitly says
    so (a "🚫 دوبله" checklist line) or it labels itself as subtitled with
    no mention of a dub at all. The user wants dubbed audio only, not
    subtitles over the original audio."""
    text = raw_text or ""
    if NEGATED_SUBTITLE_RE.search(text):
        return False  # "without subtitles" is a pro-dub statement, not a subtitle marker

    dub_status = None
    for line in text.splitlines():
        status = _line_dub_status(line)
        if status == "no_dub":
            return True  # explicit "dub does not apply" line is decisive
        if status == "has_dub":
            dub_status = "has_dub"
    if dub_status == "has_dub":
        return False

    haystack = text.lower()
    has_subtitle_marker = any(marker.lower() in haystack for marker in SUBTITLE_MARKERS)
    return has_subtitle_marker


def parse_post(text: str) -> ParsedPost:
    text = text or ""
    result = ParsedPost()

    episode_match = EPISODE_RE.search(text)
    if episode_match:
        result.is_episode = True
        result.episode_number = int(episode_match.group(1))

    quality_match = QUALITY_RE.search(text)
    if quality_match:
        result.quality = quality_match.group(1).lower()

    year_match = YEAR_RE.search(text)
    if year_match:
        result.year = int(year_match.group(0))

    genre_match = GENRE_LINE_RE.search(text)
    if genre_match:
        raw = genre_match.group(1)
        result.genres = [g.strip() for g in re.split(r"[,/،]", raw) if g.strip()]

    studio_match = STUDIO_LINE_RE.search(text)
    if studio_match:
        result.studio = studio_match.group(1).strip()

    first_line = _first_line(text)
    result.title = _clean_title(first_line) or first_line

    # Everything after the first line, minus metadata lines we already parsed,
    # is treated as the synopsis.
    remaining_lines = text.splitlines()[1:]
    synopsis_lines = [
        line
        for line in remaining_lines
        if line.strip()
        and not GENRE_LINE_RE.search(line)
        and not STUDIO_LINE_RE.search(line)
        and not EPISODE_RE.search(line)
        and not (QUALITY_RE.search(line) and len(line.strip()) < 30)
        and not re.fullmatch(r"(?:year|год|سال)\s*[:：]?\s*\d{4}", line.strip(), re.IGNORECASE)
    ]
    result.synopsis = "\n".join(synopsis_lines).strip()

    return result
