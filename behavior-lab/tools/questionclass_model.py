"""A tiny character-level CNN, trained from scratch (no HuggingFace
download, no tokenizer dependency), that reads a question string and
predicts its rhetorical shape: DEFINITION, COMPARISON, PROCESS,
FACTUAL_DETAIL, or GENERAL.

Character-level on purpose: the swarm asks questions in both Spanish and
English, with typos, accents, and varied phrasing. A char CNN has a tiny
fixed vocabulary, no out-of-vocabulary problem, and learns prefix/keyword
cues ("what is", "¿cómo", "vs", a bare year) without a word tokenizer. The
task is shallow classification of a short string, so a couple of 1-D
convolutions over character embeddings are the right-sized tool.
"""
import numpy as np
import torch
import torch.nn as nn

LABELS = ("DEFINITION", "COMPARISON", "PROCESS", "FACTUAL_DETAIL", "GENERAL")
LABEL_TO_IDX = {label: idx for idx, label in enumerate(LABELS)}

# Index 0 is reserved for padding and any unknown character.
VOCAB = " abcdefghijklmnopqrstuvwxyz0123456789áéíóúñü¿?¡!.,;:'\"-()/&%"
CHAR_TO_IDX = {char: idx + 1 for idx, char in enumerate(VOCAB)}
VOCAB_SIZE = len(VOCAB) + 1
MAX_LEN = 100


def encode(text: str) -> np.ndarray:
    """Lowercases, maps each character to its vocab index, truncates or
    right-pads to MAX_LEN. Shared by training, file inference, and the HTTP
    service so preprocessing never drifts between them."""
    lowered = text.strip().lower()[:MAX_LEN]
    indices = [CHAR_TO_IDX.get(char, 0) for char in lowered]
    indices.extend([0] * (MAX_LEN - len(indices)))
    return np.array(indices, dtype="int64")


class QuestionTypeClassifier(nn.Module):
    def __init__(self) -> None:
        super().__init__()
        self.embedding = nn.Embedding(VOCAB_SIZE, 20, padding_idx=0)
        self.features = nn.Sequential(
            nn.Conv1d(20, 32, kernel_size=5, padding=2), nn.ReLU(),
            nn.Conv1d(32, 32, kernel_size=3, padding=1), nn.ReLU(),
            nn.AdaptiveMaxPool1d(1),
        )
        self.head = nn.Linear(32, len(LABELS))

    def forward(self, tokens: torch.Tensor) -> torch.Tensor:
        embedded = self.embedding(tokens).permute(0, 2, 1)  # (batch, channels, length)
        pooled = self.features(embedded).flatten(1)
        return self.head(pooled)


def count_parameters(model: nn.Module) -> int:
    return sum(p.numel() for p in model.parameters() if p.requires_grad)


def predict_text(model: QuestionTypeClassifier, text: str) -> dict:
    model.eval()
    tokens = torch.from_numpy(encode(text)).unsqueeze(0)
    with torch.no_grad():
        logits = model(tokens)
        probabilities = torch.softmax(logits, dim=1)[0]
    best = int(probabilities.argmax().item())
    return {"type": LABELS[best], "confidence": float(probabilities[best].item())}
