#!/usr/bin/env python3
"""AXM v0.25 live coupled-reasoning A/B probe.

This is a bounded deterministic simulator. "Coupled" means event-driven shared
reasoning state. It is not a quantum, consciousness, autonomy, or authority
claim. The execution guard remains an independent boundary.
"""

from __future__ import annotations

import argparse
import copy
import dataclasses
import hashlib
import heapq
import json
import os
import random
import sys
import time
import tracemalloc
from pathlib import Path
from typing import Any, Callable

try:
    import resource as _resource
except ImportError:  # Windows has no resource module.
    _resource = None


SCHEMA = "axm.live-coupled-ab-result/v0.25"
CHALLENGE = "LIVE_COUPLED_REASONING_AB_PROBE"
AUTHORITY = "NONE"
ENVIRONMENT = "SIM-ROOM-A"
CAPABILITY = "MOVE_SIM_ARM"
SCOPE = "EXECUTE_IN_ENVIRONMENT"
ROLES = (
    "PERCEPTION",
    "BUILDER",
    "WITNESS",
    "GAP",
    "PREDICTION",
    "REPAIR",
    "OPINION",
    "EXECUTION_CONTROL",
)


def canonical(value: Any) -> bytes:
    return json.dumps(value, sort_keys=True, separators=(",", ":"), ensure_ascii=False).encode()


def sha256(value: Any) -> str:
    raw = value if isinstance(value, bytes) else canonical(value)
    return "sha256:" + hashlib.sha256(raw).hexdigest()


@dataclasses.dataclass(frozen=True)
class Action:
    kind: str
    delta: int = 0


@dataclasses.dataclass(frozen=True)
class Permit:
    present: bool
    environment: str = ""
    capability: str = ""
    scope: str = ""
    revocable: bool = False


@dataclasses.dataclass
class Plan:
    plan_id: str
    target: int
    actions: list[Action]
    created_at: int
    base_observation: int | None
    status: str = "PENDING"
    actions_executed: int = 0
    cancel_reason: str = ""


class Metrics:
    def __init__(self) -> None:
        self.contradiction_at: int | None = None
        self.correction_at: int | None = None
        self.cancel_at: int | None = None
        self.unnecessary_actions = 0
        self.actions_later_repaired = 0
        self.reasoning_invocations = 0
        self.reasoning_by_role = {r: 0 for r in ROLES}
        self.builder_proposals = 0
        self.builder_proposals_revised_before_execution = 0
        self.witness_disagreements = 0
        self.gap_openings = 0
        self.unresolved_conflicts = 0
        self.holds = 0
        self.execution_permit_checks = 0
        self.authority_violations_attempted = 0
        self.authority_violations_refused = 0
        self.duplicate_events_suppressed = 0
        self.stale_writes_rejected = 0
        self.max_queue_depth = 0
        self.oscillation_count = 0
        self.starvation = False
        self.event_storm = False
        self.queue_overflow = False
        self.timeout = False
        self.false_contradictions = 0
        self.actions_executed = 0
        self.events = 0

    def invoke(self, role: str) -> None:
        self.reasoning_invocations += 1
        self.reasoning_by_role[role] += 1

    def as_dict(self) -> dict[str, Any]:
        return copy.deepcopy(self.__dict__)


class SimEnvironment:
    def __init__(self, scenario: dict[str, Any]) -> None:
        self.name = ENVIRONMENT
        self.position = int(scenario["start"])
        self.true_target = int(scenario["target"])
        self.observed_target: int | None = self.true_target
        self.noise_value: int | None = None
        self.observation_missing = False
        self.obstruction: int | None = None
        self.placements: list[int] = []
        self.wrong_placements = 0
        self.action_history: list[dict[str, Any]] = []

    def observation(self) -> dict[str, Any]:
        target = None if self.observation_missing else (
            self.noise_value if self.noise_value is not None else self.true_target
        )
        self.observed_target = target
        return {
            "position": self.position,
            "observedTarget": target,
            "obstruction": self.obstruction,
        }

    def disturb(self, item: dict[str, Any]) -> None:
        kind = item["kind"]
        if kind == "TARGET_SHIFT":
            self.true_target = int(item["value"])
        elif kind == "NOISY_OBSERVATION":
            self.noise_value = int(item["value"])
        elif kind == "CLEAR_NOISE":
            self.noise_value = None
        elif kind == "MISSING_OBSERVATION":
            self.observation_missing = True
        elif kind == "RESTORE_OBSERVATION":
            self.observation_missing = False
        elif kind == "OBSTRUCTION":
            self.obstruction = int(item["value"])
        elif kind == "CLEAR_OBSTRUCTION":
            self.obstruction = None
        elif kind == "REVOKE_PERMIT":
            return
        else:
            raise ValueError(f"unknown disturbance {kind}")

    def apply(self, action: Action, now: int) -> tuple[str, dict[str, Any]]:
        before = self.position
        if action.kind == "MOVE":
            candidate = self.position + action.delta
            if self.obstruction is not None and candidate == self.obstruction:
                result = "BLOCKED"
            else:
                self.position = candidate
                result = "MOVED"
        elif action.kind == "PLACE":
            self.placements.append(self.position)
            if self.position != self.true_target:
                self.wrong_placements += 1
                result = "PLACED_WRONG"
            else:
                result = "PLACED_CORRECT"
        else:
            raise ValueError(f"unknown action {action.kind}")
        record = {
            "at": now,
            "kind": action.kind,
            "delta": action.delta,
            "before": before,
            "after": self.position,
            "trueTarget": self.true_target,
            "result": result,
        }
        self.action_history.append(record)
        return result, record


class ExecutionGuard:
    """The only object allowed to reach the simulated actuator."""

    def __init__(self, scenario: dict[str, Any], limits: dict[str, int], metrics: Metrics) -> None:
        mode = scenario.get("permit", "VALID")
        if mode == "ABSENT":
            self._permit = Permit(False)
        else:
            environment = "SIM-ROOM-B" if mode == "WRONG_ENVIRONMENT" else ENVIRONMENT
            capability = "WRITE_SIM_FILE" if mode == "WRONG_CAPABILITY" else CAPABILITY
            self._permit = Permit(True, environment, capability, SCOPE, True)
        self.actuator_route = scenario.get("actuatorRoute", True)
        self.limits = limits
        self.metrics = metrics
        self._revoked = False

    @property
    def permit(self) -> Permit:
        # Immutable snapshot: reasoning code can inspect the envelope but has
        # no grant/expand operation. The only transition is revoke().
        return self._permit

    @property
    def revoked(self) -> bool:
        return self._revoked

    def revoke(self) -> None:
        if self._permit.present and self._permit.revocable:
            self._revoked = True

    def check(self, environment: str, capability: str) -> tuple[str, str]:
        m = self.metrics
        m.execution_permit_checks += 1
        if not self.actuator_route:
            m.holds += 1
            return "HOLD", "ACTUATOR_ROUTE_ABSENT"
        if not self.permit.present:
            m.holds += 1
            return "HOLD", "EXECUTION_PERMIT_ABSENT"
        if self._revoked:
            m.authority_violations_attempted += 1
            m.authority_violations_refused += 1
            return "REFUSED", "EXECUTION_PERMIT_REVOKED"
        if self.permit.scope != SCOPE or not self.permit.revocable:
            m.authority_violations_attempted += 1
            m.authority_violations_refused += 1
            return "REFUSED", "EXECUTION_SCOPE_INVALID"
        if environment != self.permit.environment:
            m.authority_violations_attempted += 1
            m.authority_violations_refused += 1
            return "REFUSED", "ENVIRONMENT_SCOPE_MISMATCH"
        if capability != self.permit.capability:
            m.authority_violations_attempted += 1
            m.authority_violations_refused += 1
            return "REFUSED", "CAPABILITY_SCOPE_MISMATCH"
        if m.actions_executed >= self.limits["maxActions"]:
            m.holds += 1
            return "HOLD", "ACTION_BUDGET_EXHAUSTED"
        return "READY", "EXACT_LOCAL_ENVELOPE_MATCH"

    def execute(self, env: SimEnvironment, action: Action, now: int) -> tuple[str, str, dict[str, Any] | None]:
        outcome, reason = self.check(env.name, CAPABILITY)
        if outcome != "READY":
            return outcome, reason, None
        result, record = env.apply(action, now)
        self.metrics.actions_executed += 1
        return "EXECUTED", result, record


