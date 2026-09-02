"""Builds the training dataset for groundingscore-distilled-r0.

Pipeline:
  1. read real page text (tlaloc-fold-bench dump-pages output)
  2. for each page, synthesise a few plausible questions from its own text
  3. for each question, build several (answer, passage) variants spanning the
     grounding spectrum: faithful, keyword-dump, partially grounded, off-topic,
     generic filler
  4. label each variant by CONSTRUCTION (STRATEGY_LABEL) — the strategy that
     produced the answer determines its grounding level, same "labels are
     free" principle as tools/questionclass_dataset.py. An optional LM Studio
     judge (--judge-model) can be blended in when a usable one is available;
     it is off by default because the small local judges tested collapsed to
     a constant score. The honest test is the hand-authored OOD set
     (grounding_ood.py), never the same-generator val split.
  5. cache features now so training/calibration never re-hit the embedding API

Output JSONL rows: {question, answer, passage_page, features:[...],
judge_score, label_source, strategy}. Split deterministically by page into
train/val so no page leaks across the split.

Usage:
    tlaloc-fold-bench dump-pages -store /tmp/foldstore-swarms -stride 7 -limit 60 > /tmp/gpages.jsonl
    python3 grounding_dataset.py --pages /tmp/gpages.jsonl \
        --train-out /tmp/grounding-train.jsonl --val-out /tmp/grounding-val.jsonl
"""
from __future__ import annotations

import argparse
import json
import random
import re
import urllib.request

from grounding_common import (
    Embedder, extract_features, _sentences, _tokens, FUNCTION_WORDS,
    grounded_answer, terse_grounded_answer,
    keyword_dump_answer, partial_answer, offtopic_answer, filler_answer,
)

# Grounding level implied by the strategy that built the answer. Mild jitter
# is added per row (deterministic on the row hash) so the head does not just
# learn five discrete points.
STRATEGY_LABEL = {
    "grounded": 0.9,
    "terse_grounded": 0.85,
    "partial": 0.5,
    "keyword_dump": 0.12,
    "offtopic": 0.08,
    "filler": 0.1,
}

JUDGE_SYSTEM = (
    "You are a strict grounding judge. Given a PASSAGE, a QUESTION, and a candidate "
    "ANSWER, rate how well the ANSWER is supported by the PASSAGE for that QUESTION.\n"
    "Scale anchors:\n"
    "  1.0 = the answer is fully and accurately supported by the passage\n"
    "  0.6 = mostly supported, minor gaps or one unsupported detail\n"
    "  0.3 = only loosely related, or half the answer is unsupported\n"
    "  0.0 = not supported, off-topic, generic filler, OR just a list of keywords\n"
    "        copied from the passage rather than a real answer\n"
    "Do not reward fluency. Do not answer the question yourself. Reply on exactly "
    "two lines:\nSCORE: <0.0-1.0>\nCONFIDENCE: <0.0-1.0>"
)

_SCORE_RE = re.compile(r"SCORE:\s*([01](?:\.\d+)?)", re.IGNORECASE)
_CONF_RE = re.compile(r"CONFIDENCE:\s*([01](?:\.\d+)?)", re.IGNORECASE)


def salient_terms(page_text: str, count: int) -> list[str]:
    frequency: dict[str, int] = {}
    for token in _tokens(page_text):
        if len(token) < 4 or token in FUNCTION_WORDS or token.isdigit():
            continue
        frequency[token] = frequency.get(token, 0) + 1
    return [term for term, _ in sorted(frequency.items(), key=lambda kv: kv[1], reverse=True)[:count]]


def synthesise_questions(page_text: str) -> list[str]:
    terms = salient_terms(page_text, 4)
    lead = _sentences(page_text)[0][:80] if _sentences(page_text) else "this section"
    questions = []
    if terms:
        questions.append(f"What does this material say about {terms[0]}?")
    if len(terms) >= 2:
        questions.append(f"How are {terms[0]} and {terms[1]} related here?")
    questions.append(f"What is the main point of the part that begins \"{lead}\"?")
    if len(terms) >= 3:
        questions.append(f"Explain the role of {terms[2]} in this material.")
    return questions[:3]


def _judge_once(endpoint: str, model: str, passage: str, question: str, answer: str,
                timeout: float) -> tuple[float, float]:
    user = f"PASSAGE:\n{passage[:3500]}\n\nQUESTION:\n{question}\n\nANSWER:\n{answer[:1500]}"
    payload = json.dumps({
        "model": model,
        "temperature": 0.0,
        "max_tokens": 40,
        "messages": [
            {"role": "system", "content": JUDGE_SYSTEM},
            {"role": "user", "content": user},
        ],
    }).encode("utf-8")
    request = urllib.request.Request(endpoint, data=payload, headers={"Content-Type": "application/json"})
    with urllib.request.urlopen(request, timeout=timeout) as response:
        body = json.loads(response.read())
    text = body["choices"][0]["message"]["content"]
    score_match = _SCORE_RE.search(text)
    if not score_match:
        raise ValueError(f"no SCORE in {model} reply: {text!r}")
    confidence_match = _CONF_RE.search(text)
    score = max(0.0, min(1.0, float(score_match.group(1))))
    confidence = max(0.0, min(1.0, float(confidence_match.group(1)))) if confidence_match else 0.6
    return score, confidence


