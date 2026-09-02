"""Generates an OUT-OF-DISTRIBUTION evaluation set for
questionclass-charcnn-r0. Deliberately unlike questionclass_dataset.py:
indirect/embedded questions, imperatives with no question word, multi-clause
sentences, ES/EN code-switching, dropped accents and casing, and topics
*outside* the training vocabulary (finance, biology, cooking). The
rhetorical-shape label is still known by construction. This is the slice
that decides whether the model has any business being ACTIVE — see
internal/tlaloque/calibration.

Usage:
    python3 questionclass_ood.py --seed 7007 --count 300 --out /tmp/qclass-ood.jsonl
"""
import argparse
import json
import random

from questionclass_model import LABEL_TO_IDX

# Topics on purpose far from the swarm-domain training vocabulary.
OFF_DOMAIN = [
    "compound interest", "the Krebs cycle", "sourdough fermentation",
    "double-entry bookkeeping", "photosynthesis", "tectonic subduction",
    "the offside rule", "monetary tightening", "mitochondrial dna",
    "gluten development", "amortization schedules", "capillary action",
    "el interes compuesto", "la fotosintesis", "la fermentacion del pan",
    "la contabilidad por partida doble", "la tectonica de placas",
]

# Surface forms that share little structure with the training templates.
FORMS = {
    "DEFINITION": [
        "i keep hearing the term {a} but honestly i have no idea what it refers to",
        "someone said {a} today and i just nodded — what is that actually",
        "a colleague used {a} in a meeting, mind unpacking the term for me",
        "no tengo claro a que le llaman {a}, me lo explicas",
        "pretend i'm five: {a}",
        "the doc assumes i know what {a} is. i don't.",
    ],
    "COMPARISON": [
        "i can never keep {a} and {b} straight, how do they actually differ",
        "my notes treat {a} and {b} as the same thing, is that wrong",
        "if i already understand {a}, what's the extra idea in {b}",
        "siempre confundo {a} con {b}, en que se distinguen realmente",
        "someone told me {a} is just {b} with a new name, true or not",
        "trade-offs of {a} versus {b} for a small team",
    ],
    "PROCESS": [
        "i get what {a} is, i just don't get how it actually happens step by step",
        "what has to go right, in order, for {a} to occur",
        "walk me from nothing to {a}, i learn better as a sequence",
        "no entiendo el como: por que mecanismo termina ocurriendo {a}",
        "if i wanted to reproduce {a} myself, what would i do first, then next",
        "the chapter says {a} 'emerges' but never says through what",
    ],
    "FACTUAL_DETAIL": [
        "somewhere in here there's a number for {a}, i need that exact figure",
        "which section has the concrete measurement for {a}",
        "the appendix table lists {a}, what value does it give",
        "en que pagina esta el dato exacto de {a}",
        "i just need the count they report for {a}, not the discussion",
        "the caption of the {a} figure cites a value, what is it",
    ],
    "GENERAL": [
        "give me the gist of {a}, i don't need details yet",
        "just orient me on {a} before i read the whole thing",
        "high level, what is this part about {a} trying to tell me",
        "en general de que va la seccion sobre {a}",
        "i have two minutes — the {a} summary please",
        "broadly, what should i take away about {a}",
    ],
}

IN_DOMAIN_HINTS = [
    "swarm behavior", "ant coordination", "decentralized agents",
    "emergent order", "robot collectives", "feedback in swarms",
]


def roughen(text: str, rng: random.Random) -> str:
    if rng.random() < 0.35:
        text = text.replace("i ", "I ", 1) if text.startswith("i ") else text
    if rng.random() < 0.3:
        text = text + "?"
    if rng.random() < 0.15:  # drop a random space (typo)
        pos = rng.randint(1, max(1, len(text) - 2))
        text = text[:pos] + text[pos + 1:]
    return text


def main() -> int:
    parser = argparse.ArgumentParser()
    parser.add_argument("--seed", type=int, required=True)
    parser.add_argument("--count", type=int, required=True)
    parser.add_argument("--out", required=True)
    args = parser.parse_args()

    rng = random.Random(args.seed)
    per_class = args.count // len(LABEL_TO_IDX)
    with open(args.out, "w") as handle:
        for label in LABEL_TO_IDX:
            for _ in range(per_class):
                form = rng.choice(FORMS[label])
                # 60% genuinely off-domain, 40% in-domain but oddly phrased.
                if rng.random() < 0.6:
                    first, second = rng.sample(OFF_DOMAIN, 2)
                else:
                    first, second = rng.sample(IN_DOMAIN_HINTS, 2)
                text = roughen(form.format(a=first, b=second), rng)
                handle.write(json.dumps({
                    "text": text, "label": label, "label_idx": LABEL_TO_IDX[label],
                }, ensure_ascii=False) + "\n")
    print(f"wrote {per_class * len(LABEL_TO_IDX)} OOD examples (seed {args.seed}) to {args.out}")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
