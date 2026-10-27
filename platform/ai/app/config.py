from __future__ import annotations

import os
from dataclasses import dataclass
from pathlib import Path

from dotenv import load_dotenv


ROOT_DIR = Path(__file__).resolve().parent.parent
ENV_FILE = ROOT_DIR / ".env"

load_dotenv(ENV_FILE)


@dataclass(frozen=True)
class Settings:
    provider: str
    base_url: str
    api_key: str
    model: str
    timeout_ms: int
    host: str
    port: int


def load_settings() -> Settings:
    return Settings(
        provider=os.getenv("AI_PROVIDER", "openai").strip() or "openai",
        base_url=os.getenv("AI_BASE_URL", "").strip(),
        api_key=os.getenv("AI_API_KEY", "").strip(),
        model=os.getenv("AI_MODEL", "deepseek-v4-flash").strip() or "deepseek-v4-flash",
        timeout_ms=int(os.getenv("AI_TIMEOUT_MS", "60000").strip() or "60000"),
        host=os.getenv("AI_HOST", "0.0.0.0").strip() or "0.0.0.0",
        port=int(os.getenv("AI_PORT", "9100").strip() or "9100"),
    )


SETTINGS = load_settings()