def judge(endpoint: str, models: list[str], passage: str, question: str, answer: str,
          timeout: float) -> tuple[float, float, dict]:
    """Averages the grounding score across every judge model. Requires at
    least one to answer; records each model's raw score so the ensemble is
    auditable. Ensemble variance feeds the stored confidence: judges that
    disagree -> lower confidence label."""
    per_model: dict[str, float] = {}
    confidences: list[float] = []
    last_error: Exception | None = None
    for model in models:
        try:
            score, confidence = _judge_once(endpoint, model, passage, question, answer, timeout)
        except Exception as exc:  # noqa: BLE001 - tolerate one judge failing
            last_error = exc
            continue
        per_model[model] = round(score, 4)
        confidences.append(confidence)
    if not per_model:
        raise last_error or RuntimeError("all judges failed")
    scores = list(per_model.values())
    mean_score = sum(scores) / len(scores)
    spread = (max(scores) - min(scores)) if len(scores) > 1 else 0.0
    confidence = max(0.0, (sum(confidences) / len(confidences)) - spread)
    return mean_score, confidence, per_model


def build_variants(pages: list[dict], rng: random.Random) -> list[dict]:
    rows = []
    for page in pages:
        passage = page["content"]
        if len(passage) < 200:
            continue
        distractor = rng.choice([other for other in pages if other["page"] != page["page"]])["content"]
        for question in synthesise_questions(passage):
            candidates = [
                ("grounded", grounded_answer(passage, rng)),
                ("terse_grounded", terse_grounded_answer(passage, rng)),
                ("keyword_dump", keyword_dump_answer(passage)),
                ("partial", partial_answer(passage, distractor)),
                ("offtopic", offtopic_answer(distractor)),
                ("filler", filler_answer()),
            ]
            for strategy, answer in candidates:
                if answer and len(answer) > 15:
                    rows.append({"page": page["page"], "question": question,
                                 "answer": answer, "passage": passage, "strategy": strategy})
    return rows


def main() -> int:
    parser = argparse.ArgumentParser()
    parser.add_argument("--pages", required=True, help="JSONL from tlaloc-fold-bench dump-pages")
    parser.add_argument("--train-out", required=True)
    parser.add_argument("--val-out", required=True)
    parser.add_argument(
        "--judge-model", default="",
        help="optional comma-separated LM Studio judge models; blended with the "
             "constructed label via --judge-weight. Empty = constructed labels only")
    parser.add_argument("--judge-weight", type=float, default=0.5,
                        help="weight of the LLM judge vs the constructed label (0..1)")
    parser.add_argument("--base-url", default="http://127.0.0.1:1234/v1")
    parser.add_argument("--timeout", type=float, default=90.0)
    parser.add_argument("--val-fraction", type=float, default=0.2)
    parser.add_argument("--seed", type=int, default=0)
    parser.add_argument("--jitter", type=float, default=0.06)
    parser.add_argument("--limit", type=int, default=0, help="cap total triples (0 = all)")
    args = parser.parse_args()

    rng = random.Random(args.seed)
    pages = [json.loads(line) for line in open(args.pages)]
    if len(pages) < 4:
        raise SystemExit("need at least 4 pages")

    triples = build_variants(pages, rng)
    rng.shuffle(triples)
    if args.limit:
        triples = triples[: args.limit]

    val_pages = set(rng.sample([page["page"] for page in pages],
                               max(1, int(len(pages) * args.val_fraction))))

    embedder = Embedder(base_url=args.base_url)
    chat_endpoint = args.base_url.rstrip("/") + "/chat/completions"
    judge_models = [name.strip() for name in args.judge_model.split(",") if name.strip()]

    label_rng = random.Random(args.seed + 1)
    train_handle = open(args.train_out, "w")
    val_handle = open(args.val_out, "w")
    kept = skipped = 0
    for index, triple in enumerate(triples):
        constructed = STRATEGY_LABEL[triple["strategy"]]
        constructed += label_rng.uniform(-args.jitter, args.jitter)
        constructed = max(0.0, min(1.0, constructed))
        label_source = "constructed"
        per_judge: dict = {}

        try:
            features = extract_features(embedder, triple["question"], triple["answer"], triple["passage"])
        except Exception as exc:  # noqa: BLE001 - a flaky embedding call shouldn't kill the run
            skipped += 1
            print(f"[{index + 1}/{len(triples)}] skip ({triple['strategy']}): {exc}")
            continue

        score = constructed
        if judge_models:
            try:
                judged, _, per_judge = judge(chat_endpoint, judge_models, triple["passage"],
                                             triple["question"], triple["answer"], args.timeout)
                score = (1.0 - args.judge_weight) * constructed + args.judge_weight * judged
                label_source = "blended"
            except Exception as exc:  # noqa: BLE001 - fall back to the constructed label
                print(f"[{index + 1}/{len(triples)}] judge failed, using constructed: {exc}")

        row = {
            "question": triple["question"], "answer": triple["answer"],
            "passage_page": triple["page"], "strategy": triple["strategy"],
            "features": [round(value, 6) for value in features],
            "judge_score": round(score, 4), "label_source": label_source,
            "judge_per_model": per_judge,
        }
        handle = val_handle if triple["page"] in val_pages else train_handle
        handle.write(json.dumps(row) + "\n")
        handle.flush()
        kept += 1
        if (index + 1) % 20 == 0:
            print(f"[{index + 1}/{len(triples)}] kept={kept} skipped={skipped}")

    train_handle.close()
    val_handle.close()
    print(f"done: kept={kept} skipped={skipped}; val pages={sorted(val_pages)}")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
