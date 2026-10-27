"""Download the local embedding and reranking models into the project.

The model weights are intentionally stored under MODELS_DIR (default: models/)
instead of the user-level Hugging Face cache.  This makes the project bundle
portable to the target server.
"""

import argparse
import os
from pathlib import Path

from huggingface_hub import snapshot_download
from sentence_transformers import CrossEncoder, SentenceTransformer


MODELS_DIR = Path(os.getenv("MODELS_DIR", "models"))
MODELS = {
    "bge-m3": "BAAI/bge-m3",
    "bge-reranker-v2-m3": "BAAI/bge-reranker-v2-m3",
}


def download_and_validate(directory_name, repo_id):
    destination = MODELS_DIR / directory_name
    print("downloading {0} -> {1}".format(repo_id, destination))
    snapshot_download(repo_id=repo_id, local_dir=str(destination))
    if directory_name == "bge-m3":
        model = SentenceTransformer(str(destination))
        if model.get_sentence_embedding_dimension() != 1024:
            raise RuntimeError("bge-m3 vector dimension is not 1024")
    else:
        CrossEncoder(str(destination))
    print("ready: {0}".format(destination))


def main():
    parser = argparse.ArgumentParser()
    parser.add_argument("--only", choices=tuple(MODELS))
    args = parser.parse_args()
    MODELS_DIR.mkdir(parents=True, exist_ok=True)
    selected_models = (
        {args.only: MODELS[args.only]} if args.only else MODELS
    )
    for directory_name, repo_id in selected_models.items():
        download_and_validate(directory_name, repo_id)
    print("models ready")


if __name__ == "__main__":
    main()
