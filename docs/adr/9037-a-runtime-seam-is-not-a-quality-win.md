# ADR 9037: A runtime seam is not a quality win

## Status

Experimental evaluation contract only.

## Context

WALDO v0.35 wraps the real MLX/PyTorch-backed `inference.Session` used by `model chat`. Its response-boundary membrane preserves LOW-consequence neural output and withholds unresolved HIGH-consequence output. It does not inject grounding into the rendered prompt or rewrite the neural answer.

The next claim cannot be established by more deterministic simulator runs. It requires a raw-versus-hybrid comparison using the same real model and generation conditions.

## Decision

1. Keep the v0.35 runtime frozen.
2. Run raw and hybrid modes through the same WALDO binary, model identity, prompt, seed, temperature, top-p, and token limit.
3. Change only the three v0.35 environment controls required to select mode, case-bound grounding, and trace output.
4. Balance raw-first and hybrid-first pair order.
5. Score exact synthetic oracles, passthrough, holds, recovery, and duration separately.
6. Never count a HOLD as a correct answer.
7. Report correct HIGH answers suppressed separately from incorrect HIGH answers withheld.
8. Freeze the runner, contract, scoring policy, model identity, WALDO binary, options, and development result before fresh held-out cases exist.
9. Store raw prompts/outputs only in the private run directory; publish a hash-bound summary without those texts.
10. Do not create a composite "better" score without a separately declared cost model.

## Consequence

This package can measure the real behavior v0.35 actually implements. It cannot demonstrate content correction because content correction is not present at this rung. A promising result may justify a later, separate template-aware integration before `model.Interaction.Prompt` finishes rendering.

## Boundary

`A_RUNTIME_SEAM_IS_NOT_A_QUALITY_WIN`

No model result, quality claim, merge, install, promotion, permission, execution, or CANON authority is created by this decision.
