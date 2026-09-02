# P0_AUDIT — parrot-capability-r0

| base_id | family | pages | evidence | address | text | image | validation | human |
|---|---|---|:-:|:-:|:-:|:-:|---|---|
| e2e-entity-01 | entity | [28] | ✓ | ✓ | ✓ | ✓ | PASS | — |
| e2e-factual-01 | exact | [31] | ✓ | ✓ | ✓ | ✓ | PASS | — |
| e2e-factual-02 | exact | [37] | ✓ | ✓ | ✓ | ✓ | PASS | — |
| e2e-locate-01 | choice | [50] | ✓ | ✓ | ✓ | ✓ | PASS | — |
| e2e-locate-02 | choice | [37] | ✓ | ✓ | ✓ | ✓ | PASS | — |
| e2e-locate-03 | choice | [66] | ✓ | ✓ | ✓ | ✓ | PASS | — |
| e2e-locate-04 | choice | [38] | ✓ | ✓ | ✓ | ✓ | PASS | — |
| e2e-locate-05 | choice | [51] | ✓ | ✓ | ✓ | ✓ | PASS | — |
| e2e-numeric-01 | numeric | [30] | ✓ | ✓ | ✓ | ✓ | FAIL: the question leaks the answer | — |
| e2e-numeric-02 | numeric | [28] | ✓ | ✓ | ✓ | ✓ | PASS | — |
| e2e-numeric-03 | numeric | [30] | ✓ | ✓ | ✓ | ✓ | PASS | — |
| e2e-numeric-04 | numeric | [28] | ✓ | ✓ | ✓ | ✓ | FAIL: cloze answer appears 3 times in the page (ambiguous) | — |
| e2e-numeric-05 | numeric | [30] | ✓ | ✓ | ✓ | ✓ | PASS | — |
| e2e-synthesis-01 | choice | [30] | ✓ | ✓ | ✓ | ✓ | PASS | — |
| e2e-synthesis-02 | choice | [28] | ✓ | ✓ | ✓ | ✓ | PASS | — |
| e2e-synthesis-03 | choice | [30] | ✓ | ✓ | ✓ | ✓ | PASS | — |
| e2e-synthesis-04 | choice | [30] | ✓ | ✓ | ✓ | ✓ | PASS | — |

## Summary

```
BASE QUESTIONS    17
TEXT VARIANTS     17
IMAGE VARIANTS    17
TOTAL RECORDS     34

locate       5
entity       1
factual      2
numeric      5
synthesis    4

validation failures    2
missing evidence       0
missing address        0
human rejected         0
```

**GATE: NOT GREEN** — do not freeze. Need 30 base questions, have 17. Categories are not 6/6/6/6/6. Fix the failing rows above. Never force questions to hit a quota — if a category cannot be filled naturally, change the source.
