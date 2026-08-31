#!/usr/bin/env python3
import argparse
import base64
import hashlib
import json
import os
from http.server import BaseHTTPRequestHandler, ThreadingHTTPServer

BASELINE_SHA = os.environ.get("BASELINE_SHA256", "").strip().lower()


def answer_for(question: str, baseline: bool, diagnostic: bool) -> str:
    q = question.lower()
    if "box" in q and "arrow" in q:
        return "BOX means CELL, ARROW means TRANSITION, RING means CHECKPOINT, and X/TIME means temporal position."
    if "cells or agents" in q:
        return "A, B, C"
    if "initial state" in q:
        return "A is ACTIVE."
    if "causes b" in q:
        if baseline:
            answer = "I cannot locate the declared transition."
            if diagnostic:
                trace = {
                    "schema": "tlaloc.origami-debug-trace.r0",
                    "status": "FAIL",
                    "last_completed_stage": "ROSETTA",
                    "selected_codec": "ST2",
                    "last_instruction": "READ_ROSETTA",
                    "next_instruction": "LOCATE_T2",
                    "failure_code": "T2_NOT_FOUND",
                    "evidence_refs": ["T0", "T1"],
                    "confidence": 0.9,
                }
                answer += "\nORIGAMI_DEBUG_R0=" + json.dumps(trace, separators=(",", ":"))
            return answer
        return "A ACTIVE causes B to become ACTIVE."
    if "after b" in q:
        return "After B becomes ACTIVE, A becomes DONE and C becomes ACTIVE."
    if "checkpoint" in q:
        return "T0 T2 T4 T6 T8"
    if "literal video" in q:
        return "No. It is not a literal video frame sequence; it is a declared semantic temporal program."
    if "final states" in q:
        return "A DONE, B DONE, C ACTIVE."
    if "sha-256" in q:
        return "NOT_VERIFIED: no mechanical exact decoder is available to this model."
    return "UNKNOWN"


def extract_request(payload):
    messages = payload.get("messages") or []
    system = ""
    question = ""
    image_bytes = b""
    for message in messages:
        role = message.get("role")
        content = message.get("content")
        if role == "system" and isinstance(content, str):
            system = content
        if role != "user" or not isinstance(content, list):
            continue
        for part in content:
            if part.get("type") == "text":
                question = str(part.get("text", ""))
            if part.get("type") == "image_url":
                url = (part.get("image_url") or {}).get("url", "")
                marker = ";base64,"
                if marker in url:
                    image_bytes = base64.b64decode(url.split(marker, 1)[1])
    return system, question, image_bytes


class Handler(BaseHTTPRequestHandler):
    def log_message(self, fmt, *args):
        return

    def do_GET(self):
        if self.path == "/health":
            body = b'{"status":"ok"}\n'
            self.send_response(200)
            self.send_header("Content-Type", "application/json")
            self.send_header("Content-Length", str(len(body)))
            self.end_headers()
            self.wfile.write(body)
            return
        self.send_error(404)

    def do_POST(self):
        if self.path != "/v1/chat/completions":
            self.send_error(404)
            return
        try:
            length = int(self.headers.get("Content-Length", "0"))
            payload = json.loads(self.rfile.read(length))
            system, question, image_bytes = extract_request(payload)
            image_sha = hashlib.sha256(image_bytes).hexdigest()
            baseline = bool(BASELINE_SHA) and image_sha == BASELINE_SHA
            diagnostic = "DIAGNOSTIC MODE" in system.upper()
            answer = answer_for(question, baseline, diagnostic)
            response = {
                "choices": [{"message": {"content": answer}}],
                "usage": {"prompt_tokens": 0, "completion_tokens": 0},
            }
            body = (json.dumps(response) + "\n").encode()
            self.send_response(200)
            self.send_header("Content-Type", "application/json")
            self.send_header("Content-Length", str(len(body)))
            self.end_headers()
            self.wfile.write(body)
        except Exception as exc:
            body = (json.dumps({"error": str(exc)}) + "\n").encode()
            self.send_response(500)
            self.send_header("Content-Type", "application/json")
            self.send_header("Content-Length", str(len(body)))
            self.end_headers()
            self.wfile.write(body)


def main():
    parser = argparse.ArgumentParser()
    parser.add_argument("--host", default="127.0.0.1")
    parser.add_argument("--port", type=int, default=18765)
    args = parser.parse_args()
    if not BASELINE_SHA:
        raise SystemExit("BASELINE_SHA256 is required")
    server = ThreadingHTTPServer((args.host, args.port), Handler)
    server.serve_forever()


if __name__ == "__main__":
    main()
