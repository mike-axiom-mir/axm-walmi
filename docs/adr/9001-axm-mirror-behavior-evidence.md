# ADR 9001: Keep AXM/Mirror behavior evidence downstream of WALDO BOMs

Status: experimental fork decision

## Context

WALDO already records inspectable lineage from corpus selection through training,
model, and release artifacts. AXM/Mirror research explores a different boundary:
what evidence should remain attached after a model is placed in a concrete
interaction, given context, permissions, tools, verification, disagreement, and
an observed outcome.

The public AXM Mirror branch treats machine state and human rendering as views of
one trace, preserves unknown/contradictory states, keeps generated output as a
proposal until verified, and forbids learned weights from rewriting roots,
permissions, evidence, or release gates.

## Decision

Add a fork-only behavior-evidence record *after* WALDO's existing BOM boundary.
Do not redefine corpus, run, model, or release BOMs in this experiment.

The v0.1 record:

- binds to WALDO model or release BOM identity by SHA-256;
- stores only digests for context, request, and output by default;
- preserves verification states and dissent separately rather than producing a
  trust or intelligence score;
- requires explicit permission and outcome states;
- requires all execution, training, promotion, canon, and world-action authority
  fields to remain false;
- seals the exact record with SHA-256 so later mutation is detectable.

Sealing proves record identity only. It does not prove that the model is safe,
correct, conscious, aligned, legally usable, or independently verified.

## Consequences

This keeps the experiment additive and reviewable. WALDO remains the source and
model-lineage system; AXM/Mirror can investigate downstream behavior provenance
without silently changing WALDO's contracts.

The first CLI is deliberately separate (`waldo-axm-mirror`) rather than adding a
new upstream `waldo` command. Integration can be reconsidered only after the
record, tests, and local experiment produce useful evidence.
