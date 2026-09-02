"""Generates labeled question-type training data. Pure Python: the labels
are free because each question is built from a template whose rhetorical
class is known by construction. Bilingual (English + Spanish) and roughened
with surface noise (casing, punctuation, filler prefixes) so the model
learns the actual cues ("what is", "¿cómo", " vs ", a bare year) rather
than memorizing templates.

Usage:
    python3 questionclass_dataset.py --seed 1001 --count 6000 --out /tmp/qclass-train.jsonl
"""
import argparse
import json
import random

from questionclass_model import LABEL_TO_IDX

TOPICS = [
    "swarm intelligence", "emergent behavior", "stigmergy", "self-organization",
    "ant colony optimization", "particle swarm optimization", "flocking",
    "decentralized control", "multi-agent systems", "collective decision making",
    "pheromone signaling", "task allocation", "robot swarms", "consensus protocols",
    "feedback loops", "scalability", "fault tolerance", "the boids model",
    "quorum sensing", "division of labor", "swarm robotics", "cellular automata",
    "reinforcement learning", "distributed algorithms", "agent communication",
    "stigmergy", "a superorganism", "emergence", "flocking", "schooling",
    "the shortest-path mechanism", "the coordination process", "the convergence rate",
    "el comportamiento emergente", "la inteligencia de enjambre", "la estigmergia",
    "la autoorganización", "los sistemas multiagente", "el control descentralizado",
    "la asignación de tareas", "los enjambres de robots", "los bucles de retroalimentación",
    "la tolerancia a fallos", "la toma de decisiones colectiva", "la señalización por feromonas",
    "un superorganismo", "la emergencia", "el proceso de coordinación",
]

# Referents for FACTUAL_DETAIL questions that anchor to a locator rather
# than a year (page / figure / table / section number).
LOCATORS = ["page", "figure", "table", "section", "chapter"]
ES_LOCATORS = ["página", "figura", "tabla", "sección", "capítulo"]

EN_TOPICS = [topic for topic in TOPICS if not topic.startswith(("el ", "la ", "los ", "las "))]
ES_TOPICS = [topic for topic in TOPICS if topic.startswith(("el ", "la ", "los ", "las "))]

YEARS = [str(year) for year in range(1987, 2025)]

EN_FILLERS = ["", "", "", "Tell me, ", "I wonder, ", "Quick question: ", "For my notes, "]
ES_FILLERS = ["", "", "", "Oye, ", "Tengo una duda: ", "Para mis apuntes, ", "Rápido: "]

