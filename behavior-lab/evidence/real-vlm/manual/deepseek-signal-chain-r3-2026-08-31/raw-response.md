Based on the diagram, this describes a temporal cell-identity system with a redundant state machine. Here's the core logic:

· T1 (Rosetta) defines the framework: cells, transitions, checkpoints, and time.
· T2 is the active supergraph with 3 cells (A, B, C). Only A starts ACTIVE; B and C are IDLE.
· Rules execute in parallel each step (using a snapshot to avoid cascading):
  1. If A is ACTIVE → B becomes ACTIVE (A stays ACTIVE this step).
  2. If B is ACTIVE → A becomes DONE.
  3. If B is ACTIVE → C becomes ACTIVE.
  4. If C is ACTIVE → B becomes DONE.

Execution flow (step-by-step):

Step Pre‑step state Rules applied (simultaneously) Post‑step state
0 A=ACT, B=IDLE, C=IDLE Rule 1 fires A=ACT, B=ACT, C=IDLE
1 A=ACT, B=ACT, C=IDLE Rules 1,2,3 fire (A→B already active; B→A done; B→C active) A=DONE, B=ACT, C=ACT
2 A=DONE, B=ACT, C=ACT Rules 2,3,4 fire (B→A no effect; B→C already active; C→B done) A=DONE, B=DONE, C=ACT
3 A=DONE, B=DONE, C=ACT Rule 4 fires (C→B done) A=DONE, B=DONE, C=DONE

Stable at step 3 (all DONE). The system reports stable and stops.

Payload: The exact program is encoded as ZLIB-compressed JSON with SHA256 + CRC checksums (for integrity).

In short: a deterministic, redundant cell-state machine that activates and completes cells in sequence, ending in a stable all‑DONE state.
