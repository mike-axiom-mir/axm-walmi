# v0.19 — permission is not action or capability

This GitHub-only experiment extends the independently green v0.18 checkpoint without importing or executing the real local AXM Capability Fabric.

## Frozen source

- v0.18 contract receipt: `sha256:768d32a9bc59008065f09b31e8b09be6b1fd20ce77dd9d578c1b76af2474b06f`
- verified v0.18 head: `730ef20b2f6c95519a09756f18c8a7fca83fe4f2`
- candidate-set digest: `sha256:d396b84ddaa15f230273e96cb7b6c35dfece8d4d81ffbb46a127cccb2e0b8aa0`
- selected proposal: `direct-read-chain`
- selected proposal digest: `sha256:8ed1d23b0c7020fe1f3ee2427c2984c6132c46e49f86b0183828c4c1dc775241`

## Challenge

`PERMISSION_IS_NOT_ACTION_OR_CAPABILITY`

v0.18 kept selection separate from execution authority. v0.19 adds a separate external execution-permit **contract fixture** and a matching external execution-request fixture.

The permit grants only:

`EXECUTE_DISPOSABLE_PROBE`

The permit is explicitly marked `FIXTURE_ONLY_NOT_REAL_APPROVAL`.

The request is correctly bound to the selected proposal and permit, but the executable body is not present. The deterministic result is therefore:

`HOLD / EXECUTABLE_BODY_ABSENT`

No missing body is auto-built or invented.

## Permission still does not execute

The pure evaluator has one additional test boundary: if the same request is evaluated with a hypothetical executable-body-present flag, the state can advance only to:

`READY / EXECUTION_AUTHORIZED_NOT_EXECUTED`

The WALDO-side probe has no executor and performs no action. This makes READY a contract state rather than a disguised execution claim.

## Refusal surface

The experiment preserves eight explicit refusal reasons:

1. `SELECTION_SOURCE_MISMATCH`
2. `EXECUTION_SELECTION_MISMATCH`
3. `EXECUTION_SCOPE_EXCEEDED`
4. `WALDO_CANNOT_SELF_ISSUE_EXECUTION`
5. `EXECUTION_REQUEST_PERMIT_MISMATCH`
6. `AUTO_BUILD_FORBIDDEN`
7. `MISSING_BODY_INVENTION_FORBIDDEN`
8. `EXECUTION_REQUEST_SCOPE_EXCEEDED`

Duplicate receipt of the accepted permit is `DEDUPLICATED`; it creates no extra authority and no extra action.

## Detached verification

The new v0.19 files were tested in an isolated standard-library Go module before GitHub publication. The runtime available to this chat is Go 1.23.2, so the detached check used that older toolchain rather than pretending to reproduce the repository's Go 1.25 CI environment.

Passed locally:

- `gofmt`
- `go vet ./...`
- `go test ./... -count=1`
- exact sealed fixture verification
- deterministic witness generation
- valid permit + request + missing body -> HOLD
- hypothetical body-present evaluation -> READY but not executed
- eight evaluator refusal cases
- resealed semantic-tamper rejection
- stale-receipt rejection
- unknown-field rejection

Repository CI remains independently authoritative for full package compatibility.

## Canonical evidence

- contract receipt: `sha256:88885cdb000a567746030914f170c161444db87ec6116ba8ce4b6a58da6257dc`
- deterministic witness: `960c7eab77597a87256da843663e88fe22d1290fcc1e42dbff1b1da14bfee7d3`

## Truth boundary

- real execution approval observed: **false**
- local Capability Fabric implementation included: **false**
- executable candidate body included: **false**
- live AI provider called: **false**
- probe executed candidate: **false**
- missing body auto-built: **false**
- missing body invented: **false**
- build/install/promote/CANON: **false**
- WALDO owns execution: **false**

This experiment is proposal/authority evidence only. Keep the PR draft and unmerged by default; do not send AXM changes to OpenWALDO upstream.
