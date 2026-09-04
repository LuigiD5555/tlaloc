// Package tonalt1arms executes the frozen TONAL T1 experiment against a live
// vision-language model. It orchestrates Arm A (monolithic Parrot), Arm B
// (Parrot-centric DAG), and Arm C (heterogeneous Tonal) across 60 frozen
// workflows, recording every model call and result deterministically.
//
// This package owns execution and result-freezing only. It does not own
// allocation, gold-generation, or experiment design — those are frozen,
// read-only inputs under experiments/tonal-t1/d4/.
package tonalt1arms