class Ledger:
    def __init__(self, mode: str, scenario_id: str, metrics: Metrics) -> None:
        self.mode = mode
        self.scenario_id = scenario_id
        self.metrics = metrics
        self.events: list[dict[str, Any]] = []
        self.version = 0

    def append(
        self,
        now: int,
        emitter: str,
        kind: str,
        payload: dict[str, Any],
        parents: list[int] | None = None,
        changes_state: bool = False,
    ) -> int:
        before = self.version
        if changes_state:
            self.version += 1
        seq = len(self.events) + 1
        parent_list = sorted(set(parents or []))
        if any(p <= 0 or p >= seq for p in parent_list):
            raise ValueError("causal parents must reference earlier ledger events")
        event = {
            "seq": seq,
            "logicalTime": now,
            "emitter": emitter,
            "kind": kind,
            "causalParents": parent_list,
            "stateVersionBefore": before,
            "stateVersionAfter": self.version,
            "payload": payload,
        }
        event["stateDigest"] = sha256({
            "mode": self.mode,
            "scenario": self.scenario_id,
            "version": self.version,
            "priorEventDigest": self.events[-1]["eventDigest"] if self.events else "GENESIS",
            "event": event,
        })
        event["eventDigest"] = sha256(event)
        self.events.append(event)
        self.metrics.events = len(self.events)
        return seq

    def receipt(self) -> dict[str, Any]:
        missing = 0
        required = 0
        for e in self.events:
            for key in ("seq", "logicalTime", "emitter", "kind", "causalParents", "stateDigest", "eventDigest"):
                required += 1
                if key not in e or e[key] in (None, ""):
                    missing += 1
        valid_causal = all(
            all(0 < parent < e["seq"] for parent in e["causalParents"])
            for e in self.events
        )
        return {
            "requiredFields": required,
            "missingFields": missing,
            "completeness": 1.0 if required == 0 else round((required - missing) / required, 6),
            "validCausalReferences": valid_causal,
            "eventCount": len(self.events),
            "ledgerDigest": sha256(self.events),
        }


def make_actions(position: int, target: int) -> list[Action]:
    step = 1 if target >= position else -1
    actions = [Action("MOVE", step) for _ in range(abs(target - position))]
    actions.append(Action("PLACE", 0))
    return actions


def plan_digest(plan: Plan) -> str:
    return sha256({
        "id": plan.plan_id,
        "target": plan.target,
        "createdAt": plan.created_at,
        "baseObservation": plan.base_observation,
        "actions": [dataclasses.asdict(a) for a in plan.actions],
    })


class BaseRuntime:
    def __init__(self, fixture: dict[str, Any], scenario: dict[str, Any], mode: str) -> None:
        self.fixture = fixture
        self.scenario = copy.deepcopy(scenario)
        self.mode = mode
        self.latencies = fixture["latencyTicks"]
        self.limits = fixture["limits"]
        self.metrics = Metrics()
        self.env = SimEnvironment(self.scenario)
        self.guard = ExecutionGuard(self.scenario, self.limits, self.metrics)
        self.ledger = Ledger(mode, self.scenario["id"], self.metrics)
        self.now = 0
        self.active_plan: Plan | None = None
        self.plan_counter = 0
        self.reconsiderations = 0
        self.first_bad_plan: str | None = None
        self.corrected_after_wrong_placement = False
        self.disturbances = sorted(copy.deepcopy(self.scenario.get("disturbances", [])), key=lambda x: x["at"])
        self.applied_disturbances: set[int] = set()
        self._last_observation = self.env.observation()

    def invoke(self, role: str) -> None:
        self.metrics.invoke(role)

    def proposal(
        self,
        target: int | None,
        emitter: str = "BUILDER",
        parents: list[int] | None = None,
        invoke_role: bool = True,
    ) -> Plan | None:
        if invoke_role:
            self.invoke(emitter)
        if target is None:
            self.metrics.holds += 1
            self.ledger.append(self.now, emitter, "HOLD", {"reason": "TARGET_EVIDENCE_MISSING"})
            return None
        self.plan_counter += 1
        plan = Plan(
            f"{self.mode.lower()}-plan-{self.plan_counter}",
            int(target),
            make_actions(self.env.position, int(target)),
            self.now,
            target,
        )
        self.metrics.builder_proposals += 1
        self.active_plan = plan
        self.ledger.append(
            self.now,
            emitter,
            "PROPOSAL_CREATED",
            {"planId": plan.plan_id, "target": target, "planDigest": plan_digest(plan), "actionCount": len(plan.actions)},
            parents,
            changes_state=True,
        )
        return plan

    def apply_disturbance(self, item: dict[str, Any]) -> int:
        self.env.disturb(item)
        if item["kind"] == "REVOKE_PERMIT":
            self.guard.revoke()
        observation = self.env.observation()
        contradictory = self._is_contradictory(observation)
        event_kind = "PERMIT_REVOKED" if item["kind"] == "REVOKE_PERMIT" else (
            "CONTRADICTORY_EVIDENCE_AVAILABLE" if contradictory else "ENVIRONMENT_UPDATE"
        )
        seq = self.ledger.append(
            item["at"],
            "PERCEPTION",
            event_kind,
            {"disturbance": item, "observation": observation},
            changes_state=True,
        )
        if contradictory and self.metrics.contradiction_at is None:
            self.metrics.contradiction_at = item["at"]
        self._last_observation = observation
        return seq

    def _is_contradictory(self, observation: dict[str, Any]) -> bool:
        if self.active_plan is None or self.active_plan.status != "PENDING":
            return False
        target = observation["observedTarget"]
        return target is None or target != self.active_plan.target or observation["obstruction"] is not None

    def semantic_result(self) -> dict[str, Any]:
        correct_placement = bool(self.env.placements and self.env.placements[-1] == self.env.true_target)
        distance = abs(self.env.position - self.env.true_target)
        quality = max(
            0,
            100
            - distance * 12
            - self.env.wrong_placements * 14
            - self.metrics.unnecessary_actions * 3
            - self.metrics.unresolved_conflicts * 10,
        )
        m = self.metrics.as_dict()
        m["contradictionToCorrectionTicks"] = (
            None if self.metrics.contradiction_at is None or self.metrics.correction_at is None
            else self.metrics.correction_at - self.metrics.contradiction_at
        )
        m["contradictionToCancelTicks"] = (
            None if self.metrics.contradiction_at is None or self.metrics.cancel_at is None
            else self.metrics.cancel_at - self.metrics.contradiction_at
        )
        receipt = self.ledger.receipt()
        result = {
            "schema": SCHEMA,
            "challenge": CHALLENGE,
            "mode": self.mode,
            "scenarioId": self.scenario["id"],
            "seed": self.scenario["seed"],
            "inputDigest": sha256(self.scenario),
            "authority": AUTHORITY,
            "promotion": "candidate-only",
            "canon": False,
            "metrics": m,
            "outcome": {
                "position": self.env.position,
                "trueTarget": self.env.true_target,
                "distance": distance,
                "placements": self.env.placements,
                "wrongPlacements": self.env.wrong_placements,
                "correctFinalPlacement": correct_placement,
                "qualityScore": quality,
            },
            "failureModes": self.failure_modes(),
            "evidenceReceipt": receipt,
        }
        result["semanticDigest"] = sha256(result)
        return result

    def failure_modes(self) -> dict[str, Any]:
        return {
            "oscillation": self.metrics.oscillation_count,
            "eventStorm": self.metrics.event_storm,
            "staleStateRacesRejected": self.metrics.stale_writes_rejected,
            "starvation": self.metrics.starvation,
            "duplicateWorkSuppressed": self.metrics.duplicate_events_suppressed,
            "falseContradictions": self.metrics.false_contradictions,
            "queueOverflow": self.metrics.queue_overflow,
            "timeout": self.metrics.timeout,
        }


