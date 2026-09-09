# AXM Local Workshop v0.1

A deliberately small local workspace for Waldo first and additional AI identities later.

## What exists now

- Local browser chat workspace.
- Waldo is the seeded identity; sessions are separate.
- Native Waldo model bridge inside the workshop process.
- External OpenAI-compatible model endpoints remain an override for later models and vision adapters.
- A goal field per session.
- A real opt-in background goal runtime per session.
- Start / pause controls and selectable heartbeat interval.
- Background heartbeats write visible progress notes into the session.
- No overlapping heartbeat calls for the same session.
- Foreground messages retain visible pending, failed, and answered delivery state.
- Failed foreground messages can be retried without duplicating the user turn.
- Runtime state survives process restarts; active sessions resume when the workshop is running again.
- Windows and Linux user auto-start installers.
- Explicit consent requests with approve/reject history.
- Three memory layers:
  - session memory;
  - AI identity memory;
  - shared vault memory.
- Creative Room image intake and per-session gallery.
- Images can be attached to chat requests for vision-capable endpoints.
- Local JSON persistence plus local media files.

The workshop is intentionally not an IDE, giant agent framework, vector database, or security platform. The workspace owns state, memory, consent, media, and heartbeat scheduling; the selected model runtime generates replies.

## Run Waldo natively

From the repository root:

```bash
AXM_AI_MODEL=waldo go run ./cmd/workshop
```

Then open:

```text
http://127.0.0.1:7788
```

When `AXM_AI_BASE_URL` is not set, the workshop starts its own small local compatibility bridge on `127.0.0.1:7789` and routes chat into WALDO's real local model artifacts through `inference.OpenAXMHybrid`. The model name defaults to `waldo` and is resolved from WALDO's configured model root (normally beneath `~/.waldo/models`).

The bridge is not a second AI installation. It lives in the same workshop binary so chat and background heartbeats can use native WALDO inference without keeping another server command running.

## Use another local model endpoint

Setting `AXM_AI_BASE_URL` overrides the native Waldo bridge:

```bash
AXM_AI_BASE_URL=http://127.0.0.1:11434/v1 \
AXM_AI_MODEL=my-model \
go run ./cmd/workshop
```

This keeps the workshop usable with other local AI builds later.

Optional environment variables:

- `AXM_AI_BASE_URL` — override with an OpenAI-compatible API base URL; when omitted, native Waldo mode is used.
- `AXM_AI_MODEL` — WALDO/local model name; defaults to `waldo`.
- `AXM_AI_KEY` — optional bearer key for an external endpoint.
- `AXM_WORKSHOP_NATIVE_WALDO=off` — disable automatic native bridge startup.
- `AXM_WALDO_BRIDGE_PORT` — native bridge port; defaults to `7789`.
- `AXM_WALDO_MAX_TOKENS` — native generation limit; defaults to `512`.
- `AXM_WORKSHOP_PORT` — workshop UI port; defaults to `7788`.
- `AXM_WORKSHOP_DATA` — defaults to `~/.axm-workshop`.

## Auto-start with the computer

The installers build one `axm-workshop` binary. The native Waldo bridge and heartbeat scheduler are part of that binary.

### Windows

From PowerShell in the repository:

```powershell
.\cmd\workshop\install-autostart.ps1
```

This builds the workshop under `%LOCALAPPDATA%\AXM\LocalWorkshop` and registers **AXM Local Workshop** as a Scheduled Task at user logon. It starts the task immediately too.

Remove auto-start without deleting workshop memory/media:

```powershell
.\cmd\workshop\uninstall-autostart.ps1
```

### Linux

```bash
bash ./cmd/workshop/install-autostart.sh
```

This builds `~/.local/bin/axm-workshop` and enables a `systemd --user` service. It normally starts when the user logs in and restarts after process failures.

Remove auto-start without deleting workshop memory/media:

```bash
bash ./cmd/workshop/uninstall-autostart.sh
```

The auto-start scripts do not delete `~/.axm-workshop` or WALDO model data.

## Background goal runtime

1. Enter a goal in the session.
2. Pick a heartbeat interval: 30 seconds, 1 minute, 5 minutes, 15 minutes, or 1 hour.
3. Press **Start background**.
4. The first heartbeat is scheduled almost immediately; later heartbeats use the selected interval.
5. Each successful heartbeat adds a visible `Goal heartbeat` assistant message to the session.
6. Press **Pause** to prevent future heartbeats.

The scheduler runs one heartbeat at a time per session. If a model response takes longer than the configured interval, another request is not stacked on top of it. A failed model call is recorded visibly and retried on the next normal interval while the session remains active.

Foreground chat failures stay attached to the exact user message with the model-endpoint error and a **Retry** action. Retrying reuses that retained turn rather than adding a duplicate. If the Workshop stops while a foreground response is pending, the next start marks that turn as interrupted and retryable instead of leaving an ambiguous permanent loading state.

## Creative Room and native Waldo

The Creative Room still stores and shares images with the session. The current native WALDO inference path is text generation, so it does **not** pretend to inspect image pixels. In native mode an attached image is represented to Waldo by an explicit text marker saying that the image exists but its pixels were not inspected.

If `AXM_AI_BASE_URL` points to a vision-capable OpenAI-compatible endpoint, the existing workshop request contains the image data and that endpoint can inspect it. A native WALDO vision capability can be added later without changing the Creative Room storage model.

## Storage

By default the workshop writes only to:

```text
~/.axm-workshop/
  state.json
  media/
```

Conversation history stays with its session. Explicit identity memory is visible to sessions for that identity. Vault memory is visible across identities. Chat content is not silently promoted into durable memory.

## Consent boundary

Consent items are records, not hidden execution permissions. Approving an item records the decision; it does not secretly run a tool, change a file, connect an app, or write memory.

The background runtime can think, continue a goal, and produce a progress note. It does not gain separate tool/app/file execution authority from being active.

## Verification history

Before the native bridge change, `go test ./...` and an end-to-end fake-endpoint runtime probe passed locally: goal saved, runtime activated, background heartbeat fired without a user chat request, model response was written into the session, pulse count incremented, and the next heartbeat was scheduled.

The native bridge adds focused prompt/Creative-Room tests under `waldo_native_bridge_test.go`. The branch also carries a workshop-specific CI gate so post-bridge compile/test status is evidence from the repository rather than assumed from the earlier test.

## Deliberate v0.1 limits

- Native WALDO mode requires a usable trained/downloaded local WALDO model with the selected name.
- Current native WALDO inference is text-only; image pixels require a vision-capable adapter/model.
- App connectors are not wired yet. They should enter through the existing explicit adapter/consent boundary.
- Auto-start is user-level: Windows Scheduled Task at logon or Linux `systemd --user`. It does not force machine-wide boot privileges.
- No merge is performed by this branch.
