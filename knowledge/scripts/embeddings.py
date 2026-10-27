"""Cloud embedding provider (OpenAI-compatible endpoint, e.g. DashScope text-embedding-v4).

Keeps the embedding backend behind a small interface so a local model can be
swapped in later without touching the ingestion/retrieval code.
"""

from __future__ import annotations

import os
from pathlib import Path
from typing import Sequence


class EmbeddingProviderError(RuntimeError):
    pass


class CloudEmbeddingProvider:
    def __init__(
        self,
        *,
        base_url: str,
        api_key: str,
        model: str = "text-embedding-v4",
        dimension: int = 1024,
    ) -> None:
        if not api_key or not api_key.strip():
            raise EmbeddingProviderError("EMBEDDING_API_KEY is not configured.")
        if not base_url or not base_url.strip():
            raise EmbeddingProviderError("EMBEDDING_BASE_URL is not configured.")
        self.model = model
        self.dimension = dimension
        from openai import OpenAI

        self._client = OpenAI(api_key=api_key, base_url=base_url.rstrip("/"))

    def embed_texts(self, texts: Sequence[str]) -> list[list[float]]:
        inputs = [text for text in texts]
        if not inputs:
            return []
        if any(not isinstance(text, str) or not text.strip() for text in inputs):
            raise EmbeddingProviderError("Embedding inputs must be non-empty strings.")

        response = self._client.embeddings.create(
            model=self.model,
            input=inputs,
            dimensions=self.dimension,
        )
        data = list(response.data)
        if len(data) != len(inputs):
            raise EmbeddingProviderError(
                "Embedding response count mismatch: expected {0}, got {1}".format(len(inputs), len(data))
            )
        # DashScope may return items out of order; reorder by index.
        data = sorted(data, key=lambda item: getattr(item, "index", 0))
        vectors: list[list[float]] = []
        for item in data:
            embedding = getattr(item, "embedding", None)
            if not isinstance(embedding, list) or not embedding:
                raise EmbeddingProviderError("Embedding response item has no vector.")
            vector = [float(value) for value in embedding]
            if len(vector) != self.dimension:
                raise EmbeddingProviderError(
                    "Embedding dimension mismatch: expected {0}, got {1}".format(self.dimension, len(vector))
                )
            vectors.append(vector)
        return vectors

    def embed_text(self, text: str) -> list[float]:
        return self.embed_texts([text])[0]


def configured_provider() -> CloudEmbeddingProvider:
    from scripts import model_config

    cfg = model_config.get_config("embedding")
    return CloudEmbeddingProvider(
        base_url=cfg.get("base_url", ""),
        api_key=cfg.get("api_key", ""),
        model=cfg.get("model", "text-embedding-v4"),
        dimension=int(cfg.get("dimension", 1024)),
    )


def _load_dotenv() -> None:
    try:
        from dotenv import load_dotenv
    except ImportError:
        return
    project_root = Path(__file__).resolve().parents[1]
    load_dotenv(project_root / ".env", override=False)
