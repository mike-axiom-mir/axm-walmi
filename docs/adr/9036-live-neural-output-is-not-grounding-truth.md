# ADR 9036: Live neural output is not grounding truth

## Status

Experimental runtime integration only.

## Context

WALDO v0.24 through v0.34 tested coupled reasoning, grounding, provenance-root awareness, and consequence gating primarily through deterministic research harnesses. Those experiments did not insert the resulting grounding architecture into WALDO's actual model chat inference path.

The fork already contains real MLX and PyTorch inference sessions behind `inference.Session.Generate`. It also contains AXM deterministic evidence, authority, and coupled-reasoning code under `internal/axmmirror`.

The missing boundary is a reversible runtime seam between those two existing sides.

## Decision

1. Route WALDO `model chat` through an experimental session wrapper around the existing `inference.Session` interface.
2. Do not modify MLX, PyTorch, tokenizer, training, model-state, or worker internals.
3. Build grounding state deterministically from a strict typed input owned by `internal/axmmirror`.
4. Preserve the exact already-rendered model prompt in v0.35; do not prepend generic text after a model-specific chat template has been rendered.
5. Treat every neural answer as a candidate output, never evidence, permission, execution authority, promotion, or CANON.
6. Keep verified/reference items explicitly weaker than immutable truth; freshness and candidate status remain visible.
7. Preserve neural-primary behavior for stable and LOW-consequence uncertain grounding.
8. For HIGH-consequence unresolved grounding, allow the real neural session to produce a candidate privately, hash that candidate, then withhold it and return a deterministic `HOLD`.
9. Default the experimental v0.35 branch to the hybrid wrapper while preserving `WALDO_AXM_HYBRID=raw` (also `off`, `0`, or `false`) as the unchanged neural fallback.
10. If `WALDO_AXM_GROUNDING_FILE` is absent, use a closed-authority empty grounding capsule rather than inventing facts.
11. An explicitly supplied grounding file fails closed on schema drift, unknown fields, unsupported states, duplicate identities, or authority widening.
12. Optional traces contain hashes and bounded runtime metadata rather than raw prompts by default.
13. This integration proves a real runtime connection only. It does not prove that neural + deterministic behavior is better until a real model A/B run is performed.

## Consequence

The WALDO neural runtime and the AXM deterministic grounding layer now share a real model-chat execution path at the response boundary. The integration remains reversible and does not fork the model backends themselves.

This is not yet token-level continuous coupled reasoning and v0.35 deliberately does not inject grounding text into an already-rendered model-specific prompt. The real neural result is observed as a non-authoritative candidate; deterministic grounding may pass it through or replace a HIGH-consequence unresolved candidate with a bounded hold.

## Boundary

`neural output != grounding truth != permission != execution != promotion != CANON`

No merge, installation, production default, or general quality claim is implied by this experimental branch.
