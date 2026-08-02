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
