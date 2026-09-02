"""Resident inference service for groundingscore-distilled-r0. Speaks the
tlaloque.CapabilityRequest/CapabilityResponse HTTP_JSON contract unmodified
(same shape as questionclass_serve.py and cmd/tlaloc-embedding-scorer).

The heavy part (MiniLM-L6 embeddings) is delegated to the LM Studio instance
this service points at; the trained head is a few hundred parameters and runs
in microseconds on CPU.

Input  (CapabilityRequest.input): {question, model_answer, page_content}
Output (CapabilityResponse.output): {score, confidence, keywords_matched,
       keywords_total, notes} — the answerscore.ScoreOutput shape, so this
       worker drops straight into answerscore.ScoreAnswer's candidate chain.

Usage:
    python3 grounding_serve.py --checkpoint ../models/groundingscore-distilled-r0.pt \
        --lm-studio http://127.0.0.1:1234/v1 --addr :8793
"""
from __future__ import annotations

import argparse
import json
from http.server import BaseHTTPRequestHandler, ThreadingHTTPServer

import numpy as np
import torch

from grounding_common import Embedder, extract_features
from grounding_model import GroundingScorer, count_parameters, predict

WORKER_ID = "groundingscore-distilled-r0"


def load_checkpoint(path: str) -> tuple[GroundingScorer, np.ndarray, np.ndarray]:
    blob = torch.load(path, map_location="cpu")
    model = GroundingScorer()
    model.load_state_dict(blob["state_dict"])
    model.eval()
    return model, np.asarray(blob["mean"], dtype="float32"), np.asarray(blob["std"], dtype="float32")


def make_handler(model, mean, std, embedder: Embedder, checkpoint_path: str):
    class Handler(BaseHTTPRequestHandler):
        def _send_json(self, status: int, payload: dict) -> None:
            body = json.dumps(payload).encode("utf-8")
            self.send_response(status)
            self.send_header("Content-Type", "application/json")
            self.send_header("Content-Length", str(len(body)))
            self.end_headers()
            self.wfile.write(body)

        def do_GET(self) -> None:
            if self.path != "/health":
                self._send_json(404, {"error": "not found"})
                return
            self._send_json(200, {
                "status": "ok", "worker_id": WORKER_ID,
                "checkpoint": checkpoint_path, "parameters": count_parameters(model),
            })

        def do_POST(self) -> None:
            if self.path != "/execute":
                self._send_json(404, {"error": "not found"})
                return
            length = int(self.headers.get("Content-Length", "0"))
            try:
                envelope = json.loads(self.rfile.read(length))
                payload = envelope["input"]
                question = str(payload["question"])
                answer = str(payload["model_answer"])
                passage = str(payload["page_content"])
                if not answer.strip() or not passage.strip():
                    raise ValueError("model_answer and page_content must be non-empty")
            except Exception as exc:  # noqa: BLE001 - reported to the caller
                self._send_json(400, {"error": f"invalid request: {exc}"})
                return

            try:
                features = extract_features(embedder, question, answer, passage)
                result = predict(model, features, mean, std)
            except Exception as exc:  # noqa: BLE001 - embedding backend down, etc.
                self._send_json(502, {"error": f"scoring failed: {exc}"})
                return

            self._send_json(200, {
                "worker_id": WORKER_ID,
                "output": {
                    "score": round(result["score"], 4),
                    "confidence": round(result["confidence"], 4),
                    "keywords_matched": 0,
                    "keywords_total": 0,
                    "notes": "distilled grounding scorer (MiniLM features + trained MLP)",
                },
                "confidence": round(result["confidence"], 4),
            })

        def log_message(self, format: str, *fmt_args) -> None:  # noqa: A002 - stdlib signature
            print(f"[grounding_serve] {self.address_string()} {format % fmt_args}")

    return Handler


def main() -> int:
    parser = argparse.ArgumentParser()
    parser.add_argument("--checkpoint", required=True)
    parser.add_argument("--lm-studio", default="http://127.0.0.1:1234/v1")
    parser.add_argument("--addr", default=":8793")
    args = parser.parse_args()

    model, mean, std = load_checkpoint(args.checkpoint)
    embedder = Embedder(base_url=args.lm_studio)
    print(f"loaded {args.checkpoint} ({count_parameters(model):,} parameters)")

    host, _, port = args.addr.rpartition(":")
    server = ThreadingHTTPServer((host, int(port)), make_handler(model, mean, std, embedder, args.checkpoint))
    print(f"grounding_serve listening on {args.addr}")
    server.serve_forever()
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
