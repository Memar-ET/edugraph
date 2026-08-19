#!/usr/bin/env python3
"""
Script to pre-download and verify the intfloat/multilingual-e5-large embedding model.
Usage:
    python scripts/download_embedding_model.py
"""

import sys
import structlog
from sentence_transformers import SentenceTransformer

logger = structlog.get_logger()

MODEL_NAME = "intfloat/multilingual-e5-large"


def main():
    print(f"[*] Downloading/loading model: {MODEL_NAME} (~2.3-2.5 GB)...")
    try:
        model = SentenceTransformer(MODEL_NAME)
        test_text = "passage: EduGraph AI embedding model verification"
        embedding = model.encode(test_text, normalize_embeddings=True)
        dim = len(embedding)
        print(f"[+] Success! Model loaded into memory.")
        print(f"[+] Vector dimension: {dim} (Expected: 1024)")
        if dim != 1024:
            print(f"[!] Warning: Expected 1024 dimensions but got {dim}", file=sys.stderr)
            sys.exit(1)
        print("[+] Model verification complete.")
    except Exception as e:
        print(f"[!] Failed to load model {MODEL_NAME}: {e}", file=sys.stderr)
        sys.exit(1)


if __name__ == "__main__":
    main()
