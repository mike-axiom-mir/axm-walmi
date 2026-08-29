# AXM WALDO experiment v0.35 — live neural grounding runtime

Challenge: `NEURAL_OUTPUT_IS_NOT_GROUNDING_TRUTH`

Parent checkpoint: v0.34 `66cbe8700542b249c8e7a97f75a971affa85ea83`.

## What changes in this rung

This is the first rung in the current reasoning ladder that intentionally changes WALDO's **real model-chat runtime path**, rather than adding only a detached deterministic simulator.

Existing path:

`model chat -> inference.Open -> MLX/PyTorch Session.Generate`

v0.35 experimental path:

`model chat -> inference.OpenAXMHybrid -> existing inference.Open -> existing MLX/PyTorch Session -> AXMHybridSession -> same neural Session.Generate -> deterministic response grounding`

The MLX/PyTorch implementations and workers remain unchanged.

## Default and fallback

On this experimental branch, `model chat` is routed through `OpenAXMHybrid` by default.

Explicit raw fallback:

```bash
WALDO_AXM_HYBRID=raw waldo model chat <model> "prompt"
```

`off`, `0`, and `false` are equivalent raw-fallback values.

With the hybrid wrapper active, interactive chat exposes the integration through the backend label, for example:

`PYTORCH+AXM-HYBRID-V0.35`

## Grounding input

An optional strict JSON grounding input can be supplied through:

```bash
WALDO_AXM_GROUNDING_FILE=/path/to/grounding.json waldo model chat <model> "prompt"
```

Example:

```json
{
  "schema": "axm.waldo.live-neural-grounding/v0.35",
  "state": "UNCERTAIN",
  "consequence": "LOW",
  "references": [
    {
      "id": "ref-1",
      "text": "A bounded current reference",
      "class": "VERIFIED_REFERENCE",
      "freshness": "CURRENT",
      "provenanceRoot": "receipt:example"
    }
  ],
  "authority": {
    "tool_execution": false,
    "training": false,
    "promotion": false,
    "canon": false,
    "world_action": false
  }
}
```

If no grounding file is supplied, the wrapper creates a closed-authority `STABLE/LOW` capsule with no external facts. It does not invent evidence.

## Hybrid behavior

- `STABLE` / LOW consequence: neural generation remains primary and the already-rendered prompt is passed to the original session unchanged.
- `UNCERTAIN` / LOW consequence: neural generation remains primary; v0.35 records the unresolved grounding state at the response boundary without rewriting the model-specific prompt.
- `UNCERTAIN` or `CONFLICT` / HIGH consequence: the real neural session still produces a private candidate, but the wrapper does not stream that draft; it hashes the candidate and returns a bounded deterministic grounding hold instead.
- Neural output is always treated as a candidate in the hybrid trace; its text is not promoted to evidence.
- The session wrapper delegates `Close` to the original runtime.

Optional hash-only trace output is **append-only JSONL** so one generation receipt does not silently erase the previous one:

```bash
WALDO_AXM_HYBRID_TRACE=/tmp/waldo-hybrid-trace.jsonl waldo model chat <model> "prompt"
```

Trace rows contain hashes and bounded runtime metadata, not raw prompts or raw neural output. The trace file is forced to mode `0600`.

## What this does and does not prove

This branch can prove, through Go tests and repository CI, that:

- the real WALDO model-chat consumer is connected to the wrapper;
- the wrapper delegates to the same `inference.Session` abstraction used by MLX/PyTorch;
- grounding input is strict and fail-closed;
- low-consequence operation keeps neural generation primary and preserves the rendered prompt exactly;
- HIGH unresolved grounding can observe a real neural candidate and recruit the deterministic fallback without leaking the withheld draft;
- raw WALDO inference remains explicitly recoverable;
- optional hybrid traces are append-only, hash-only JSONL receipts.

It **cannot** prove neural+deterministic quality improvement without running an actual WALDO model on a compatible host. No model weights/runtime are available inside the current build environment.

It is also not token-level continuous coupled reasoning yet. The real integration point in v0.35 is the **response boundary**. Template-aware grounding injection must happen earlier, before `model.Interaction.Prompt` finishes rendering special tokens; v0.35 intentionally does not fake that by editing a rendered prompt.

## Next measurement

The first real A/B should use the same local model, prompts, seed/options, and evidence packets:

1. raw `inference.Session`;
2. v0.35 hybrid session;
3. adversarial noisy/stale/conflicting grounding packets;
4. cases where grounding is correct but becomes stale;
5. cases where neural output catches a real change missing from deterministic references.

Measure answer correctness where a verifiable oracle exists, false grounding, missed novelty, uncertainty preservation, latency/token overhead, deterministic hold rate, and recovery after stale references.

Do not tune the runtime from the held-out A/B before freezing a development policy.

## Authority

`neural output != evidence != grounding truth != permission != execution != promotion != CANON`

**AXM pokes and logs.**
