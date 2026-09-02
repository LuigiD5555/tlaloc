"""Writes a hand-authored out-of-distribution evaluation set for
groundingscore-distilled-r0: (question, answer, passage, label) rows where
the passage is NOT a page from the training book and the answer is NOT one
of grounding_common's mechanical perturbations. label is the author's
grounding judgement in [0, 1].

This is the honest test — same role as questionclass_ood.py. In-distribution
val accuracy on template-built triples is nearly free and means little on its
own.

Usage:
    python3 grounding_ood.py --out /tmp/grounding-ood.jsonl
"""
from __future__ import annotations

import argparse
import json

PASSAGE_SWARM = (
    "A thinking swarm is a collection of simple autonomous agents whose local "
    "interactions produce global problem-solving behaviour that no single agent "
    "is programmed to perform. Coordination is decentralised: each agent follows "
    "a small rule set and reacts to nearby agents and the environment. Robustness "
    "comes from redundancy, since the loss of individual agents rarely changes the "
    "collective outcome."
)
PASSAGE_STIGMERGY = (
    "Stigmergy is indirect coordination through the environment. An agent changes "
    "the environment, and that change later prompts action by the same or another "
    "agent. Ant trail pheromones are the classic example: foragers deposit "
    "pheromone on the way back from food, and the accumulated trail biases the "
    "route choices of others without any direct message passing."
)
PASSAGE_CONSENSUS = (
    "Distributed consensus lets agents agree on a value despite failures and "
    "message delays. Quorum-based protocols require a majority to acknowledge a "
    "proposal before it is committed, which guarantees that any two decisions "
    "overlap in at least one honest participant. The cost is latency: every "
    "decision waits for a round trip to a majority."
)
PASSAGE_EMERGENCE = (
    "Emergence describes properties of a system that are not present in its parts. "
    "In multi-agent systems, flocking, lane formation in crowds, and division of "
    "labour all emerge from agents optimising local objectives. Emergent behaviour "
    "can be desirable or harmful, and it is generally hard to predict from the "
    "agent rules alone."
)

ROWS = [
    # fully grounded paraphrases
    (PASSAGE_SWARM, "What makes a thinking swarm robust?",
     "It stays robust because the agents are redundant, so losing a few of them "
     "usually does not change what the group achieves.", 0.95),
    (PASSAGE_STIGMERGY, "How do ants coordinate without messaging each other?",
     "They coordinate through the environment: returning foragers leave pheromone, "
     "and the growing trail nudges other ants toward the same route.", 0.95),
    (PASSAGE_CONSENSUS, "Why does quorum-based consensus add latency?",
     "Because each decision must wait for a round trip to a majority of agents "
     "before it can be committed.", 0.9),
    (PASSAGE_EMERGENCE, "Can emergent behaviour be predicted from agent rules?",
     "The passage says it is generally hard to predict emergent behaviour from the "
     "agent rules alone.", 0.9),
    # correct but terse
    (PASSAGE_SWARM, "Is swarm coordination centralised?",
     "No, it is decentralised.", 0.85),
    # partially grounded (one unsupported claim added)
    (PASSAGE_STIGMERGY, "What is stigmergy?",
     "Stigmergy is indirect coordination through the environment, and it was first "
     "formally proven optimal by Dorigo in 1992.", 0.4),
    (PASSAGE_CONSENSUS, "What do quorum protocols guarantee?",
     "They guarantee that any two decisions share at least one honest participant, "
     "and they also make the network immune to denial-of-service attacks.", 0.45),
    # loosely related / half unsupported
    (PASSAGE_SWARM, "What is a thinking swarm?",
     "A thinking swarm is a large neural network trained on many GPUs to imitate "
     "collective human reasoning.", 0.1),
    (PASSAGE_EMERGENCE, "Give an example of emergence in crowds.",
     "Lane formation in crowds is given as an example of emergent behaviour.", 0.9),
    # off-topic coherent answers
    (PASSAGE_CONSENSUS, "How do agents reach consensus?",
     "Pheromone trails accumulate over time and bias the foraging routes that "
     "other ants choose.", 0.05),
    (PASSAGE_EMERGENCE, "What is emergence?",
     "Emergence is the latency cost paid when every decision waits for a majority "
     "round trip.", 0.05),
    # keyword dumps (all words from the passage, no answer structure)
    (PASSAGE_SWARM, "What is a thinking swarm?",
     "swarm agents autonomous local interactions global decentralised rule "
     "environment redundancy collective outcome", 0.05),
    (PASSAGE_STIGMERGY, "What is stigmergy?",
     "stigmergy indirect coordination environment agent pheromone trail foragers "
     "food route choices message passing", 0.05),
    # fluent hallucination (no lexical overlap, plausible, unsupported)
    (PASSAGE_CONSENSUS, "What is distributed consensus?",
     "Distributed consensus is achieved by electing a permanent leader node that "
     "broadcasts its decisions and never needs acknowledgement from the others.", 0.1),
    # generic filler
    (PASSAGE_EMERGENCE, "What does this passage explain?",
     "This passage explains several important concepts and provides useful context "
     "about the broader topic under discussion.", 0.1),
    (PASSAGE_SWARM, "Summarise the passage.",
     "The text covers a number of relevant ideas and situates them within the "
     "wider literature on the subject.", 0.1),
    # grounded, different phrasing of the question
    (PASSAGE_STIGMERGY, "Does stigmergy involve direct communication between agents?",
     "No. Stigmergy works without any direct message passing; agents affect each "
     "other only by changing the shared environment.", 0.95),
    (PASSAGE_SWARM, "Who decides what each agent in a swarm does?",
     "No central controller does. Each agent follows its own small rule set and "
     "reacts to nearby agents and the environment.", 0.9),
    # partially correct answer to a factual question
    (PASSAGE_CONSENSUS, "What is required before a proposal is committed?",
     "A majority of agents must acknowledge it first.", 0.9),
    (PASSAGE_EMERGENCE, "Is emergent behaviour always beneficial?",
     "No, the passage notes it can be desirable or harmful.", 0.9),
    # answer that contradicts the passage
    (PASSAGE_SWARM, "How does a swarm handle the loss of individual agents?",
     "Losing even one agent typically causes the whole swarm to fail, since every "
     "agent has a unique irreplaceable role.", 0.05),
    (PASSAGE_STIGMERGY, "When do foragers deposit pheromone?",
     "They deposit it only on the way toward the food, never on the way back.", 0.1),
    (PASSAGE_CONSENSUS, "What is the benefit of quorum overlap?",
     "It ensures any two committed decisions share at least one honest participant.", 0.9),
    (PASSAGE_EMERGENCE, "Where does division of labour come from in multi-agent systems?",
     "It emerges from agents optimising their own local objectives.", 0.9),
]


def main() -> int:
    parser = argparse.ArgumentParser()
    parser.add_argument("--out", required=True)
    args = parser.parse_args()
    with open(args.out, "w") as handle:
        for passage, question, answer, label in ROWS:
            handle.write(json.dumps({
                "question": question, "answer": answer,
                "passage": passage, "label": label,
            }) + "\n")
    print(f"wrote {len(ROWS)} hand-authored OOD rows to {args.out}")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