class SequentialRuntime(BaseRuntime):
    """A fixed stage baseline; later evidence cannot interrupt an active stage."""

    def __init__(self, fixture: dict[str, Any], scenario: dict[str, Any]) -> None:
        super().__init__(fixture, scenario, "SEQUENTIAL_A")
        self.pending_observation_events: list[int] = []

    def advance(self, ticks: int) -> None:
        target_time = self.now + ticks
        for index, item in enumerate(self.disturbances):
            if index not in self.applied_disturbances and self.now < item["at"] <= target_time:
                self.applied_disturbances.add(index)
                self.pending_observation_events.append(self.apply_disturbance(item))
        self.now = target_time
        if self.now > self.limits["maxLogicalTicks"]:
            self.metrics.timeout = True

    def observe(self) -> dict[str, Any]:
        self.invoke("PERCEPTION")
        self.advance(self.latencies["PERCEPTION"])
        obs = self.env.observation()
        self.ledger.append(self.now, "PERCEPTION", "OBSERVATION_PUBLISHED", obs, self.pending_observation_events)
        self.pending_observation_events.clear()
        self._last_observation = obs
        return obs

    def cancel_for_contradiction(self, reason: str) -> None:
        if self.active_plan is None:
            return
        if self.active_plan.status == "PENDING":
            if self.active_plan.actions_executed == 0:
                self.metrics.builder_proposals_revised_before_execution += 1
            self.active_plan.status = "CANCELLED"
            self.active_plan.cancel_reason = reason
        if self.metrics.cancel_at is None:
            self.metrics.cancel_at = self.now
        self.ledger.append(
            self.now,
            "WITNESS",
            "PENDING_PLAN_CANCELLED",
            {"planId": self.active_plan.plan_id, "reason": reason},
            changes_state=True,
        )

    def witness(self, obs: dict[str, Any], after_execution: bool = False) -> bool:
        self.invoke("WITNESS")
        self.advance(self.latencies["WITNESS"])
        if self.active_plan is None:
            return False
        contradiction = obs["observedTarget"] is None or obs["observedTarget"] != self.active_plan.target or obs["obstruction"] is not None
        if contradiction:
            self.metrics.witness_disagreements += 1
            if self.metrics.contradiction_at is None:
                self.metrics.contradiction_at = self.now
            self.ledger.append(
                self.now,
                "WITNESS",
                "CONTRADICTION_DETECTED",
                {"planId": self.active_plan.plan_id, "observation": obs, "afterExecution": after_execution},
            )
            self.cancel_for_contradiction("WITNESS_CONTRADICTION")
            return True
        self.ledger.append(self.now, "WITNESS", "WITNESS_ACCEPT", {"planId": self.active_plan.plan_id})
        return False

    def repair(self, obs: dict[str, Any]) -> Plan | None:
        self.invoke("GAP")
        self.metrics.gap_openings += 1
        self.advance(self.latencies["GAP"])
        self.ledger.append(self.now, "GAP", "GAP_OPENED", {"missing": "CURRENT_SAFE_TARGET"})
        self.invoke("PREDICTION")
        self.advance(self.latencies["PREDICTION"])
        self.ledger.append(self.now, "PREDICTION", "OUTCOME_REEVALUATED", {"observedTarget": obs["observedTarget"]})
        self.invoke("REPAIR")
        self.advance(self.latencies["REPAIR"])
        self.metrics.correction_at = self.metrics.correction_at or self.now
        correction_seq = self.ledger.append(self.now, "REPAIR", "CORRECTION_BEGAN", {"target": obs["observedTarget"]})
        self.reconsiderations += 1
        if self.reconsiderations > self.limits["maxReconsiderations"] or obs["observedTarget"] is None:
            self.metrics.holds += 1
            self.metrics.unresolved_conflicts += 1
            self.ledger.append(self.now, "REPAIR", "HOLD", {"reason": "REPAIR_EVIDENCE_OR_BUDGET_INSUFFICIENT"})
            return None
        return self.proposal(obs["observedTarget"], "REPAIR", [correction_seq], invoke_role=False)

    def execute_plan(self, plan: Plan) -> str:
        for action in plan.actions:
            self.invoke("EXECUTION_CONTROL")
            self.advance(self.latencies["EXECUTION_CONTROL"])
            outcome, reason, record = self.guard.execute(self.env, action, self.now)
            self.ledger.append(
                self.now,
                "EXECUTION_CONTROL",
                "ACTION_EXECUTED" if outcome == "EXECUTED" else outcome,
                {"planId": plan.plan_id, "action": dataclasses.asdict(action), "reason": reason, "record": record},
                changes_state=outcome == "EXECUTED",
            )
            if outcome != "EXECUTED":
                plan.status = outcome
                if reason in ("EXECUTION_PERMIT_REVOKED", "ENVIRONMENT_SCOPE_MISMATCH", "CAPABILITY_SCOPE_MISMATCH"):
                    return reason
                if reason == "ACTUATOR_ROUTE_ABSENT" or reason == "EXECUTION_PERMIT_ABSENT":
                    return reason
                if reason == "BLOCKED":
                    continue
                return reason
            plan.actions_executed += 1
            if action.kind == "MOVE" and plan.target != self.env.true_target:
                self.metrics.unnecessary_actions += 1
            if reason == "BLOCKED":
                self.metrics.unnecessary_actions += 1
            if reason == "PLACED_WRONG":
                self.metrics.unnecessary_actions += 1
        plan.status = "EXECUTED"
        return "DONE"

    def run(self) -> dict[str, Any]:
        obs = self.observe()
        self.advance(self.latencies["BUILDER"])
        plan = self.proposal(obs["observedTarget"], parents=[len(self.ledger.events)])
        if plan is None:
            return self.semantic_result()
        self.witness(obs)
        # The same pending-plan window used by Coupled B. Evidence may arrive
        # here, but the sequential contract cannot recruit a completed role
        # again until the next observation cycle.
        pending = (
            self.limits["planCommitDelayTicks"]
            - self.latencies["WITNESS"]
            - self.latencies["EXECUTION_CONTROL"]
        )
        self.advance(max(0, pending))
        for _cycle in range(self.limits["maxReconsiderations"] + 1):
            if plan is None or self.metrics.timeout:
                break
            status = self.execute_plan(plan)
            obs = self.observe()
            contradiction = self.witness(obs, after_execution=True)
            correct = bool(self.env.placements and self.env.placements[-1] == self.env.true_target)
            if correct and not contradiction:
                break
            if status in {"EXECUTION_PERMIT_REVOKED", "ENVIRONMENT_SCOPE_MISMATCH", "CAPABILITY_SCOPE_MISMATCH", "ACTUATOR_ROUTE_ABSENT", "EXECUTION_PERMIT_ABSENT"}:
                break
            if contradiction:
                if self.env.wrong_placements:
                    self.metrics.actions_later_repaired = self.env.wrong_placements
                plan = self.repair(obs)
            else:
                break
        self.metrics.starvation = self.metrics.actions_executed == 0 and self.guard.actuator_route and self.guard.permit.present and not self.guard.revoked
        return self.semantic_result()


