"""groundingscore-distilled-r0: a tiny MLP (a few hundred parameters) that
regresses the strong judge's grounding score from the 12 hand-designed
features in grounding_common.extract_features.

Two outputs:
  score      - sigmoid, the distilled grounding score in [0, 1]
  confidence - sigmoid, the model's self-estimate of how close its score is
               to the judge's. Trained to predict 1 - |score - target|, so a
               low confidence genuinely means "this input is unlike what I
               learned". The swarm consolidator uses it to decide whether to
               fall back to a heavier judge.

Deliberately not a transformer, not a tree ensemble: the features already
encode the semantics (via frozen MiniLM cosines) and the lexical tells; the
head only has to learn how to weigh and combine ~12 numbers.
"""
from __future__ import annotations

import numpy as np
import torch
import torch.nn as nn

from grounding_common import FEATURE_DIM, FEATURE_NAMES

HIDDEN = 24


class GroundingScorer(nn.Module):
    def __init__(self) -> None:
        super().__init__()
        self.trunk = nn.Sequential(
            nn.Linear(FEATURE_DIM, HIDDEN), nn.ReLU(),
            nn.Linear(HIDDEN, HIDDEN // 2), nn.ReLU(),
        )
        self.score_head = nn.Linear(HIDDEN // 2, 1)
        self.confidence_head = nn.Linear(HIDDEN // 2, 1)

    def forward(self, features: torch.Tensor) -> tuple[torch.Tensor, torch.Tensor]:
        hidden = self.trunk(features)
        score = torch.sigmoid(self.score_head(hidden)).squeeze(-1)
        confidence = torch.sigmoid(self.confidence_head(hidden)).squeeze(-1)
        return score, confidence


def count_parameters(model: nn.Module) -> int:
    return sum(parameter.numel() for parameter in model.parameters() if parameter.requires_grad)


def normalizer_from(feature_rows: np.ndarray) -> tuple[np.ndarray, np.ndarray]:
    """Per-feature mean/std for standardisation. Saved alongside the weights
    so the service applies the exact same transform."""
    mean = feature_rows.mean(axis=0)
    std = feature_rows.std(axis=0)
    std[std < 1e-6] = 1.0
    return mean, std


def predict(model: GroundingScorer, features: list[float],
            mean: np.ndarray, std: np.ndarray) -> dict:
    model.eval()
    standardised = (np.asarray(features, dtype="float32") - mean) / std
    with torch.no_grad():
        score, confidence = model(torch.from_numpy(standardised).float().unsqueeze(0))
    return {"score": float(score.item()), "confidence": float(confidence.item())}


assert len(FEATURE_NAMES) == FEATURE_DIM
