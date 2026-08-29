# WALDO v0.52 — Dual Neural Connection Sockets

Status: **EXPERIMENTAL / TEST**

This experiment adds two provider-neutral optional neural connection sockets behind the Mirror/WALDO body.

## Shape

- `NEURAL_A`
- `NEURAL_B`

Both are disconnected by default. They are not additional permanent identities, brains, agents, or authority holders. A host connects an exact adapter/model identity to a socket, then Mirror, WALDO, or a triggered Hermes outer-eye role may request one bounded neural contribution when another neural perspective is useful.

## Why two sockets

Two sockets allow controlled neural diversity without turning every task into multi-model orchestration. They can be used for challenge, creative variants, domain second opinions, plan review, discovery, or specialist spot assistance.

The two sockets may point to different providers/models, the same provider with different models, or remain empty.

## Provider-neutral boundary

The socket fabric does not know providers by name. A connection binds adapter identity/version/digest, model identity/version, declared capabilities, context-window ceiling and output-token ceiling.

## Request/result boundary

Every request names one requester (`MIRROR`, `WALDO`, or `HERMES`), one purpose, one task reference, one public packet and capability/output ceilings. Each request allows exactly one call.

No private chain-of-thought, scratchpad, hidden reasoning, logits or internal activations are accepted into the socket contract.

Every result returns as `UNTRUSTED_NEURAL_RESULT` candidate data. It grants no identity merge, memory promotion, tool execution, workspace mutation, permission, install, deploy, merge, promotion or CANON authority. Disconnect/reconnect invalidates stale prior requests.

## Later integration

The specialist-team fabric can request a socket when a genuinely different neural perspective is useful. A determinization steward can then inspect the visible episode and propose recurring mechanical work as a deterministic fast path.

## Verification

`node shared/neural-connection-sockets/selftest.js`

Expected checkpoint: **14/14 PASS**.
