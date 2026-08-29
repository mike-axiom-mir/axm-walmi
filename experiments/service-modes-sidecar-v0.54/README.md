# WALDO Service Modes sidecar — v0.54 base

Status: **EXPERIMENTAL / TEST**

This side branch starts from WALDO v0.54 without touching the incoming local continuation branch.

## Goal

A user supplies an overall **service subject**. The Service Mode Composer deterministically derives a bounded service package:

`subject -> service mode -> relevant connected app capabilities -> Mirror/WALDO specialist views -> Hermes trigger policy -> optional neural-socket purposes`

The composer does not call an app, activate a connection, grant permissions, invoke a model, or execute a tool.

## Connected app model

The host supplies provider-neutral app descriptors:

- stable app id
- display label
- connection state
- capability tags

Only `CONNECTED` apps can satisfy a service capability. `AVAILABLE`, `DISABLED`, and `UNKNOWN` are never silently treated as connected.

A connection is not authorization. The plan contains both a minimal primary set that can cover required capability groups and all other connected apps relevant to the service.

## Initial service modes

- Software Creation
- Research & Analysis
- Communication & Outreach
- Scheduling & Administration
- Data Analysis
- Design & Media Creation
- Project Delivery
- Travel Planning
- General Service fallback

Each mode declares required/optional app-capability groups, explicit Mirror and WALDO specialist profiles, Hermes trigger reasons, and optional neural-socket purposes.

## Specialist integration

Service profiles compile through the existing Specialist Library and the existing 5-seat Mirror / 5-seat WALDO ephemeral team fabric. They therefore retain the current specialist lifecycle:

- temporary by default;
- no permission grant from the specialist package;
- no automatic memory promotion;
- controller-owned pin-lock behavior remains a separate lifecycle decision.

Hermes remains trigger-only. A service mode can declare why Hermes may be useful but does not wake Hermes merely because the mode was selected.

## Neural integration

The two neural sockets may be declared connected by the host. The service mode may suggest bounded purposes such as `CHALLENGE`, `PLAN_REVIEW`, or `CREATIVE_VARIANT`, but a suggestion never performs a neural call and never merges identity or memory.

## Truth boundary

Service Mode is composition, not authority.

It may report a degraded state when a required app capability or specialist profile is unavailable. It does not invent a connection, downgrade requirements silently, or replace a missing app with an unrelated tool.

This sidecar is intentionally independent so it can later be rebased or grafted onto the new local WALDO PR after that work lands.
