"""Create and validate Elasticsearch indices for the fintech knowledge base.

Run after Elasticsearch is healthy:
    python scripts/create_indices.py

The command is idempotent: it creates only absent indices and validates the
vector dimension of existing ones, preventing accidental use of a different
embedding model with the same index.

Indices:
    kb_finance_doc - 科技金融企业长文档（语义切块）
"""

import os
import sys

from elasticsearch import Elasticsearch
from elasticsearch import exceptions as es_exceptions


ES_URL = os.getenv("ES_URL", "http://localhost:9202")
VECTOR_DIMS = int(os.getenv("EMBEDDING_DIMENSION", "1024"))  # 与 embedding 模型维度一致

INDEX_NAMES = ("kb_finance_doc",)


def _text_field(*, with_keyword=True):
    field = {
        "type": "text",
        "analyzer": "ik_max_word",
        "search_analyzer": "ik_smart",
    }
    if with_keyword:
        field["fields"] = {"keyword": {"type": "keyword", "ignore_above": 512}}
    return field


def _vector_field():
    return {
        "type": "dense_vector",
        "dims": VECTOR_DIMS,
        "index": True,
        "similarity": "cosine",
    }


FINANCE_PROPERTIES = {
    "doc_id": {"type": "keyword"},
    "document_id": {"type": "keyword"},
    "title": _text_field(),
    "source": {"type": "keyword"},
    "source_file": {"type": "keyword"},
    "document_type": {"type": "keyword"},
    "chunk_id": {"type": "keyword"},
    "chunk_index": {"type": "integer"},
    # 语义切块时 LLM 产出的小标题与关键词
    "chunk_title": _text_field(),
    "section_title": _text_field(),
    "section_key": {"type": "keyword"},
    "table_header": _text_field(),
    "table_index": {"type": "integer"},
    "table_row_start": {"type": "integer"},
    "table_row_end": {"type": "integer"},
    "document_table_count": {"type": "integer"},
    "document_table_row_count": {"type": "integer"},
    "is_structured": {"type": "boolean"},
    "keywords": {"type": "keyword"},
    "heading_path": {"type": "keyword"},
    "content": _text_field(),
    "content_vector": _vector_field(),
    "company_code": {"type": "keyword"},
    "domain_path": {"type": "keyword"},
    "domain_level1": {"type": "keyword"},
    "domain_level2": {"type": "keyword"},
    "domain_level3": {"type": "keyword"},
    "domain_leaf_id": {"type": "keyword"},
    "catalog_version": {"type": "integer"},
    "document_version": {"type": "keyword"},
    "acl": {"type": "keyword"},
    "content_hash": {"type": "keyword"},
    "ingested_at": {"type": "date"},
    "parser": {"type": "keyword"},
}


INDEX_DEFINITIONS = {
    "kb_finance_doc": FINANCE_PROPERTIES,
}


def index_definition(properties):
    return {
        "settings": {"number_of_shards": 1, "number_of_replicas": 0},
        "mappings": {"dynamic": "strict", "properties": properties},
    }


def validate_existing_index(client, index_name, properties):
    mapping = client.indices.get_mapping(index=index_name)[index_name]["mappings"]
    existing = mapping.get("properties", {})
    vector_field = "content_vector"
    vector = existing.get(vector_field, {})
    actual_dims = vector.get("dims")
    if vector.get("type") != "dense_vector" or actual_dims != VECTOR_DIMS:
        raise RuntimeError(
            "{0} has incompatible {1} mapping (expected dense_vector/{2}, found {3}/{4}). "
            "Recreate it before changing embedding models.".format(
                index_name, vector_field, VECTOR_DIMS, vector.get("type"), actual_dims
            )
        )
    missing = {name: definition for name, definition in properties.items() if name not in existing}
    if missing:
        client.indices.put_mapping(index=index_name, properties=missing)
        print("updated mapping for {0}: {1}".format(index_name, ", ".join(missing)))


def main():
    client = Elasticsearch(ES_URL)
    try:
        if not client.ping():
            raise RuntimeError("Elasticsearch did not respond to ping")
    except es_exceptions.ConnectionError as error:
        raise RuntimeError(
            "Cannot connect to Elasticsearch at {0}. Start the container first.".format(ES_URL)
        ) from error

    for index_name in INDEX_NAMES:
        if client.indices.exists(index=index_name):
            validate_existing_index(client, index_name, INDEX_DEFINITIONS[index_name])
            print("validated {0}".format(index_name))
            continue
        client.indices.create(index=index_name, **index_definition(INDEX_DEFINITIONS[index_name]))
        print("created {0}".format(index_name))


if __name__ == "__main__":
    try:
        main()
    except (RuntimeError, es_exceptions.ApiError, es_exceptions.ConnectionError) as error:
        print("index setup failed: {0}".format(error), file=sys.stderr)
        sys.exit(1)
