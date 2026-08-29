# WALDO v0.51 — website creation loop probe

Status: **EXPERIMENTAL / TEST**

This experiment runs the exact human brief `Create a website.` through WALDO's vendored Creation Fabric without allowing a substitute model or working-chat-authored website source.

The observed path is:

1. the route atlas selects `web-application`;
2. deterministic role preferences bind `document-ui` to HTML and `browser-runtime` to JavaScript;
3. the Creation Fabric produces `CREATION_SESSION_PLAN_READY_NO_EXECUTION`;
4. the run holds at `HELD_WALDO_SOURCE_CANDIDATE_ABSENT` because no WALDO neural source candidate was supplied;
5. the exact-byte candidate writer is ready but is not invoked without source bytes.

No website is claimed. Filling the source gap with another model or a hand-written template would invalidate this benchmark.

## Candidate writer

`waldo-axm-mirror apply-candidate <request.json> <workspace-root> <receipt.json>` accepts a hash-bound create/update/delete transaction. It preflights the whole request, rejects unsafe paths and symlinks, checks stale hashes and byte limits, applies exact supplied bytes, and emits a receipt declaring that the writer did not author the code.

## Experience and benchmark result

This was a real local execution observation, not a simulated success. It is retained as `OBSERVED_EXECUTION_TRACE` / `MEMORY_ONLY` so the exact capability gap can reach later WALDO context. The benchmark is `INFRASTRUCTURE_BLOCKED`: route and planning passed, while source, render, verification, and repair could not run because this machine had no WALDO neural source candidate or trainable WALDO checkpoint.

The failed attempt is not a positive answer to imitate. v0.51 adds a separate outcome-conditioned trajectory projection: the user brief remains the user turn, the failed attempt and observed outcome become tool evidence, and the visible lesson is the only supervised assistant turn. This lets incomplete, harmful, or inconclusive attempts teach WALDO without requiring completion and without rewarding the failed action.

The v0.51 run produced a verified `OUTCOME_CONDITIONED_REFLECTION` projection for normal `assistant-response-modeling`. Actual weight mutation still requires an installed training backend and a trainable WALDO checkpoint; neither was present in this machine run.

## Authority

No network, install, deployment, merge, promotion, CANON, or model-weight authority is granted.
