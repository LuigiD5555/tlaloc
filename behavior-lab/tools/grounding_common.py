"""Shared pieces for groundingscore-distilled-r0: the LM Studio embedding
client, the deterministic (question, answer, passage) -> feature-vector
transform, and the answer-perturbation strategies used to build training
triples.

The feature transform is the ONLY thing the trained MLP sees. It is kept
deliberately small and hand-designed: cheap cosine signals over frozen
MiniLM-L6 embeddings plus a few lexical statistics that target the known
failure mode of raw embedding similarity (a keyword dump of the passage
scores as "relevant" even though it is not a real answer). See
tools/GROUNDING_RESULTS.md.

Preprocessing here is imported by dataset build, training, calibration and
the HTTP service alike, so it never drifts between them.
"""
from __future__ import annotations

import json
import math
import re
import urllib.request

EMBED_MODEL = "text-embedding-all-minilm-l6-v2-embedding"
EMBED_DIM = 384

FEATURE_NAMES = (
    "cos_answer_passage",
    "cos_question_passage",
    "cos_question_answer",
    "answer_support_mean",
    "answer_support_min",
    "answer_support_max",
    "keyword_answer_in_passage",
    "keyword_question_in_answer",
    "verbatim_trigram_frac",
    "length_ratio",
    "type_token_ratio_answer",
    "function_word_frac_answer",
)
FEATURE_DIM = len(FEATURE_NAMES)

# A small closed set of English + Spanish function words. Real prose carries
# ~30-45% of these; a raw keyword dump of a passage carries almost none.
FUNCTION_WORDS = frozenset("""
a an the this that these those of in on at to from by for with without about
into over under between and or but nor so yet as is are was were be been being
it its they them their his her our your my we you he she i not no if then than
un una el la los las de en a por para con sin sobre entre y o pero como es son
era eran ser este esa eso estos esas su sus mi tu nuestro se lo le les del al que
""".split())

_WORD = re.compile(r"[a-záéíóúñü0-9]+", re.IGNORECASE)
_SENT_SPLIT = re.compile(r"(?<=[.!?])\s+|\n+")


def _tokens(text: str) -> list[str]:
    return [match.group(0).lower() for match in _WORD.finditer(text)]


def _sentences(text: str) -> list[str]:
    parts = [segment.strip() for segment in _SENT_SPLIT.split(text) if segment.strip()]
    return parts or ([text.strip()] if text.strip() else [])


def _cosine(left: list[float], right: list[float]) -> float:
    dot = sum(x * y for x, y in zip(left, right))
    norm_left = math.sqrt(sum(x * x for x in left))
    norm_right = math.sqrt(sum(y * y for y in right))
    if norm_left == 0.0 or norm_right == 0.0:
        return 0.0
    return dot / (norm_left * norm_right)


class Embedder:
    """Calls LM Studio's OpenAI-compatible /v1/embeddings once per unique
    string, memoized for the lifetime of the instance. No third-party SDK:
    a single urllib POST, same as tools/questionclass_serve.py's HTTP shape.
    """

    def __init__(self, base_url: str = "http://127.0.0.1:1234/v1", model: str = EMBED_MODEL,
                 timeout: float = 30.0) -> None:
        self.endpoint = base_url.rstrip("/") + "/embeddings"
        self.model = model
        self.timeout = timeout
        self._cache: dict[str, list[float]] = {}

    def _fetch(self, inputs: list[str]) -> list[list[float]]:
        payload = json.dumps({"model": self.model, "input": inputs}).encode("utf-8")
        request = urllib.request.Request(
            self.endpoint, data=payload, headers={"Content-Type": "application/json"})
        with urllib.request.urlopen(request, timeout=self.timeout) as response:
            body = json.loads(response.read())
        ordered = sorted(body["data"], key=lambda item: item["index"])
        return [item["embedding"] for item in ordered]

    def embed_many(self, texts: list[str]) -> list[list[float]]:
        """One HTTP call for every not-yet-cached string in `texts` (LM
        Studio's /v1/embeddings accepts a list), then serves the whole list
        from cache. Batching here is the difference between the dataset build
        taking ~40 minutes and taking hours."""
        keys = [text.strip() or " " for text in texts]
        missing = [key for key in dict.fromkeys(keys) if key not in self._cache]
        for start in range(0, len(missing), 64):
            chunk = missing[start:start + 64]
            for key, vector in zip(chunk, self._fetch(chunk)):
                self._cache[key] = vector
        return [self._cache[key] for key in keys]

    def embed(self, text: str) -> list[float]:
        return self.embed_many([text])[0]


