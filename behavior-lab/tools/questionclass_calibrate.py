"""Measures questionclass-charcnn-r0 and emits a CalibrationProfile JSON in
the exact schema of tlaloc/behavior-lab/internal/tlaloque/calibration
(tlaloc.calibration-profile.r0). This produces *evidence of competence*,
separate from softmax confidence — the profile is what the swarm consults
before trusting the model, and what the admission gate checks.

Usage:
    python3 questionclass_calibrate.py \
        --checkpoint ../models/questionclass-charcnn-r0.pt \
        --in-dist /tmp/qclass-test.jsonl \
        --ood /tmp/qclass-ood.jsonl \
        --out ../models/questionclass-charcnn-r0.calibration.json
"""
import argparse
import datetime
import hashlib
import json

import torch

from questionclass_model import QuestionTypeClassifier, predict_text

SCHEMA = "tlaloc.calibration-profile.r0"
ECE_BINS = 10
THRESHOLDS = [0.5, 0.6, 0.7, 0.8, 0.9, 0.95]


def load_rows(path: str) -> list[dict]:
    return [json.loads(line) for line in open(path)]


def set_id(path: str) -> str:
    return hashlib.sha256(open(path, "rb").read()).hexdigest()[:16]


def score(model, rows: list[dict]) -> list[tuple[float, bool]]:
    """Returns (confidence, correct) per row. The model never abstains on
    its own here — abstention is a downstream policy, not a model output."""
    out = []
    for row in rows:
        prediction = predict_text(model, row["text"])
        out.append((prediction["confidence"], prediction["type"] == row["label"]))
    return out


def accuracy(scored: list[tuple[float, bool]]) -> float:
    return sum(correct for _, correct in scored) / len(scored) if scored else 0.0


def brier(scored: list[tuple[float, bool]]) -> float:
    if not scored:
        return 0.0
    return sum((conf - (1.0 if correct else 0.0)) ** 2 for conf, correct in scored) / len(scored)


def ece(scored: list[tuple[float, bool]], bins: int) -> float:
    if not scored:
        return 0.0
    buckets = [[0, 0.0, 0] for _ in range(bins)]  # count, confidence_sum, correct
    for conf, correct in scored:
        index = min(int(conf * bins), bins - 1)
        buckets[index][0] += 1
        buckets[index][1] += conf
        buckets[index][2] += 1 if correct else 0
    total = len(scored)
    result = 0.0
    for count, conf_sum, correct in buckets:
        if count == 0:
            continue
        gap = abs(conf_sum / count - correct / count)
        result += (count / total) * gap
    return result


def eval_slice(scored: list[tuple[float, bool]]) -> dict:
    return {
        "n": len(scored),
        "accuracy": round(accuracy(scored), 4),
        "coverage": 1.0,  # the model itself does not abstain during measurement
        "ece": round(ece(scored, ECE_BINS), 4),
        "brier": round(brier(scored), 4),
    }


def abstention_curve(scored: list[tuple[float, bool]]) -> list[dict]:
    total = len(scored)
    curve = []
    for threshold in THRESHOLDS:
        covered = [correct for conf, correct in scored if conf >= threshold]
        curve.append({
            "threshold": threshold,
            "coverage": round(len(covered) / total, 4) if total else 0.0,
            "covered_accuracy": round(sum(covered) / len(covered), 4) if covered else 0.0,
        })
    return curve


def pick_confidence_floor(ood_curve: list[dict]) -> float:
    """Lowest threshold whose OOD covered accuracy clears 0.85. If no
    threshold does, the model has no confidence level at which it can be
    trusted on unfamiliar input: return an unreachable floor (1.01) so the
    runtime policy always abstains and the admission gate always refuses."""
    for point in ood_curve:
        if point["covered_accuracy"] >= 0.85:
            return point["threshold"]
    return 1.01


def main() -> int:
    parser = argparse.ArgumentParser()
    parser.add_argument("--checkpoint", required=True)
    parser.add_argument("--in-dist", required=True)
    parser.add_argument("--ood", required=True)
    parser.add_argument("--out", required=True)
    parser.add_argument("--worker-id", default="questionclass-charcnn-r0")
    parser.add_argument("--model-version", default="r0")
    parser.add_argument("--training-distribution-id", default="questionclass-synthetic-templates-v2")
    args = parser.parse_args()

    model = QuestionTypeClassifier()
    model.load_state_dict(torch.load(args.checkpoint))
    model.eval()

    in_scored = score(model, load_rows(args.in_dist))
    ood_scored = score(model, load_rows(args.ood))
    ood_curve = abstention_curve(ood_scored)

    profile = {
        "schema": SCHEMA,
        "worker_id": args.worker_id,
        "model_version": args.model_version,
        "training_distribution_id": args.training_distribution_id,
        "calibration_set_id": f"in:{set_id(args.in_dist)}|ood:{set_id(args.ood)}",
        "in_distribution": eval_slice(in_scored),
        "out_of_distribution": eval_slice(ood_scored),
        "abstention_curve": ood_curve,
        "supported_domains": [],
        "unsupported_domains": [],
        "confidence_floor": pick_confidence_floor(ood_curve),
        "generated_at": datetime.datetime.now(datetime.timezone.utc).isoformat(),
    }

    with open(args.out, "w") as handle:
        json.dump(profile, handle, indent=2)
    print(json.dumps(profile, indent=2))
    print(f"\nwrote {args.out}")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
