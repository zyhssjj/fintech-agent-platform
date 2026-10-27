from __future__ import annotations

from typing import Any

import httpx

from app.config import SETTINGS


class OpenAICompatibleClient:
    def __init__(self) -> None:
        self._base_url = SETTINGS.base_url.rstrip("/")
        self._api_key = SETTINGS.api_key
        self._model = SETTINGS.model
        self._timeout = SETTINGS.timeout_ms / 1000

    def is_configured(self) -> bool:
        return bool(self._base_url and self._api_key and self._model)

    async def chat(self, system_prompt: str, user_prompt: str) -> str:
        if not self.is_configured():
            raise RuntimeError("AI service is not configured. Please fill ai/.env first.")

        payload: dict[str, Any] = {
            "model": self._model,
            "messages": [
                {"role": "system", "content": system_prompt},
                {"role": "user", "content": user_prompt},
            ],
            "temperature": 0.2,
        }
        headers = {
            "Authorization": f"Bearer {self._api_key}",
            "Content-Type": "application/json",
        }
        async with httpx.AsyncClient(timeout=self._timeout) as client:
            response = await client.post(f"{self._base_url}/chat/completions", json=payload, headers=headers)
            response.raise_for_status()
            data = response.json()

        try:
            return (data["choices"][0]["message"]["content"] or "").strip()
        except (KeyError, IndexError, TypeError) as exc:
            raise RuntimeError("AI upstream returned an unexpected response.") from exc