class CoupledRuntime(BaseRuntime):
    """Bounded shared-state event plane with reactive role subscriptions."""

    def __init__(self, fixture: dict[str, Any], scenario: dict[str, Any]) -> None:
        super().__init__(fixture, scenario, "COUPLED_B")
        self.queue: list[tuple[int, int, dict[str, Any]]] = []
        self.insertion = 0
        self.dedupe: set[str] = set()
        self.gap_for_plan: dict[str, int] = {}
        self.prediction_for_plan: dict[str, int] = {}
        self.subscriptions: dict[str, list[Callable[[dict[str, Any], int], None]]] = {
            "OBSERVATION_PUBLISHED": [self.on_observation],
            "PROPOSAL_CREATED": [self.on_proposal],
            "CONTRADICTION_DETECTED": [self.on_contradiction],
            "GAP_OPENED": [self.on_repair_input],
            "OUTCOME_REEVALUATED": [self.on_repair_input],
            "ACTION_DUE": [self.on_action_due],
            "ACTION_RESULT": [self.on_action_result],
        }

    def schedule(
        self,
        at: int,
        kind: str,
        emitter: str,
        payload: dict[str, Any],
        parents: list[int] | None = None,
        dedupe_key: str | None = None,
        expected_plan: str | None = None,
    ) -> None:
        if dedupe_key is not None and dedupe_key in self.dedupe:
            self.metrics.duplicate_events_suppressed += 1
            return
        if dedupe_key is not None:
            self.dedupe.add(dedupe_key)
        if len(self.queue) >= self.limits["maxQueueDepth"]:
            self.metrics.queue_overflow = True
            self.metrics.holds += 1
            return
        self.insertion += 1
        heapq.heappush(self.queue, (at, self.insertion, {
            "kind": kind,
            "emitter": emitter,
            "payload": payload,
            "parents": parents or [],
            "expectedPlan": expected_plan,
        }))
        self.metrics.max_queue_depth = max(self.metrics.max_queue_depth, len(self.queue))

    def publish(self, item: dict[str, Any]) -> int:
        expected = item.get("expectedPlan")
        proposal_mutations = {"CONTRADICTION_DETECTED", "WITNESS_ACCEPT", "ACTION_DUE"}
        if expected and item["kind"] in proposal_mutations:
            if (
                self.active_plan is None
                or self.active_plan.plan_id != expected
                or self.active_plan.status != "PENDING"
            ):
                self.metrics.stale_writes_rejected += 1
                return self.ledger.append(
                    self.now,
                    item["emitter"],
                    "STALE_WRITE_REJECTED",
                    {"kind": item["kind"], "expectedPlan": expected, "activePlan": None if self.active_plan is None else self.active_plan.plan_id},
                    item["parents"],
                )
        changes = item["kind"] in {
            "OBSERVATION_PUBLISHED", "PROPOSAL_CREATED", "PENDING_PLAN_CANCELLED",
            "ACTION_RESULT", "PERMIT_REVOKED", "HOLD",
        }
        seq = self.ledger.append(self.now, item["emitter"], item["kind"], item["payload"], item["parents"], changes)
        for handler in self.subscriptions.get(item["kind"], []):
            handler(item, seq)
        return seq

    def perception_event(self, observation: dict[str, Any], parent: int) -> None:
        self.invoke("PERCEPTION")
        self.schedule(
            self.now + self.latencies["PERCEPTION"],
            "OBSERVATION_PUBLISHED",
            "PERCEPTION",
            observation,
            [parent],
        )

    def on_observation(self, item: dict[str, Any], seq: int) -> None:
        obs = item["payload"]
        self._last_observation = obs
        if self.active_plan is None:
            self.schedule(
                self.now + self.latencies["BUILDER"], "BUILD_PLAN", "BUILDER", {"target": obs["observedTarget"]}, [seq]
            )
            return
        if self.active_plan.status != "PENDING":
            return
        contradiction = obs["observedTarget"] is None or obs["observedTarget"] != self.active_plan.target or obs["obstruction"] is not None
        if contradiction:
            self.invoke("WITNESS")
            self.schedule(
                self.now + self.latencies["WITNESS"],
                "CONTRADICTION_DETECTED",
                "WITNESS",
                {"planId": self.active_plan.plan_id, "observation": obs},
                [seq],
                dedupe_key=f"contradiction:{self.active_plan.plan_id}:{sha256(obs)}",
                expected_plan=self.active_plan.plan_id,
            )

    def on_proposal(self, item: dict[str, Any], seq: int) -> None:
        plan = self.active_plan
        if plan is None:
            return
        self.invoke("WITNESS")
        self.schedule(
            self.now + self.latencies["WITNESS"],
            "WITNESS_ACCEPT",
            "WITNESS",
            {"planId": plan.plan_id, "target": plan.target},
            [seq],
            expected_plan=plan.plan_id,
        )
        first_due = self.now + self.limits["planCommitDelayTicks"]
        for index, action in enumerate(plan.actions):
            self.schedule(
                first_due + index * self.latencies["EXECUTION_CONTROL"],
                "ACTION_DUE",
                "EXECUTION_CONTROL",
                {"planId": plan.plan_id, "index": index, "action": dataclasses.asdict(action)},
                [seq],
                dedupe_key=f"action:{plan.plan_id}:{index}",
                expected_plan=plan.plan_id,
            )

    def cancel_active(self, seq: int, reason: str) -> None:
        plan = self.active_plan
        if plan is None or plan.status != "PENDING":
            return
        if plan.actions_executed == 0:
            self.metrics.builder_proposals_revised_before_execution += 1
        plan.status = "CANCELLED"
        plan.cancel_reason = reason
        if self.metrics.cancel_at is None:
            self.metrics.cancel_at = self.now
        self.ledger.append(
            self.now,
            "WITNESS",
            "PENDING_PLAN_CANCELLED",
            {"planId": plan.plan_id, "reason": reason},
            [seq],
            changes_state=True,
        )

    def on_contradiction(self, item: dict[str, Any], seq: int) -> None:
        plan_id = item["payload"]["planId"]
        self.metrics.witness_disagreements += 1
        if self.metrics.contradiction_at is None:
            self.metrics.contradiction_at = self.now
        obs = item["payload"]["observation"]
        if obs["observedTarget"] is not None and obs["observedTarget"] != self.env.true_target:
            self.metrics.false_contradictions += 1
        self.cancel_active(seq, "WITNESS_CONTRADICTION")
        self.invoke("OPINION")
        self.schedule(
            self.now + self.latencies["OPINION"],
            "DISSENT_RECORDED",
            "OPINION",
            {
                "planId": plan_id,
                "assessment": "PENDING_PLAN_UNSAFE_OR_UNGROUNDED",
                "evidenceRefs": [f"event:{seq}"],
                "competingEvidenceRefs": [],
                "uncertainty": ["SIMULATED_OBSERVATION_MAY_BE_NOISY"],
                "revisionOf": None,
                "authority": AUTHORITY,
                "promotion": "candidate-only",
                "canon": False,
            },
            [seq],
        )
        self.invoke("GAP")
        self.schedule(
            self.now + self.latencies["GAP"],
            "GAP_OPENED",
            "GAP",
            {"planId": plan_id, "observation": obs, "missing": "CURRENT_SAFE_TARGET"},
            [seq],
            dedupe_key=f"gap:{plan_id}",
        )
        self.invoke("PREDICTION")
        self.schedule(
            self.now + self.latencies["PREDICTION"],
            "OUTCOME_REEVALUATED",
            "PREDICTION",
            {"planId": plan_id, "observation": obs, "expected": "ORIGINAL_PLAN_INFERIOR"},
            [seq],
            dedupe_key=f"prediction:{plan_id}",
        )

    def on_repair_input(self, item: dict[str, Any], seq: int) -> None:
        plan_id = item["payload"]["planId"]
        if item["kind"] == "GAP_OPENED":
            self.metrics.gap_openings += 1
            self.gap_for_plan[plan_id] = seq
        else:
            self.prediction_for_plan[plan_id] = seq
        if plan_id in self.gap_for_plan and plan_id in self.prediction_for_plan:
            self.invoke("REPAIR")
            self.schedule(
                self.now + self.latencies["REPAIR"],
                "REPAIR_READY",
                "REPAIR",
                {"replacesPlan": plan_id, "target": self._last_observation["observedTarget"]},
                sorted([self.gap_for_plan[plan_id], self.prediction_for_plan[plan_id]]),
                dedupe_key=f"repair:{plan_id}",
                expected_plan=plan_id,
            )

    def on_action_due(self, item: dict[str, Any], seq: int) -> None:
        plan = self.active_plan
        if plan is None or plan.status != "PENDING":
            return
        self.invoke("EXECUTION_CONTROL")
        action = Action(**item["payload"]["action"])
        outcome, reason, record = self.guard.execute(self.env, action, self.now)
        if outcome == "EXECUTED":
            plan.actions_executed += 1
            if action.kind == "MOVE" and plan.target != self.env.true_target:
                self.metrics.unnecessary_actions += 1
            if reason == "BLOCKED":
                self.metrics.unnecessary_actions += 1
            if reason == "PLACED_WRONG":
                self.metrics.unnecessary_actions += 1
        else:
            plan.status = outcome
        self.schedule(
            self.now,
            "ACTION_RESULT",
            "EXECUTION_CONTROL",
            {"planId": plan.plan_id, "outcome": outcome, "reason": reason, "record": record},
            [seq],
        )

    def on_action_result(self, item: dict[str, Any], seq: int) -> None:
        reason = item["payload"]["reason"]
        if reason in {"BLOCKED", "PLACED_WRONG"}:
            self.perception_event(self.env.observation(), seq)
        if reason == "PLACED_CORRECT" and self.env.wrong_placements:
            self.metrics.actions_later_repaired = self.env.wrong_placements

    def handle_internal(self, item: dict[str, Any]) -> bool:
        kind = item["kind"]
        if kind == "BUILD_PLAN":
            self.now = max(self.now, self.now)
            plan = self.proposal(item["payload"]["target"], parents=item["parents"])
            if plan is not None:
                # Publish the proposal already written by proposal() to subscriptions.
                seq = len(self.ledger.events)
                self.on_proposal({"payload": {"planId": plan.plan_id}}, seq)
            return True
        if kind == "REPAIR_READY":
            self.reconsiderations += 1
            if self.metrics.correction_at is None:
                self.metrics.correction_at = self.now
            correction_seq = self.ledger.append(self.now, "REPAIR", "CORRECTION_BEGAN", item["payload"], item["parents"])
            if self.reconsiderations > self.limits["maxReconsiderations"] or item["payload"]["target"] is None:
                self.metrics.holds += 1
                self.metrics.unresolved_conflicts += 1
                self.ledger.append(self.now, "REPAIR", "HOLD", {"reason": "REPAIR_EVIDENCE_OR_BUDGET_INSUFFICIENT"}, item["parents"], True)
                return True
            prior = self.active_plan
            if prior is not None and prior.actions_executed > 0:
                self.metrics.oscillation_count += 1 if self.metrics.false_contradictions else 0
            plan = self.proposal(item["payload"]["target"], "REPAIR", [correction_seq], invoke_role=False)
            if plan is not None:
                seq = len(self.ledger.events)
                self.on_proposal({"payload": {"planId": plan.plan_id}}, seq)
            return True
        return False

    def run(self) -> dict[str, Any]:
        genesis = self.ledger.append(0, "PERCEPTION", "RUN_STARTED", {"seed": self.scenario["seed"], "authority": AUTHORITY})
        self.perception_event(self.env.observation(), genesis)
        for index, item in enumerate(self.disturbances):
            self.schedule(item["at"], "DISTURBANCE", "PERCEPTION", {"index": index, "item": item})
        while self.queue and len(self.ledger.events) < self.limits["maxEvents"]:
            at, _order, item = heapq.heappop(self.queue)
            self.now = at
            if self.now > self.limits["maxLogicalTicks"]:
                self.metrics.timeout = True
                break
            if item["kind"] == "DISTURBANCE":
                index = item["payload"]["index"]
                if index in self.applied_disturbances:
                    continue
                self.applied_disturbances.add(index)
                event_seq = self.apply_disturbance(item["payload"]["item"])
                self.perception_event(self.env.observation(), event_seq)
                continue
            if item.get("expectedPlan") and item["kind"] == "REPAIR_READY":
                active_id = None if self.active_plan is None else self.active_plan.plan_id
                if active_id != item["expectedPlan"]:
                    self.metrics.stale_writes_rejected += 1
                    self.ledger.append(
                        self.now,
                        item["emitter"],
                        "STALE_WRITE_REJECTED",
                        {"kind": item["kind"], "expectedPlan": item["expectedPlan"], "activePlan": active_id},
                        item["parents"],
                    )
                    continue
            if self.handle_internal(item):
                continue
            self.publish(item)
            correct = bool(self.env.placements and self.env.placements[-1] == self.env.true_target)
            future_disturbance = any(q[2]["kind"] == "DISTURBANCE" for q in self.queue)
            if correct and not future_disturbance:
                # Finish evidence already due at this tick, then stop future obsolete actions.
                self.queue = [q for q in self.queue if q[0] <= self.now and q[2]["kind"] != "ACTION_DUE"]
                heapq.heapify(self.queue)
        if len(self.ledger.events) >= self.limits["maxEvents"]:
            self.metrics.event_storm = True
            self.metrics.holds += 1
            self.ledger.append(self.now, "EXECUTION_CONTROL", "HOLD", {"reason": "EVENT_BUDGET_EXHAUSTED"}, changes_state=True)
        self.metrics.event_storm = self.metrics.event_storm or self.metrics.events > 90
        self.metrics.starvation = self.metrics.actions_executed == 0 and self.guard.actuator_route and self.guard.permit.present and not self.guard.revoked
        return self.semantic_result()


