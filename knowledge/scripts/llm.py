"""Cloud LLM client (OpenAI-compatible endpoint, e.g. DashScope qwen)."""

from __future__ import annotations

import json
import os
import re
from pathlib import Path


class LLMError(RuntimeError):
    pass


def _load_dotenv() -> None:
    try:
        from dotenv import load_dotenv
    except ImportError:
        return
    load_dotenv(Path(__file__).resolve().parents[1] / ".env", override=False)


def get_client():
    from openai import OpenAI

    from scripts import model_config

    cfg = model_config.get_config("llm")
    base_url = cfg.get("base_url", "")
    api_key = cfg.get("api_key", "")
    if not api_key or not base_url:
        raise LLMError("LLM 未配置（可在模型配置页设置）")
    return OpenAI(api_key=api_key, base_url=base_url.rstrip("/"))


def chat(system: str, user: str, *, model: str | None = None, temperature: float = 0.2,
         json_mode: bool = False) -> str:
    from scripts import model_config

    client = get_client()
    default_model = model_config.get_config("llm").get("model", "qwen-max")
    kwargs: dict = dict(
        model=model or default_model,
        messages=[
            {"role": "system", "content": system},
            {"role": "user", "content": user},
        ],
        temperature=temperature,
    )
    if json_mode:
        kwargs["response_format"] = {"type": "json_object"}
    response = client.chat.completions.create(**kwargs)
    content = response.choices[0].message.content
    return (content or "").strip()


def extract_json(text: str):
    """Extract a JSON object from LLM output (tolerates markdown fences)."""
    value = (text or "").strip()
    if value.startswith("```"):
        value = re.sub(r"^```(?:json)?\s*", "", value, flags=re.IGNORECASE)
        value = re.sub(r"\s*```$", "", value)
    start = value.find("{")
    end = value.rfind("}")
    if start < 0 or end <= start:
        raise LLMError("LLM 未返回有效 JSON: " + value[:200])
    try:
        return json.loads(value[start : end + 1])
    except json.JSONDecodeError as error:
        raise LLMError("LLM 返回的 JSON 无法解析: " + value[:200]) from error


def chat_json(system: str, user: str, *, model: str | None = None):
    return extract_json(chat(system, user, model=model, json_mode=True))
