# AXM WALDO experiment v0.32 — Code Fabric grounding binding

Challenge: `SPECIALIST_OUTPUT_IS_NOT_GROUNDING_AUTHORITY`

Parent checkpoint: v0.31 `dbd7a9ad7f098943f383f21f28fe69265df817fe`.

Exact Platform donor: cumulative PR #50 -> #55 head
`24031c227f42ea179489bfa0f5ee7709d02965be`.

## Purpose

Bind WALDO's Grounding Membrane to the real tested Code Capability Fabric
output contracts without copying Platform execution logic into WALDO.

The donor surface includes the 102 language organs / specialist eyes,
5,100 machine-native grammar heuristics, 102 template banks with 1,224
verified-mechanic recipes, and 20 first-class discipline lenses. The exact
interface blob identities and recorded Platform gate runs are in
`donor-receipt.json`.

WALDO does **not** execute copied Code Fabric code in this experiment. It
witnesses typed donor outputs and translates them into grounding packets.

## Translation boundary

Accepted donor schemas in this slice:

- `axm.code-native-discovery-seam-report/v1`
- `axm.code.human-discipline-route.v1`
- `axm.code.discipline-grammar-intersection.v1`
- `axm.code.template-selection.v1`

The experiment also defines one separate current-verifier observation schema
for testing the crossing from candidate/context into stronger grounding:

- `axm.waldo.current-verifier-observation/v0.32`

Rules:

- `DISCOVERY_CANDIDATE` -> uncertainty/candidate, never plan authority.
- `WEAK_SIGNAL` -> uncertainty, never plan authority.
- `NATIVE_REVIEW` -> coupled review context.
- discipline lens output -> context, not proof.
- grammar x discipline intersection -> review direction, not proof.
- `VERIFIED_VAULT` template -> verified mechanic reference, not arbitrary-source correctness.
- Pattern Nursery candidate -> unverified candidate.
- only a **current, provenance-bound verifier observation** may emit
  `PASS_CHANGE` into coupled reasoning.
- stale or provenance-unbound verification is damped to uncertainty.
- `PASS_CHANGE` is still reasoning evidence, not execution permission.

## Verification

`python3 -m unittest -v test_probe.py`

Observed locally before publication: **14/14 PASS**.

The tests include fail-closed authority widening, unknown donor states/schemas,
stale verifier handling, unbound verifier handling, and the unchanged
ExecutionGuard permit/environment/capability boundary.

## Authority

`capability != specialist output != grounding != reasoning effect != permission != execution != promotion != CANON`

Every adapter packet has authority `NONE`.

No PR, merge, install, promotion, or CANON change is implied by this branch.

**AXM pokes and logs.**
