import logging
from typing import Any, Optional

import httpx

logger = logging.getLogger("api_client")


class ApiClient:
    """Thin client for the core Go API's internal (service-to-service) routes."""

    def __init__(self, base_url: str, internal_key: str):
        self._client = httpx.AsyncClient(
            base_url=base_url,
            headers={"X-Internal-Key": internal_key, "Content-Type": "application/json"},
            timeout=15.0,
        )

    async def close(self):
        await self._client.aclose()

    async def _request(self, method: str, path: str, **kwargs) -> Optional[dict]:
        resp = await self._client.request(method, path, **kwargs)
        if resp.status_code >= 400:
            logger.error("API %s %s failed: %s %s", method, path, resp.status_code, resp.text)
            resp.raise_for_status()
        if resp.status_code == 204 or not resp.content:
            return None
        return resp.json()

    async def list_source_channels(self, active_only: bool = True) -> list[dict]:
        result = await self._request(
            "GET", "/api/v1/source-channels", params={"active_only": str(active_only).lower()}
        )
        return result or []

    async def update_source_channel_cursor(self, channel_id: int, last_message_id: int):
        await self._request(
            "POST",
            f"/api/v1/source-channels/{channel_id}/cursor",
            json={"last_synced_message_id": last_message_id},
        )

    async def upsert_anime(self, payload: dict[str, Any]) -> dict:
        return await self._request("POST", "/api/v1/anime", json=payload)

    async def upsert_episode(self, payload: dict[str, Any]) -> dict:
        return await self._request("POST", "/api/v1/episodes", json=payload)

    async def delete_episode_by_source(self, source_channel_id: int, source_message_id: int):
        await self._request(
            "DELETE",
            "/api/v1/episodes",
            json={"source_channel_id": source_channel_id, "source_message_id": source_message_id},
        )

    async def create_import_log(
        self,
        source_channel_id: Optional[int],
        telegram_message_id: int,
        action: str,
        detail: str = "",
        anime_id: Optional[int] = None,
        episode_id: Optional[int] = None,
    ):
        try:
            await self._request(
                "POST",
                "/api/v1/import-logs",
                json={
                    "source_channel_id": source_channel_id,
                    "telegram_message_id": telegram_message_id,
                    "action": action,
                    "detail": detail,
                    "anime_id": anime_id,
                    "episode_id": episode_id,
                },
            )
        except Exception:
            logger.exception("failed to write import log (non-fatal)")