def extract_features(embedder: Embedder, question: str, answer: str, passage: str) -> list[float]:
    """Deterministic (question, answer, passage) -> FEATURE_DIM floats. The
    trained head only ever sees this vector."""
    question = question.strip()
    answer = answer.strip()
    passage = passage.strip()

    answer_sentences = _sentences(answer)[:12]
    passage_chunks = _sentences(passage)[:30] or [passage or " "]

    chunk_texts = [chunk[:600] for chunk in passage_chunks]
    sentence_texts = [sentence[:600] for sentence in answer_sentences]
    batch = [question, answer, passage[:4000]] + chunk_texts + sentence_texts
    vectors = embedder.embed_many(batch)
    question_vector, answer_vector, passage_vector = vectors[0], vectors[1], vectors[2]
    chunk_start = 3
    sentence_start = chunk_start + len(chunk_texts)
    chunk_vectors = vectors[chunk_start:sentence_start]
    sentence_vectors = vectors[sentence_start:]

    per_sentence_support = [
        max((_cosine(sentence_vector, chunk_vector) for chunk_vector in chunk_vectors), default=0.0)
        for sentence_vector in sentence_vectors
    ]
    if not per_sentence_support:
        per_sentence_support = [0.0]

    answer_tokens = _tokens(answer)
    passage_tokens = _tokens(passage)
    passage_token_set = set(passage_tokens)
    question_tokens = _tokens(question)
    answer_token_set = set(answer_tokens)

    content_answer = [token for token in answer_tokens if token not in FUNCTION_WORDS]
    keyword_answer_in_passage = (
        sum(1 for token in set(content_answer) if token in passage_token_set) / max(len(set(content_answer)), 1)
    )
    content_question = [token for token in question_tokens if token not in FUNCTION_WORDS]
    keyword_question_in_answer = (
        sum(1 for token in set(content_question) if token in answer_token_set) / max(len(set(content_question)), 1)
    )

    answer_trigrams = list(zip(answer_tokens, answer_tokens[1:], answer_tokens[2:]))
    passage_trigrams = set(zip(passage_tokens, passage_tokens[1:], passage_tokens[2:]))
    verbatim_trigram_frac = (
        sum(1 for trigram in answer_trigrams if trigram in passage_trigrams) / max(len(answer_trigrams), 1)
    )

    length_ratio = min(len(answer) / max(len(passage), 1), 2.0)
    type_token_ratio_answer = len(answer_token_set) / max(len(answer_tokens), 1)
    function_word_frac_answer = (
        sum(1 for token in answer_tokens if token in FUNCTION_WORDS) / max(len(answer_tokens), 1)
    )

    return [
        _cosine(answer_vector, passage_vector),
        _cosine(question_vector, passage_vector),
        _cosine(question_vector, answer_vector),
        sum(per_sentence_support) / len(per_sentence_support),
        min(per_sentence_support),
        max(per_sentence_support),
        keyword_answer_in_passage,
        keyword_question_in_answer,
        verbatim_trigram_frac,
        length_ratio,
        type_token_ratio_answer,
        function_word_frac_answer,
    ]


# --- answer perturbation strategies -------------------------------------------------

_GROUNDED_LEADS = (
    "The passage explains that ", "According to this material, ",
    "This section states that ", "The text indicates that ",
)


def grounded_answer(passage: str, rng=None) -> str:
    """A faithful answer: the two most substantial passage sentences, reordered
    and prefixed so it reads as prose rather than a verbatim copy (which would
    make verbatim-trigram overlap an unrealistic tell)."""
    sentences = sorted(_sentences(passage), key=len, reverse=True)[:2]
    if not sentences:
        return passage[:300].strip()
    if rng is not None and len(sentences) == 2 and rng.random() < 0.5:
        sentences.reverse()
    lead = (rng.choice(_GROUNDED_LEADS) if rng is not None else _GROUNDED_LEADS[0])
    body = " ".join(sentences)
    body = body[0].lower() + body[1:] if body else body
    return (lead + body).strip()


def terse_grounded_answer(passage: str, rng=None) -> str:
    """A short faithful answer: one substantial passage sentence, trimmed and
    prefixed. Real good answers are often terse paraphrases, not the two
    longest sentences reworded — training only on the latter teaches the head
    to under-score brevity."""
    candidates = [sentence for sentence in _sentences(passage) if 40 <= len(sentence) <= 180]
    if not candidates:
        candidates = sorted(_sentences(passage), key=len)[:1] or [passage[:120]]
    chosen = (rng.choice(candidates) if rng is not None else candidates[0]).strip()
    lead = (rng.choice(_GROUNDED_LEADS) if rng is not None else _GROUNDED_LEADS[0])
    return (lead + chosen[0].lower() + chosen[1:]).strip()


# Note: a "contradiction" strategy (on-topic answer stating the opposite of a
# passage fact) was tried and dropped — the feature set has no entailment
# signal, so contradiction answers are indistinguishable from grounded ones
# here and their examples only added label noise. Catching contradiction needs
# an NLI feature/model; see tools/GROUNDING_RESULTS.md (r1 direction).


def keyword_dump_answer(passage: str) -> str:
    """The adversarial case: content words from the passage, comma-joined,
    no sentence structure. High lexical overlap, not a real answer."""
    content = [token for token in dict.fromkeys(_tokens(passage)) if token not in FUNCTION_WORDS]
    return ", ".join(content[:25])


def partial_answer(passage: str, distractor_passage: str) -> str:
    """One real passage sentence plus one sentence from an unrelated page."""
    real = _sentences(passage)[:1]
    fake = sorted(_sentences(distractor_passage), key=len, reverse=True)[:1]
    return " ".join(real + fake).strip()


def offtopic_answer(distractor_passage: str) -> str:
    """A coherent answer that is simply about a different page."""
    return " ".join(sorted(_sentences(distractor_passage), key=len, reverse=True)[:2]).strip()


def filler_answer() -> str:
    """Generic non-answer the parrot sometimes emits when it has nothing."""
    return ("This section discusses several relevant concepts and provides context "
            "about the broader topic, drawing on established ideas in the field.")
