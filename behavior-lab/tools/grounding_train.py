"""Trains groundingscore-distilled-r0 on JSONL from grounding_dataset.py
(features are precomputed there, so this is a pure CPU regression fit in
seconds).

Usage:
    python3 grounding_train.py --train /tmp/grounding-train.jsonl \
        --val /tmp/grounding-val.jsonl --out ../models/groundingscore-distilled-r0.pt \
        --metrics-out ../models/groundingscore-distilled-r0.metrics.json --epochs 120
"""
from __future__ import annotations

import argparse
import json

import numpy as np
import torch
import torch.nn as nn

from grounding_model import GroundingScorer, count_parameters, normalizer_from, predict


def load(path: str) -> list[dict]:
    return [json.loads(line) for line in open(path)]


def as_arrays(rows: list[dict]) -> tuple[np.ndarray, np.ndarray]:
    features = np.asarray([row["features"] for row in rows], dtype="float32")
    targets = np.asarray([row["judge_score"] for row in rows], dtype="float32")
    return features, targets


def report(model: GroundingScorer, rows: list[dict], mean: np.ndarray, std: np.ndarray) -> dict:
    errors: list[float] = []
    by_strategy: dict[str, list[float]] = {}
    predictions: list[float] = []
    targets: list[float] = []
    for row in rows:
        result = predict(model, row["features"], mean, std)
        error = abs(result["score"] - row["judge_score"])
        errors.append(error)
        by_strategy.setdefault(row.get("strategy", "?"), []).append(error)
        predictions.append(result["score"])
        targets.append(row["judge_score"])
    predictions_array = np.asarray(predictions)
    targets_array = np.asarray(targets)
    if predictions_array.std() > 1e-6 and targets_array.std() > 1e-6:
        correlation = float(np.corrcoef(predictions_array, targets_array)[0, 1])
    else:
        correlation = 0.0
    return {
        "n": len(rows),
        "mae": round(float(np.mean(errors)), 4),
        "rmse": round(float(np.sqrt(np.mean(np.square(errors)))), 4),
        "pearson_r": round(correlation, 4),
        "mae_by_strategy": {key: round(float(np.mean(value)), 4) for key, value in sorted(by_strategy.items())},
    }


def main() -> int:
    parser = argparse.ArgumentParser()
    parser.add_argument("--train", required=True)
    parser.add_argument("--val", required=True)
    parser.add_argument("--out", required=True)
    parser.add_argument("--metrics-out", default=None)
    parser.add_argument("--epochs", type=int, default=120)
    parser.add_argument("--batch-size", type=int, default=32)
    parser.add_argument("--lr", type=float, default=3e-3)
    parser.add_argument("--seed", type=int, default=0)
    args = parser.parse_args()

    torch.manual_seed(args.seed)
    np.random.seed(args.seed)

    train_rows = load(args.train)
    val_rows = load(args.val)
    train_features, train_targets = as_arrays(train_rows)
    mean, std = normalizer_from(train_features)

    standardised = (train_features - mean) / std
    inputs = torch.from_numpy(standardised).float()
    labels = torch.from_numpy(train_targets).float()

    model = GroundingScorer()
    print(f"model parameters: {count_parameters(model):,}")
    optimizer = torch.optim.Adam(model.parameters(), lr=args.lr, weight_decay=1e-4)
    score_loss = nn.MSELoss()
    confidence_loss = nn.MSELoss()

    order = np.arange(len(train_rows))
    for epoch in range(args.epochs):
        model.train()
        np.random.shuffle(order)
        epoch_loss = 0.0
        for start in range(0, len(order), args.batch_size):
            batch = order[start:start + args.batch_size]
            batch_inputs = inputs[batch]
            batch_labels = labels[batch]
            optimizer.zero_grad()
            score, confidence = model(batch_inputs)
            target_confidence = 1.0 - (score.detach() - batch_labels).abs()
            loss = score_loss(score, batch_labels) + 0.3 * confidence_loss(confidence, target_confidence)
            loss.backward()
            optimizer.step()
            epoch_loss += loss.item()
        if (epoch + 1) % 20 == 0 or epoch == 0:
            metrics = report(model, val_rows, mean, std)
            print(f"epoch {epoch + 1}/{args.epochs} loss={epoch_loss / max(1, len(order) // args.batch_size):.4f} "
                  f"val_mae={metrics['mae']} val_r={metrics['pearson_r']}")

    torch.save({"state_dict": model.state_dict(), "mean": mean.tolist(), "std": std.tolist()}, args.out)
    print(f"saved checkpoint to {args.out}")

    final = {
        "parameters": count_parameters(model),
        "train": report(model, train_rows, mean, std),
        "val": report(model, val_rows, mean, std),
    }
    print(json.dumps(final, indent=2))
    if args.metrics_out:
        with open(args.metrics_out, "w") as handle:
            json.dump(final, handle, indent=2)
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
