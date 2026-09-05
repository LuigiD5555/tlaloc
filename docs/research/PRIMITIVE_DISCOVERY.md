# Primitive discovery — Tlaloc research direction

## Goal

Given verified behavior and Episode corpora, identify bounded operations or recurring motifs that are reusable across tasks and can be qualified independently.

## Initial source of primitives

T2 should begin with a deliberately small hand-defined candidate vocabulary so the experiment can test composition before attempting autonomous discovery.

Later Tlaloc may propose additions/removals based on evidence such as:

- repeated use across task families;
- stable input/output contracts;
- held-out success;
- lower cost than general-model execution;
- clear failure boundaries;
- successful ablation/comparison.

## Principle

The objective is not to maximize the number of Tlaloques. It is to find the smallest useful basis of capabilities that composes into a substantially larger behavior space.
