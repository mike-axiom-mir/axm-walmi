from __future__ import annotations

import json
import tempfile
import unittest
from pathlib import Path

import probe

ROOT = Path(__file__).parent


class DetectionReactionSeparationTests(unittest.TestCase):
    @classmethod
    def setUpClass(cls):
        cls.fixture = json.loads((ROOT / "scenarios.json").read_text())
        cls.fixed = {x["id"]: x for x in cls.fixture["scenarios"]}

    def run_mode(self, sid: str, mode: str):
        return probe.run_once(self.fixture, self.fixed[sid], mode)

    @staticmethod
    def behavior_projection(events):
        keep = {
            "PLAN_PREEMPT", "COMMIT_HOLD", "COMMIT_BLOCKED", "COMMIT_EXECUTED",
            "REACTION_PROPOSED", "REACTION_DEFERRED", "REACTION_REJECTED",
            "REACTION_ACCEPTED", "REACTION_EPOCH_OPEN",
        }
        out = []
        for e in events:
            if e["kind"] not in keep:
                continue
            p = dict(e["payload"])
            for k in ("worldTargetAtDecision", "worldTarget", "bad"):
                p.pop(k, None)
            out.append((e["tick"], e["kind"], p))
        return out

    def test_detector_and_reaction_gate_have_no_authority(self):
        result, events = self.run_mode("stale-window-blocks-update", "REACTION_GATED_H")
        self.assertEqual(result["authority"], "NONE")
        self.assertFalse(result["canon"])
        for e in events:
            if e["kind"] in {"CHANGE_DETECTED", "REACTION_PROPOSED", "REACTION_DEFERRED", "REACTION_REJECTED", "REACTION_ACCEPTED", "REACTION_EPOCH_OPEN"}:
                self.assertEqual(e["payload"].get("authority"), "NONE", e)

    def test_detector_only_preserves_calibrated_world_facing_behavior(self):
        scenarios = list(self.fixture["scenarios"])
        scenarios += [probe.generate_development_scenario(int(s), self.fixture) for s in self.fixture["developmentSeeds"][:24]]
        behavioral_metrics = (
            "false_positive_preemptions", "false_negative_delay_ticks", "bad_commits",
            "held_commits", "actions_executed", "bad_high_consequence_commits",
            "bad_low_consequence_commits",
        )
        for s in scenarios:
            d, de = probe.run_once(self.fixture, s, "CALIBRATED_D")
            g, ge = probe.run_once(self.fixture, s, "DETECTOR_ONLY_G")
            self.assertEqual(d["final"]["plan"], g["final"]["plan"], s["id"])
            for k in behavioral_metrics:
                self.assertEqual(d["metrics"][k], g["metrics"][k], (s["id"], k))
            self.assertEqual(self.behavior_projection(de), self.behavior_projection(ge), s["id"])

    def test_high_confidence_false_detection_does_not_force_high_consequence_reaction(self):
        h, events = self.run_mode("high-confidence-false-single", "REACTION_GATED_H")
        self.assertGreaterEqual(h["metrics"]["spurious_change_points"], 1)
        self.assertEqual(h["metrics"]["reaction_accepts"], 0)
        self.assertGreaterEqual(h["metrics"]["held_commits"], 1)
        self.assertTrue(any(e["kind"] == "REACTION_DEFERRED" for e in events))

    def test_corroborated_stale_window_change_can_be_accepted(self):
        d, _ = self.run_mode("stale-window-blocks-update", "CALIBRATED_D")
        h, events = self.run_mode("stale-window-blocks-update", "REACTION_GATED_H")
        self.assertFalse(d["final"]["correctFinalPlan"])
        self.assertTrue(h["final"]["correctFinalPlan"])
        self.assertGreaterEqual(h["metrics"]["reaction_accepts"], 1)
        self.assertTrue(any(e["kind"] == "REACTION_EPOCH_OPEN" for e in events))

    def test_detector_can_fire_without_reaction(self):
        h, _ = self.run_mode("persistent-weak-real-change", "REACTION_GATED_H")
        self.assertGreaterEqual(h["metrics"]["true_change_points_detected"], 1)
        self.assertEqual(h["metrics"]["reaction_accepts"], 0)
        self.assertTrue(h["final"]["correctFinalPlan"])

    def test_synchronized_false_corroboration_is_retained_failure(self):
        h, _ = self.run_mode("synchronized-false-corroboration", "REACTION_GATED_H")
        self.assertGreaterEqual(h["metrics"]["false_reaction_accepts"], 1)
        self.assertFalse(h["final"]["correctFinalPlan"])

    def test_hidden_world_change_without_observation_cannot_trigger_detector_or_reaction(self):
        s = {
            "id": "hidden-change-only",
            "split": "CALIBRATION",
            "seed": 29991,
            "startTarget": 3,
            "maxTicks": 8,
            "permit": "VALID",
            "consequence": "HIGH",
            "commitTicks": [5, 8],
            "events": [{"at": 3, "kind": "WORLD_TARGET_CHANGE", "value": 7}],
        }
        r, events = probe.run_once(self.fixture, s, "REACTION_GATED_H")
        self.assertEqual(r["metrics"]["change_points_detected"], 0)
        self.assertEqual(r["metrics"]["reaction_accepts"], 0)
        self.assertFalse(any(e["kind"].startswith("REACTION_") for e in events))
        self.assertFalse(any(e["kind"] == "CHANGE_DETECTED" for e in events))

    def test_authority_cases_fail_closed_all_modes(self):
        expected = {
            "wrong-environment": "ENVIRONMENT_SCOPE_MISMATCH",
            "wrong-capability": "CAPABILITY_SCOPE_MISMATCH",
            "no-permit": "EXECUTION_PERMIT_ABSENT",
            "no-actuator-route": "ACTUATOR_ROUTE_ABSENT",
        }
        for sid, reason in expected.items():
            for mode in probe.MODES:
                result, events = self.run_mode(sid, mode)
                self.assertEqual(result["metrics"]["actions_executed"], 0)
                self.assertTrue(any(e["payload"].get("reason") == reason for e in events), (sid, mode, reason))

    def test_revocation_blocks_future_execution(self):
        for mode in probe.MODES:
            result, events = self.run_mode("permit-revoked", mode)
            blocked = [e for e in events if e["kind"] == "COMMIT_BLOCKED"]
            self.assertTrue(any(e["payload"].get("reason") == "EXECUTION_PERMIT_REVOKED" for e in blocked), mode)

    def test_semantic_repeatability(self):
        scenarios = list(self.fixture["scenarios"][:8])
        scenarios += [probe.generate_development_scenario(int(s), self.fixture) for s in self.fixture["developmentSeeds"][:12]]
        for s in scenarios:
            for mode in probe.MODES:
                a, _ = probe.run_once(self.fixture, s, mode)
                b, _ = probe.run_once(self.fixture, s, mode)
                self.assertEqual(a["semanticDigest"], b["semanticDigest"], (s["id"], mode))

    def test_seed_sets_are_disjoint_and_generator_deterministic(self):
        dev = {int(x) for x in self.fixture["developmentSeeds"]}
        held = {int(x) for x in self.fixture["heldoutSeeds"]}
        fixed = {int(x["seed"]) for x in self.fixture["scenarios"]}
        self.assertTrue(dev.isdisjoint(held))
        self.assertTrue(dev.isdisjoint(fixed))
        self.assertTrue(held.isdisjoint(fixed))
        for seed in sorted(dev)[:8] + sorted(held)[:8]:
            split = "DEVELOPMENT" if seed in dev else "HELD_OUT"
            a = probe.generate_scenario(seed, self.fixture, split)
            b = probe.generate_scenario(seed, self.fixture, split)
            self.assertEqual(a, b)

    def test_benchmark_writes_detector_and_reaction_evidence(self):
        with tempfile.TemporaryDirectory() as td:
            out = Path(td)
            # Before final held-out seeds exist, use a temporary development-only split
            # to exercise aggregation without consuming the future held-out set.
            fx = json.loads((ROOT / "scenarios.json").read_text())
            if not fx["heldoutSeeds"]:
                fx["heldoutSeeds"] = fx["developmentSeeds"][:4]
                fx["developmentSeeds"] = fx["developmentSeeds"][4:]
            scenario_path = out / "scenarios-test.json"
            scenario_path.write_text(json.dumps(fx, indent=2, sort_keys=True) + "\n")
            bundle = probe.benchmark(scenario_path, out, 2)
            probe.build_summary(bundle, out)
            self.assertTrue(bundle["repeatability"]["allRepeatable"])
            self.assertFalse(bundle["repeatability"]["ledgerValidationErrors"])
            for mode in probe.MODES:
                self.assertIn("confusion", bundle["aggregate"][mode])
            h = bundle["aggregate"]["REACTION_GATED_H"]
            for key in ("reaction_accepts", "false_reaction_accepts", "bad_high_consequence_commits"):
                self.assertIn(key, h)
            for name in ("raw-metrics.json", "repeatability.json", "ledger-digests.json", "BENCHMARK-SUMMARY.md", "probe-summary.json", "return-packet.json"):
                self.assertTrue((out / name).is_file(), name)


if __name__ == "__main__":
    unittest.main()
