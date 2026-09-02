"""Resident inference service for questionclass-charcnn-r0. Speaks the
tlaloque.CapabilityRequest/CapabilityResponse HTTP_JSON contract (see
tlaloc/behavior-lab/internal/tlaloque/http_worker.go) unmodified — same
shape as origami/tools/microisa_serve.py and cmd/tlaloc-embedding-scorer:
a micro-model loaded once, called many times.

Usage:
    python3 questionclass_serve.py --checkpoint ../models/questionclass-charcnn-r0.pt --addr :8792
"""
import argparse
import json
from http.server import BaseHTTPRequestHandler, ThreadingHTTPServer

import torch

from questionclass_model import LABELS, QuestionTypeClassifier, count_parameters, predict_text

WORKER_ID = "questionclass-charcnn-r0"


def make_handler(model: QuestionTypeClassifier, checkpoint_path: str):
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
                "status": "ok",
                "worker_id": WORKER_ID,
                "checkpoint": checkpoint_path,
                "parameters": count_parameters(model),
                "labels": list(LABELS),
            })

        def do_POST(self) -> None:
            if self.path != "/execute":
                self._send_json(404, {"error": "not found"})
                return
            length = int(self.headers.get("Content-Length", "0"))
            try:
                envelope = json.loads(self.rfile.read(length))
                question = envelope["input"]["question"]
                if not isinstance(question, str) or not question.strip():
                    raise ValueError("input.question must be a non-empty string")
            except Exception as exc:  # noqa: BLE001 - reported to the caller, not swallowed
                self._send_json(400, {"error": f"invalid request: {exc}"})
                return

            prediction = predict_text(model, question)
            self._send_json(200, {
                "worker_id": WORKER_ID,
                "output": {"type": prediction["type"]},
                "confidence": prediction["confidence"],
            })

        def log_message(self, format: str, *fmt_args) -> None:  # noqa: A002 - stdlib signature
            print(f"[questionclass_serve] {self.address_string()} {format % fmt_args}")

    return Handler


def main() -> int:
    parser = argparse.ArgumentParser()
    parser.add_argument("--checkpoint", required=True)
    parser.add_argument("--addr", default=":8792", help="host:port, or :port for all interfaces")
    args = parser.parse_args()

    model = QuestionTypeClassifier()
    model.load_state_dict(torch.load(args.checkpoint))
    model.eval()
    print(f"loaded {args.checkpoint} ({count_parameters(model):,} parameters)")

    host, _, port = args.addr.rpartition(":")
    server = ThreadingHTTPServer((host, int(port)), make_handler(model, args.checkpoint))
    print(f"questionclass_serve listening on {args.addr}")
    server.serve_forever()
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
