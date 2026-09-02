"""Trains QuestionTypeClassifier on JSONL produced by
questionclass_dataset.py. CPU-only by design — the model is a few thousand
parameters.

Usage:
    python3 questionclass_train.py --train train.jsonl --val val.jsonl --out model.pt --epochs 15
"""
import argparse
import json

import numpy as np
import torch
import torch.nn as nn
from torch.utils.data import DataLoader, Dataset

from questionclass_model import LABELS, QuestionTypeClassifier, count_parameters, encode


class QuestionDataset(Dataset):
    def __init__(self, manifest_path: str) -> None:
        self.records = [json.loads(line) for line in open(manifest_path)]

    def __len__(self) -> int:
        return len(self.records)

    def __getitem__(self, idx: int):
        record = self.records[idx]
        return torch.from_numpy(encode(record["text"])), record["label_idx"]


def collate(batch):
    tokens = torch.stack([item[0] for item in batch])
    labels = torch.tensor([item[1] for item in batch], dtype=torch.long)
    return tokens, labels


def evaluate(model: nn.Module, loader: DataLoader) -> dict:
    model.eval()
    per_class_correct = np.zeros(len(LABELS))
    per_class_total = np.zeros(len(LABELS))
    with torch.no_grad():
        for tokens, labels in loader:
            preds = model(tokens).argmax(dim=1)
            for label, pred in zip(labels.tolist(), preds.tolist()):
                per_class_total[label] += 1
                if label == pred:
                    per_class_correct[label] += 1
    overall = per_class_correct.sum() / max(per_class_total.sum(), 1)
    per_class = {
        LABELS[idx]: round(per_class_correct[idx] / per_class_total[idx], 3)
        for idx in range(len(LABELS)) if per_class_total[idx] > 0
    }
    return {"overall": round(float(overall), 4), "per_class": per_class}


def main() -> int:
    parser = argparse.ArgumentParser()
    parser.add_argument("--train", required=True)
    parser.add_argument("--val", required=True)
    parser.add_argument("--out", required=True)
    parser.add_argument("--metrics-out", default=None)
    parser.add_argument("--epochs", type=int, default=15)
    parser.add_argument("--batch-size", type=int, default=64)
    parser.add_argument("--lr", type=float, default=1e-3)
    args = parser.parse_args()

    torch.manual_seed(0)
    train_loader = DataLoader(QuestionDataset(args.train), batch_size=args.batch_size, shuffle=True, collate_fn=collate)
    val_loader = DataLoader(QuestionDataset(args.val), batch_size=args.batch_size, shuffle=False, collate_fn=collate)

    model = QuestionTypeClassifier()
    print(f"model parameters: {count_parameters(model):,}")

    optimizer = torch.optim.Adam(model.parameters(), lr=args.lr)
    criterion = nn.CrossEntropyLoss()

    metrics = {}
    for epoch in range(args.epochs):
        model.train()
        total_loss = 0.0
        for tokens, labels in train_loader:
            optimizer.zero_grad()
            loss = criterion(model(tokens), labels)
            loss.backward()
            optimizer.step()
            total_loss += loss.item()
        metrics = evaluate(model, val_loader)
        print(f"epoch {epoch + 1}/{args.epochs} train_loss={total_loss / len(train_loader):.4f} val={metrics}")

    torch.save(model.state_dict(), args.out)
    print(f"saved checkpoint to {args.out}")

    if args.metrics_out:
        with open(args.metrics_out, "w") as handle:
            json.dump({"val": metrics, "parameters": count_parameters(model)}, handle, indent=2)
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
