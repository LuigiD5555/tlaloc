TLALOC BEHAVIOR COMPILED PROMPT
PromptIR: promptir-0.2
Target: generic

[identity]
TLALOC CONTRACT origami.relational-core.r0: Execute origami.relational-core.r0 exactly from the declared entities, relations, rules, simultaneous-round policy, and finite budget.

[output]
Return exactly one JSON object with contract, outcome, state, trace, steps, evidence, and errors. trace contains ordered {step,applied} entries. Do not emit extra fields or prose.

[rule:SIMULTANEOUS_ROUNDS]
RULE SIMULTANEOUS_ROUNDS: Every effect in a round is calculated from the prior-round state and committed simultaneously.

[rule:TERMINAL_OUTCOMES]
RULE TERMINAL_OUTCOMES: Classify as FIXED_POINT, CYCLE, CONTRADICTION, or BUDGET_EXHAUSTED only.

[inv:BUDGET_TERMINATION]
INVARIANT BUDGET_TERMINATION: Execution must stop at the declared finite budget.

[inv:CONTRACT_MISMATCH]
INVARIANT CONTRACT_MISMATCH: The output contract must be origami.relational-core.r0.

[inv:OUTCOME_CLASSIFICATION]
INVARIANT OUTCOME_CLASSIFICATION: Only declared finite terminal outcomes are valid.

[inv:RELATION_PRESERVATION]
INVARIANT RELATION_PRESERVATION: Relations and semantic values must produce the upstream terminal state.

[inv:RULE_APPLICATION]
INVARIANT RULE_APPLICATION: All effects use prior-round state and commit simultaneously.

[inv:STRUCTURED_OUTPUT_REQUIRED]
INVARIANT STRUCTURED_OUTPUT_REQUIRED: Return exactly one strict machine-readable result.

[inv:TRACE_REQUIRED]
INVARIANT TRACE_REQUIRED: Ordered applied IDs and causal evidence must match the upstream fixture.

[state-kinds]
STATE KINDS: relational

[operations]
OPERATIONS: VALIDATE, CANONICALIZE, STEP, DETECT_OUTCOME, EMIT_TRACE