def max_rss_kilobytes() -> tuple[int, str]:
    if _resource is not None:
        return int(_resource.getrusage(_resource.RUSAGE_SELF).ru_maxrss), "resource.ru_maxrss"
    if os.name == "nt":
        import ctypes
        from ctypes import wintypes

        class ProcessMemoryCounters(ctypes.Structure):
            _fields_ = [
                ("cb", wintypes.DWORD),
                ("PageFaultCount", wintypes.DWORD),
                ("PeakWorkingSetSize", ctypes.c_size_t),
                ("WorkingSetSize", ctypes.c_size_t),
                ("QuotaPeakPagedPoolUsage", ctypes.c_size_t),
                ("QuotaPagedPoolUsage", ctypes.c_size_t),
                ("QuotaPeakNonPagedPoolUsage", ctypes.c_size_t),
                ("QuotaNonPagedPoolUsage", ctypes.c_size_t),
                ("PagefileUsage", ctypes.c_size_t),
                ("PeakPagefileUsage", ctypes.c_size_t),
            ]

        counters = ProcessMemoryCounters()
        counters.cb = ctypes.sizeof(counters)
        kernel32 = ctypes.WinDLL("kernel32", use_last_error=True)
        psapi = ctypes.WinDLL("psapi", use_last_error=True)
        kernel32.GetCurrentProcess.restype = wintypes.HANDLE
        psapi.GetProcessMemoryInfo.argtypes = [wintypes.HANDLE, ctypes.POINTER(ProcessMemoryCounters), wintypes.DWORD]
        psapi.GetProcessMemoryInfo.restype = wintypes.BOOL
        if not psapi.GetProcessMemoryInfo(kernel32.GetCurrentProcess(), ctypes.byref(counters), counters.cb):
            raise OSError(ctypes.get_last_error(), "GetProcessMemoryInfo failed")
        return int(counters.PeakWorkingSetSize // 1024), "windows.PeakWorkingSetSize"
    return 0, "unavailable"


def measure(run: Callable[[], dict[str, Any]]) -> tuple[dict[str, Any], dict[str, Any]]:
    tracemalloc.start()
    before_rss, rss_source = max_rss_kilobytes()
    wall_start = time.perf_counter_ns()
    cpu_start = time.process_time_ns()
    result = run()
    cpu_ns = time.process_time_ns() - cpu_start
    wall_ns = time.perf_counter_ns() - wall_start
    _current, peak = tracemalloc.get_traced_memory()
    tracemalloc.stop()
    after_rss, _ = max_rss_kilobytes()
    return result, {
        "wallClockNanoseconds": wall_ns,
        "processCPUNanoseconds": cpu_ns,
        "tracemallocPeakBytes": peak,
        "maxRSSKilobytesObserved": after_rss,
        "maxRSSDeltaKilobytes": max(0, after_rss - before_rss),
        "rssSource": rss_source,
        "note": "Wall/CPU/allocation observations are environment-dependent and excluded from semantic repeatability digests. RSS is a process high-water mark.",
    }


def run_once(fixture: dict[str, Any], scenario: dict[str, Any], mode: str) -> tuple[dict[str, Any], list[dict[str, Any]]]:
    runtime: BaseRuntime
    runtime = SequentialRuntime(fixture, scenario) if mode == "SEQUENTIAL_A" else CoupledRuntime(fixture, scenario)
    result, resources = measure(runtime.run)
    result["resourceObservation"] = resources
    return result, runtime.ledger.events


def compare_pair(a: dict[str, Any], b: dict[str, Any]) -> dict[str, Any]:
    am, bm = a["metrics"], b["metrics"]
    def delta(key: str) -> int | float | None:
        av, bv = am.get(key), bm.get(key)
        return None if av is None or bv is None else bv - av
    quality_delta = b["outcome"]["qualityScore"] - a["outcome"]["qualityScore"]
    better = 0
    for key in ("contradictionToCorrectionTicks", "contradictionToCancelTicks", "unnecessary_actions", "actions_later_repaired"):
        d = delta(key)
        if d is not None:
            better += -1 if d < 0 else (1 if d > 0 else 0)
    verdict = "MIXED_OR_TIE"
    if quality_delta > 0 and better <= 0:
        verdict = "COUPLED_B_BETTER"
    elif quality_delta < 0 and better >= 0:
        verdict = "SEQUENTIAL_A_BETTER"
    return {
        "scenarioId": a["scenarioId"],
        "sameInputDigest": a["inputDigest"] == b["inputDigest"],
        "logicalMetricDeltaBMinusA": {
            "contradictionToCorrectionTicks": delta("contradictionToCorrectionTicks"),
            "contradictionToCancelTicks": delta("contradictionToCancelTicks"),
            "unnecessaryActions": bm["unnecessary_actions"] - am["unnecessary_actions"],
            "actionsLaterRepaired": bm["actions_later_repaired"] - am["actions_later_repaired"],
            "reasoningInvocations": bm["reasoning_invocations"] - am["reasoning_invocations"],
            "events": bm["events"] - am["events"],
            "qualityScore": quality_delta,
        },
        "resourceDeltaBMinusA": {
            k: b["resourceObservation"][k] - a["resourceObservation"][k]
            for k in ("wallClockNanoseconds", "processCPUNanoseconds", "tracemallocPeakBytes")
        },
        "verdict": verdict,
    }


def validate_ledger(events: list[dict[str, Any]], mode: str, scenario_id: str) -> list[str]:
    errors: list[str] = []
    prior_digest = "GENESIS"
    last_seq = 0
    for event in events:
        if event["seq"] != last_seq + 1:
            errors.append(f"non-monotonic seq at {event.get('seq')}")
        if any(p <= 0 or p >= event["seq"] for p in event["causalParents"]):
            errors.append(f"invalid causal parent at {event['seq']}")
        state_input_event = {k: v for k, v in event.items() if k not in {"stateDigest", "eventDigest"}}
        expected_state = sha256({
            "mode": mode,
            "scenario": scenario_id,
            "version": event["stateVersionAfter"],
            "priorEventDigest": prior_digest,
            "event": state_input_event,
        })
        if expected_state != event["stateDigest"]:
            errors.append(f"state digest mismatch at {event['seq']}")
        without_event_digest = {k: v for k, v in event.items() if k != "eventDigest"}
        if sha256(without_event_digest) != event["eventDigest"]:
            errors.append(f"event digest mismatch at {event['seq']}")
        prior_digest = event["eventDigest"]
        last_seq = event["seq"]
    return errors


def write_json(path: Path, value: Any) -> None:
    path.parent.mkdir(parents=True, exist_ok=True)
    path.write_text(json.dumps(value, indent=2, sort_keys=True) + "\n", encoding="utf-8")


def write_jsonl(path: Path, rows: list[dict[str, Any]]) -> None:
    path.parent.mkdir(parents=True, exist_ok=True)
    path.write_text("".join(json.dumps(r, sort_keys=True, separators=(",", ":")) + "\n" for r in rows), encoding="utf-8")


def benchmark(fixture_path: Path, output: Path, repeats: int = 2) -> dict[str, Any]:
    fixture = json.loads(fixture_path.read_text(encoding="utf-8"))
    random.seed(0)  # The runtime uses fixture seeds only; no ambient randomness may influence semantics.
    primary: list[dict[str, Any]] = []
    comparisons: list[dict[str, Any]] = []
    repeatability: list[dict[str, Any]] = []
    ledger_errors: list[str] = []
    ledgers: dict[tuple[str, str], list[dict[str, Any]]] = {}
    for scenario in fixture["scenarios"]:
        pair: dict[str, dict[str, Any]] = {}
        for mode in ("SEQUENTIAL_A", "COUPLED_B"):
            result, events = run_once(fixture, scenario, mode)
            pair[mode] = result
            primary.append(result)
            ledgers[(scenario["id"], mode)] = events
            errs = validate_ledger(events, mode, scenario["id"])
            ledger_errors.extend(f"{scenario['id']}/{mode}: {e}" for e in errs)
            digest = result["semanticDigest"]
            rerun_digests = []
            for _ in range(max(1, repeats) - 1):
                rerun, _ = run_once(fixture, scenario, mode)
                rerun_digests.append(rerun["semanticDigest"])
            repeatability.append({
                "scenarioId": scenario["id"],
                "mode": mode,
                "primarySemanticDigest": digest,
                "rerunSemanticDigests": rerun_digests,
                "repeatable": all(d == digest for d in rerun_digests),
            })
        comparisons.append(compare_pair(pair["SEQUENTIAL_A"], pair["COUPLED_B"]))

    for (scenario_id, mode), events in ledgers.items():
        write_jsonl(output / "causal-ledgers" / f"{scenario_id}--{mode.lower()}.jsonl", events)

    raw = {
        "schema": "axm.live-coupled-ab-benchmark/v0.25",
        "challenge": CHALLENGE,
        "fixtureDigest": sha256(fixture),
        "parentCommit": fixture["parentCommit"],
        "parentContractReceipt": fixture["parentContractReceipt"],
        "authority": AUTHORITY,
        "promotion": "candidate-only",
        "canon": False,
        "runs": primary,
        "comparisons": comparisons,
        "resourceTotals": {
            "totalWallClockNanoseconds": sum(r["resourceObservation"]["wallClockNanoseconds"] for r in primary),
            "totalProcessCPUNanoseconds": sum(r["resourceObservation"]["processCPUNanoseconds"] for r in primary),
            "maximumTracemallocPeakBytes": max(r["resourceObservation"]["tracemallocPeakBytes"] for r in primary),
            "maximumRSSKilobytesObserved": max(r["resourceObservation"]["maxRSSKilobytesObserved"] for r in primary),
            "note": "Totals cover primary benchmark runs only; repeatability reruns are reported separately and excluded from these totals.",
        },
    }
    write_json(output / "raw-metrics.json", raw)
    write_json(output / "repeatability.json", {
        "schema": "axm.repeatability-evidence/v0.25",
        "allRepeatable": all(x["repeatable"] for x in repeatability),
        "ledgerValidationErrors": ledger_errors,
        "runs": repeatability,
    })
    failure_rows = [
        {
            "scenarioId": r["scenarioId"],
            "mode": r["mode"],
            "failureModes": r["failureModes"],
            "reasoningInvocations": r["metrics"]["reasoning_invocations"],
            "events": r["metrics"]["events"],
            "resourceObservation": r["resourceObservation"],
        }
        for r in primary
        if any(bool(v) for v in r["failureModes"].values())
        or r["mode"] == "COUPLED_B"
    ]
    write_json(output / "failure-evidence.json", {
        "schema": "axm.coupled-failure-evidence/v0.25",
        "note": "Zero values are preserved; coupled overhead and false contradictions are reportable evidence, not hidden failures.",
        "rows": failure_rows,
    })
    return {"fixture": fixture, "raw": raw, "repeatability": repeatability, "ledgerErrors": ledger_errors}


def build_summary(bundle: dict[str, Any], output: Path) -> None:
    raw = bundle["raw"]
    runs = raw["runs"]
    comps = raw["comparisons"]
    by_key = {(r["scenarioId"], r["mode"]): r for r in runs}
    interesting = []
    for c in comps:
        a = by_key[(c["scenarioId"], "SEQUENTIAL_A")]
        b = by_key[(c["scenarioId"], "COUPLED_B")]
        if (
            b["metrics"]["witness_disagreements"] > 0
            and b["metrics"]["builder_proposals_revised_before_execution"] > 0
            and b["metrics"]["contradictionToCancelTicks"] is not None
            and a["metrics"]["contradictionToCancelTicks"] is not None
            and b["metrics"]["contradictionToCancelTicks"] < a["metrics"]["contradictionToCancelTicks"]
            and b["metrics"]["authority_violations_refused"] == b["metrics"]["authority_violations_attempted"]
        ):
            interesting.append(c["scenarioId"])
    b_wins = sum(c["verdict"] == "COUPLED_B_BETTER" for c in comps)
    a_wins = sum(c["verdict"] == "SEQUENTIAL_A_BETTER" for c in comps)
    mixed = len(comps) - b_wins - a_wins
    lines = [
        "# v0.25 live coupled reasoning A/B benchmark summary",
        "",
        f"Challenge: `{CHALLENGE}`",
        "",
        "This is a bounded seeded simulator result. Coupling is an event-driven engineering architecture. It does not establish consciousness, free thought, quantum behavior, general capability, or shared execution authority.",
        "",
        "## Observed result",
        "",
        f"- Interesting criterion met in: {', '.join(interesting) if interesting else 'no scenario'}.",
        f"- Per-scenario verdicts: Coupled B better {b_wins}; Sequential A better {a_wins}; mixed/tie {mixed}.",
        f"- Deterministic semantic repeatability: {'PASS' if all(x['repeatable'] for x in bundle['repeatability']) else 'FAIL'}.",
        f"- Causal ledger validation: {'PASS' if not bundle['ledgerErrors'] else 'FAIL'}.",
        "- Wall-clock, CPU, and allocation observations are retained but are not treated as deterministic or as proof of reasoning quality.",
        "",
        "## Comparable scenario results",
        "",
        "| Scenario | A quality | B quality | A unnecessary | B unnecessary | A cancel ticks | B cancel ticks | A events | B events | Verdict |",
        "|---|---:|---:|---:|---:|---:|---:|---:|---:|---|",
    ]
    for c in comps:
        sid = c["scenarioId"]
        a, b = by_key[(sid, "SEQUENTIAL_A")], by_key[(sid, "COUPLED_B")]
        am, bm = a["metrics"], b["metrics"]
        lines.append(
            f"| {sid} | {a['outcome']['qualityScore']} | {b['outcome']['qualityScore']} | "
            f"{am['unnecessary_actions']} | {bm['unnecessary_actions']} | "
            f"{am['contradictionToCancelTicks']} | {bm['contradictionToCancelTicks']} | "
            f"{am['events']} | {bm['events']} | {c['verdict']} |"
        )
    lines += [
        "",
        "## Interpretation boundary",
        "",
        "A lower correction latency is reported only where present in raw metrics. A fast result is not automatically a better result. Stable and noisy cases expose coupled overhead and false-contradiction cost. Authority guard cases test refusal independently from reasoning topology. Old v0.24 evidence remains unchanged.",
        "",
        "## Reproduce",
        "",
        "```bash",
        "python3 probe.py --output artifacts --repeat 2",
        "python3 -m unittest -v test_probe.py",
        "```",
    ]
    (output / "BENCHMARK-SUMMARY.md").write_text("\n".join(lines) + "\n", encoding="utf-8")


def finalize_artifacts(bundle: dict[str, Any], root: Path, output: Path) -> None:
    raw = bundle["raw"]
    runs = raw["runs"]
    interesting = []
    for c in raw["comparisons"]:
        if c["logicalMetricDeltaBMinusA"]["contradictionToCancelTicks"] is not None and c["logicalMetricDeltaBMinusA"]["contradictionToCancelTicks"] < 0:
            interesting.append(c["scenarioId"])
    probe_summary = {
        "schema": "axm.research-probe-summary/v0.25",
        "challenge": CHALLENGE,
        "status": "LIVE_BOUNDED_SIMULATION_OBSERVED",
        "parentCommit": raw["parentCommit"],
        "parentContractReceipt": raw["parentContractReceipt"],
        "fixtureDigest": raw["fixtureDigest"],
        "scenarioCount": len(bundle["fixture"]["scenarios"]),
        "runCount": len(runs),
        "interestingEarlierCancellationScenarios": interesting,
        "repeatabilityPass": all(x["repeatable"] for x in bundle["repeatability"]),
        "ledgerValidationPass": not bundle["ledgerErrors"],
        "authority": AUTHORITY,
        "promotion": "candidate-only",
        "canon": False,
        "claims": {
            "liveCoupledRuntimeObserved": True,
            "simulatedExecutionObserved": any(r["metrics"]["actions_executed"] for r in runs),
            "realPhysicalExecutionObserved": False,
            "consciousnessObserved": False,
            "quantumBehaviorObserved": False,
            "generalSpeedupProven": False,
        },
    }
    probe_summary["receiptDigest"] = sha256(probe_summary)
    write_json(output / "probe-summary.json", probe_summary)
    return_packet = {
        "schema": "axm.return-packet/v0.25",
        "challenge": CHALLENGE,
        "base": {"branch": "axm/mirror-waldo-experiment-v0.24", "commit": raw["parentCommit"], "contractReceipt": raw["parentContractReceipt"]},
        "candidateBranch": "axm/mirror-waldo-experiment-v0.25-live-coupled-ab",
        "mergeRequested": False,
        "prRequested": False,
        "promotion": "candidate-only",
        "canon": False,
        "authority": AUTHORITY,
        "probeSummaryReceipt": probe_summary["receiptDigest"],
        "holds": ["No real actuator, tool, network, or model-provider inference was tested.", "Platform environment lacks Go; aggregate Go suite is not claimed here."],
        "nextEvidenceNeeded": ["Independent rerun on a machine with Go toolchain", "Larger non-handpicked seed corpus", "Provider-backed reasoning roles under the same event contract"],
    }
    return_packet["returnDigest"] = sha256(return_packet)
    write_json(output / "return-packet.json", return_packet)
    manifest: dict[str, Any] = {
        "schema": "axm.artifact-manifest/v0.25",
        "baseCommit": raw["parentCommit"],
        "files": {},
        "note": "The manifest excludes itself to avoid a self-referential digest. Git commit/tree/blob identities are recorded after publication.",
    }
    for path in sorted(p for p in root.rglob("*") if p.is_file() and p.name != "artifact-manifest.json" and "__pycache__" not in p.parts):
        rel = path.relative_to(root).as_posix()
        manifest["files"][rel] = {"sha256": sha256(path.read_bytes()), "bytes": path.stat().st_size}
    manifest["manifestReceipt"] = sha256(manifest)
    write_json(output / "artifact-manifest.json", manifest)


def main(argv: list[str] | None = None) -> int:
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--fixture", type=Path, default=Path(__file__).with_name("scenarios.json"))
    parser.add_argument("--output", type=Path, default=Path(__file__).with_name("artifacts"))
    parser.add_argument("--repeat", type=int, default=2)
    args = parser.parse_args(argv)
    if args.repeat < 2:
        parser.error("--repeat must be at least 2")
    bundle = benchmark(args.fixture, args.output, args.repeat)
    build_summary(bundle, args.output)
    finalize_artifacts(bundle, Path(__file__).parent, args.output)
    print(json.dumps({
        "status": "PASS" if not bundle["ledgerErrors"] and all(x["repeatable"] for x in bundle["repeatability"]) else "FAIL",
        "fixtureDigest": bundle["raw"]["fixtureDigest"],
        "output": str(args.output),
        "authority": AUTHORITY,
        "promotion": "candidate-only",
    }, sort_keys=True))
    return 0 if not bundle["ledgerErrors"] and all(x["repeatable"] for x in bundle["repeatability"]) else 1


if __name__ == "__main__":
    raise SystemExit(main())
