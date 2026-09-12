# ADR 0065: Make worker output termination explicit

- Status: accepted
- Date: 2026-09-10

## Context

Schema-1 worker output frames were validated independently, but the stream had
no central transition contract. A worker could therefore emit a valid
completion followed by another event or error. The generic reader delivered
both frames, so a subprocess adapter could report checkpoint or evaluation
progress after it had already observed completion. An event-only stream could
also reach EOF without the protocol reader classifying the missing terminal
state.

Per-adapter checks are not a sufficient authority for this rule. Every present
and future consumer reads the same protocol and must observe the same terminal
boundary before it can persist progress or completion evidence.

## Decision

The schema-1 output reader owns a small deterministic stream state machine:

- the open state accepts zero or more `event` frames;
- the first `complete` or `error` frame moves the stream to a terminal state;
- no structured frame is valid after a terminal frame; and
- EOF is valid only after exactly one terminal frame has been admitted.

The reader validates a frame before applying the transition, and it rejects an
illegal post-terminal frame before invoking the consumer. Unstructured worker
diagnostics remain tolerated and are routed to the existing skipped-output
sink; they do not become protocol state. A consumer can still turn a valid
terminal error frame into its domain-specific error.

Completion payload shape remains the responsibility of the existing frame
validator. Lifecycle-specific checks, artifact verification, run-state
persistence, and promotion authority remain outside the wire reader.

## Consequences

- All adapters share one authoritative terminal-order contract.
- Post-completion progress cannot reach a consumer through this reader.
- Truncated event-only output fails closed at the protocol boundary.
- Existing schema-1 workers that emit one terminal frame remain compatible.
- The reader proves stream order, not process authorship, artifact truth, or
  that durable lifecycle writes happen atomically.
