"""Measures groundingscore-distilled-r0 and emits a CalibrationProfile JSON in
the schema of internal/tlaloque/calibration (tlaloc.calibration-profile.r0),
same as questionclass_calibrate.py.

For a regressor, a prediction counts as "correct" when it lands within
--tolerance of the reference score. "confidence" is the model's own
self-estimated confidence output. The profile is what the swarm consolidator
consults before trusting the distilled score instead of falling back to a
heavier judge.

Usage:
    python3 grounding_calibrate.py \
        --checkpoint ../models/groundingscore-distilled-r0.pt \
        --in-dist /tmp/grounding-val.jsonl \
        --ood /tmp/grounding-ood.jsonl \
        --out ../models/groundingscore-distilled-r0.calibration.json
"""
from __future__ import annotations

import argparse
import datetime
import hashlib
import json

import numpy as np

from grounding_common import Embedder, extract_features
from grounding_model import GroundingScorer, predict
from grounding_serve import load_checkpoint

SCHEMA = "tlaloc.calibration-profile.r0"
ECE_BINS = 10
THRESHOLDS = [0.5, 0.6, 0.7, 0.8, 0.9, 0.95]


def set_id(path: str) -> str:
    return hashlib.sha256(open(path, "rb").read()).hexdigest()[:16]


def score_in_dist(model, mean, std, path: str, tolerance: float) -> list[tuple[float, bool]]:
    out = []
    for line in open(path):
        row = json.loads(line)
        result = predict(model, row["features"], mean, std)
        out.append((result["confidence"], abs(result["score"] - row["judge_score"]) <= tolerance))
    return out


def score_ood(model, mean, std, path: str, tolerance: float,
              embedder: Embedder) -> list[tuple[float, bool]]:
    out = []
    for line in open(path):
        row = json.loads(line)
        features = extract_features(embedder, row["question"], row["answer"], row["passage"])
        result = predict(model, features, mean, std)
        out.append((result["confidence"], abs(result["score"] - row["label"]) <= tolerance))
    return out


def accuracy(scored):
    return sum(correct for _, correct in scored) / len(scored) if scored else 0.0


def brier(scored):
    if not scored:
        return 0.0
    return sum((conf - (1.0 if ok else 0.0)) ** 2 for conf, ok in scored) / len(scored)


def ece(scored, bins):
    if not scored:
        return 0.0
    buckets = [[0, 0.0, 0] for _ in range(bins)]
    for conf, ok in scored:
        index = min(int(conf * bins), bins - 1)
        buckets[index][0] += 1
        buckets[index][1] += conf
        buckets[index][2] += 1 if ok else 0
    total = len(scored)
    result = 0.0
    for count, conf_sum, correct in buckets:
        if count:
            result += (count / total) * abs(conf_sum / count - correct / count)
    return result


def eval_slice(scored):
    return {
        "n": len(scored),
        "accuracy": round(accuracy(scored), 4),
        "coverage": 1.0,
        "ece": round(ece(scored, ECE_BINS), 4),
        "brier": round(brier(scored), 4),
    }


def abstention_curve(scored):
    total = len(scored)
    curve = []
    for threshold in THRESHOLDS:
        covered = [ok for conf, ok in scored if conf >= threshold]
        curve.append({
            "threshold": threshold,
            "coverage": round(len(covered) / total, 4) if total else 0.0,
            "covered_accuracy": round(sum(covered) / len(covered), 4) if covered else 0.0,
        })
    return curve


def pick_confidence_floor(ood_curve, target: float) -> float:
    for point in ood_curve:
        if point["covered_accuracy"] >= target:
            return point["threshold"]
    return 1.01


def main() -> int:
    parser = argparse.ArgumentParser()
    parser.add_argument("--checkpoint", required=True)
    parser.add_argument("--in-dist", required=True)
    parser.add_argument("--ood", required=True)
    parser.add_argument("--out", required=True)
    parser.add_argument("--base-url", default="http://127.0.0.1:1234/v1")
    parser.add_argument("--tolerance", type=float, default=0.2)
    parser.add_argument("--target-accuracy", type=float, default=0.8)
    parser.add_argument("--worker-id", default="groundingscore-distilled-r0")
    parser.add_argument("--training-distribution-id", default="grounding-foldbench-h2o-danube-judge-v0")
    args = parser.parse_args()

    model, mean, std = load_checkpoint(args.checkpoint)
    embedder = Embedder(base_url=args.base_url)

    in_scored = score_in_dist(model, mean, std, args.in_dist, args.tolerance)
    ood_scored = score_ood(model, mean, std, args.ood, args.tolerance, embedder)
    ood_curve = abstention_curve(ood_scored)

    profile = {
        "schema": SCHEMA,
        "worker_id": args.worker_id,
        "model_version": "r0",
        "training_distribution_id": args.training_distribution_id,
        "calibration_set_id": f"in:{set_id(args.in_dist)}|ood:{set_id(args.ood)}",
        "tolerance": args.tolerance,
        "in_distribution": eval_slice(in_scored),
        "out_of_distribution": eval_slice(ood_scored),
        "abstention_curve": ood_curve,
        "supported_domains": [],
        "unsupported_domains": [],
        "confidence_floor": pick_confidence_floor(ood_curve, args.target_accuracy),
        "generated_at": datetime.datetime.now(datetime.timezone.utc).isoformat(),
    }
    with open(args.out, "w") as handle:
        json.dump(profile, handle, indent=2)
    print(json.dumps(profile, indent=2))
    print(f"\nwrote {args.out}")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
