# AXM WALDO experiment v0.36 — real model A/B gate

Challenge: `A_RUNTIME_SEAM_IS_NOT_A_QUALITY_WIN`

Parent checkpoint: v0.35 `1e555aa33a026315d5e4d7c55b50709d624741eb`.

Status: **development run completed and frozen; fresh held-out cases have not been seen**.

## Why this rung exists

v0.35 connected WALDO's real `model chat` path to a reversible deterministic response-boundary wrapper. That proves the runtime seam and its contracts. It does not prove that raw WALDO plus the wrapper answers better.

This rung performs the next honest measurement:

1. raw WALDO with `WALDO_AXM_HYBRID=raw`;
2. the same WALDO binary/model/prompt/options through v0.35 hybrid mode;
3. balanced raw-first and hybrid-first pair order;
4. public synthetic development cases first;
5. a hash-bound freeze before any fresh held-out cases are introduced.

No model, backend, tokenizer, worker, prompt template, permission boundary, or v0.35 runtime code is changed here.

## Critical v0.35 measurement boundary

v0.35 applies grounding at the `RESPONSE_BOUNDARY`.

- `STABLE`, or unresolved `LOW`: neural generation remains primary and should pass through.
- unresolved `HIGH`: the neural candidate is privately hashed and replaced by a deterministic HOLD.

Therefore v0.35 does **not** yet fact-correct a neural answer or inject grounding into the rendered prompt. This A/B measures:

- raw/hybrid passthrough fidelity in LOW cases;
- correct HIGH answers suppressed by a hold;
- incorrect HIGH answers withheld from exposure;
- unexpected or missed holds;
- recovery after stale grounding becomes current;
- latency overhead;
- model answer correctness where the synthetic oracle is exact.

A HOLD is never counted as a correct answer. No composite "better" score is invented.

## Development families

The checked-in development fixture covers:

- stable references;
- noisy evidence;
- stale evidence;
- conflicting evidence;
- synchronized false corroboration from one root;
- independent-but-wrong references;
- verified-but-wrong references;
- genuine changes missing from deterministic references;
- rapid changes;
- missing or late consequence evidence;
- stale-reference recovery.

The grounding packet is deliberately not placed in the prompt. Doing so would test a different, not-yet-implemented architecture.

## Validate without a model

No third-party Python packages are required.

```bash
cd research/v0.36-real-model-ab
python3 run_ab.py --validate-only
python3 -m unittest -v test_run_ab.py
```

## Reproduce development on a compatible host

The host must already have:

- the v0.35 WALDO binary from the parent checkpoint;
- an existing local WALDO model compatible with MLX or PyTorch;
- the same model available for both modes.

The runner itself does not install or train anything. It refuses to overwrite an existing output directory.

```bash
python3 run_ab.py \
  --waldo /path/to/waldo \
  --model YOUR_EXISTING_MODEL_NAME \
  --output /private/path/waldo-v036-development
```

Defaults are intentionally bounded: seed `36001`, temperature `0`, top-p `1`, max tokens `64`, and a 15-minute per-command timeout.

The private output contains prompts and model outputs. `PUBLIC-SUMMARY.json` contains metrics and hashes without raw prompt/output text. A successful development run also creates `preheldout-freeze.json`.

For the recorded smoke-model run, `github_development_ab.sh` builds WALDO, trains the repository's tiny real PyTorch fixture, and invokes the same runner with max tokens `8`. Despite its historical filename, it is a host-side reproduction helper and does not require GitHub Actions.

## Recorded development result

The 2026-08-24 run used PyTorch `2.8.0+cpu` and the repository's deliberately tiny `pytorch-smoke` model. It completed 18 raw/hybrid pairs (36 real neural generations). See `results/DEVELOPMENT-RESULT.md` and the machine-readable receipts in `results/`.

This run validates the response-boundary wrapper behavior, not answer quality. The tiny model was incorrect on every exact development oracle, so no capable-model quality claim is authorized.

## Held-out gate

Do not populate `heldout-cases.template.json` before development freezes. After the development run:

1. create a fresh, disjoint held-out case file with phase `heldout`;
2. do not change the runner, contract, scoring policy, WALDO binary, model identity, or generation options;
3. run with the development freeze:

```bash
python3 run_ab.py \
  --phase heldout \
  --cases /fresh/path/heldout-cases.json \
  --freeze /private/path/waldo-v036-development/preheldout-freeze.json \
  --waldo /path/to/waldo \
  --model YOUR_EXISTING_MODEL_NAME \
  --output /private/path/waldo-v036-heldout
```

The runner refuses held-out execution when a frozen identity differs.

## Evidence files

- `contract.json` — experiment and claim boundary.
- `scoring-policy.json` — predeclared metrics and no-composite rule.
- `development-cases.json` — public synthetic development fixture.
- `heldout-cases.template.json` — intentionally empty until development freezes.
- `run_ab.py` — real WALDO subprocess runner and freeze gate.
- `test_run_ab.py` — contract, fairness, hold-cost, recovery, and overwrite tests.

Generated run directories are private evidence and are not committed automatically.

Recorded public evidence excludes raw prompts and model outputs:

- `results/development-run.json` — hash-bound run receipt and per-case policy observations;
- `results/PUBLIC-SUMMARY.json` — redacted metrics and identities;
- `results/preheldout-freeze.json` — policy/model/binary freeze before held-out cases;
- `results/DEVELOPMENT-RESULT.md` — human-readable interpretation.

## Claim boundary

Observed only for the recorded tiny PyTorch smoke model:

- 18 completed raw/hybrid pairs;
- 9/9 exact LOW passthrough comparisons;
- 7/7 expected HIGH holds, with no missed or unexpected holds;
- 1/1 stale-reference recovery pair;
- 0/18 raw exact-oracle correctness.

Not observed:

- fresh held-out results;
- answer-content improvement from grounding;
- token-level coupled reasoning;
- general safety or superiority;
- install, merge, promotion, or CANON authority.

`runtime seam != quality win != permission != execution != CANON`

**AXM pokes and logs.**