TEMPLATES = {
    "DEFINITION": {
        "en": [
            "what is {a}", "what are {a}", "define {a}", "what does {a} mean",
            "what is meant by {a}", "can you define {a}", "what exactly is {a}",
            "give me a definition of {a}", "what is the definition of {a}",
            "what's {a}", "whats {a}", "meaning of {a}", "definition of {a}",
            "what would you call {a}", "what is {a}, exactly",
        ],
        "es": [
            "qué es {a}", "qué son {a}", "define {a}", "qué significa {a}",
            "a qué se refiere {a}", "puedes definir {a}", "cuál es la definición de {a}",
            "en qué consiste {a}", "significado de {a}", "definición de {a}",
            "qué quiere decir {a}",
        ],
    },
    "COMPARISON": {
        "en": [
            "what is the difference between {a} and {b}", "compare {a} and {b}",
            "how does {a} relate to {b}", "how is {a} different from {b}",
            "{a} vs {b}, which is better", "what do {a} and {b} have in common",
            "is {a} the same as {b}", "contrast {a} with {b}",
            "what is the relationship between {a} and {b}",
        ],
        "es": [
            "cuál es la diferencia entre {a} y {b}", "compara {a} y {b}",
            "cómo se relaciona {a} con {b}", "en qué se diferencian {a} y {b}",
            "{a} frente a {b}, cuál es mejor", "qué tienen en común {a} y {b}",
            "es lo mismo {a} que {b}", "cuál es la relación entre {a} y {b}",
        ],
    },
    "PROCESS": {
        "en": [
            "how does {a} work", "how do {a} emerge", "how is {a} achieved",
            "why does {a} happen", "how do agents coordinate through {a}",
            "walk me through how {a} works", "what steps produce {a}",
            "how would you implement {a}", "why do {a} form",
            "explain the mechanism by which {a} arises", "explain how {a} works",
            "describe the process behind {a}", "how exactly does {a} unfold",
            "by what process does {a} occur", "what makes {a} happen",
        ],
        "es": [
            "cómo funciona {a}", "cómo surge {a}", "cómo se logra {a}",
            "por qué ocurre {a}", "explícame paso a paso cómo funciona {a}",
            "qué pasos producen {a}", "cómo se implementa {a}", "por qué se forma {a}",
            "explica el mecanismo por el cual aparece {a}", "describe el proceso detrás de {a}",
            "por qué se produce {a}",
        ],
    },
    "FACTUAL_DETAIL": {
        "en": [
            "what happened with {a} in {y}", "how many experiments on {a} were run in {y}",
            "what did the {y} study of {a} conclude", "who published on {a} in {y}",
            "what was the result reported for {a} in {y}",
            "in {y}, how large was the {a} simulation",
            "which {loc} discusses {a}", "what does {loc} {n} say about {a}",
            "on {loc} {n}, what is the value for {a}",
            "how many agents were used for {a}", "what number is given for {a}",
            "which {loc} shows {a}", "what figure is cited for {a} in {y}",
        ],
        "es": [
            "qué pasó con {a} en {y}", "cuántos experimentos sobre {a} se hicieron en {y}",
            "qué concluyó el estudio de {y} sobre {a}", "quién publicó sobre {a} en {y}",
            "cuál fue el resultado reportado para {a} en {y}",
            "en {y}, qué tan grande fue la simulación de {a}",
            "qué {esloc} habla de {a}", "qué dice la {esloc} {n} sobre {a}",
            "cuántos agentes se usaron para {a}", "qué {esloc} muestra {a}",
        ],
    },
    "GENERAL": {
        "en": [
            "tell me about {a}", "summarize {a}", "give me an overview of {a}",
            "i want to know more about {a}", "what can you say about {a}",
            "anything interesting about {a}", "describe {a}", "talk about {a}",
        ],
        "es": [
            "cuéntame sobre {a}", "resume {a}", "dame un panorama de {a}",
            "quiero saber más sobre {a}", "qué me puedes decir de {a}",
            "algo interesante sobre {a}", "describe {a}", "háblame de {a}",
        ],
    },
}


def roughen(sentence: str, language: str, rng: random.Random) -> str:
    filler = rng.choice(EN_FILLERS if language == "en" else ES_FILLERS)
    text = filler + sentence
    if filler == "" and rng.random() < 0.6:
        text = text[0].upper() + text[1:]
    if language == "es" and rng.random() < 0.5:
        text = "¿" + text[0].lower() + text[1:] + "?"
    elif rng.random() < 0.7:
        text = text + "?"
    if rng.random() < 0.1:
        text = text + " " + rng.choice(["please", "thanks", "por favor", "gracias"])
    return text


def make_example(label: str, rng: random.Random) -> dict:
    language = rng.choice(["en", "es"])
    topics = EN_TOPICS if language == "en" else ES_TOPICS
    template = rng.choice(TEMPLATES[label][language])
    first, second = rng.sample(topics, 2)
    sentence = template.format(
        a=first, b=second, y=rng.choice(YEARS),
        loc=rng.choice(LOCATORS), esloc=rng.choice(ES_LOCATORS), n=rng.randint(2, 380),
    )
    return {"text": roughen(sentence, language, rng), "label": label, "label_idx": LABEL_TO_IDX[label]}


def main() -> int:
    parser = argparse.ArgumentParser()
    parser.add_argument("--seed", type=int, required=True)
    parser.add_argument("--count", type=int, required=True, help="total examples (split evenly across the 5 classes)")
    parser.add_argument("--out", required=True)
    args = parser.parse_args()

    rng = random.Random(args.seed)
    per_class = args.count // len(LABEL_TO_IDX)
    seen: set[str] = set()
    written = 0
    with open(args.out, "w") as handle:
        for label in LABEL_TO_IDX:
            produced = 0
            attempts = 0
            while produced < per_class and attempts < per_class * 40:
                attempts += 1
                example = make_example(label, rng)
                if example["text"] in seen:
                    continue
                seen.add(example["text"])
                handle.write(json.dumps(example, ensure_ascii=False) + "\n")
                produced += 1
                written += 1
    print(f"wrote {written} examples ({per_class}/class, seed {args.seed}) to {args.out}")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
