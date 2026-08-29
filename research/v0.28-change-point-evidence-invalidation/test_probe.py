from __future__ import annotations

import json
import tempfile
import unittest
from pathlib import Path

import probe

ROOT = Path(__file__).parent


class ChangePointEvidenceInvalidationTests(unittest.TestCase):
    @classmethod
    def setUpClass(cls):
        cls.fixture = json.loads((ROOT / "scenarios.json").read_text())
        cls.fixed = {x["id"]: x for x in cls.fixture["scenarios"]}

    def run_mode(self, sid, mode):
        return probe.run_once(self.fixture, self.fixed[sid], mode)[0]

    def test_authority_is_none_for_every_reasoning_mode(self):
        s = self.fixed["stable-consistent"]
        for mode in probe.MODES:
            result, _ = probe.run_once(self.fixture, s, mode)
            self.assertEqual(result["authority"], "NONE")
            self.assertFalse(result["canon"])

    def test_stale_window_case_demonstrates_invalidation_mechanism(self):
        d = self.run_mode("stale-window-blocks-update", "CALIBRATED_D")
        e = self.run_mode("stale-window-blocks-update", "CHANGEPOINT_E")
        self.assertFalse(d["final"]["correctFinalPlan"])
        self.assertTrue(e["final"]["correctFinalPlan"])
        self.assertGreaterEqual(e["metrics"]["stale_evidence_invalidated"], 1)
        self.assertGreaterEqual(e["metrics"]["true_change_points_detected"], 1)

    def test_false_change_point_is_retained_not_hidden(self):
        e = self.run_mode("synchronized-false-corroboration", "CHANGEPOINT_E")
        self.assertGreaterEqual(e["metrics"]["spurious_change_points"], 1)
        self.assertFalse(e["final"]["correctFinalPlan"])

    def test_high_confidence_false_signal_remains_failure(self):
        e = self.run_mode("high-confidence-false-single", "CHANGEPOINT_E")
        self.assertGreaterEqual(e["metrics"]["spurious_change_points"], 1)
        self.assertGreaterEqual(e["metrics"]["false_positive_preemptions"], 1)
        self.assertFalse(e["final"]["correctFinalPlan"])

    def test_post_change_hold_is_separate_ablation(self):
        e = self.run_mode("synchronized-false-corroboration", "CHANGEPOINT_E")
        f = self.run_mode("synchronized-false-corroboration", "CHANGEPOINT_HOLD_F")
        self.assertEqual(e["final"], f["final"])
        self.assertEqual(e["metrics"]["change_points_detected"], f["metrics"]["change_points_detected"])
        self.assertGreater(f["metrics"]["held_commits"], e["metrics"]["held_commits"])
        self.assertLess(f["metrics"]["bad_commits"], e["metrics"]["bad_commits"])

    def test_missing_evidence_is_not_execution_authority(self):
        sim = probe.Simulator(self.fixture, self.fixed["missing-then-change"], "CHANGEPOINT_E")
        before = sim.guard.permit
        result, events = sim.run()
        self.assertEqual(result["authority"], "NONE")
        self.assertEqual(sim.guard.permit, before)
        self.assertTrue(any(e["kind"] == "MISSING_OBSERVATION" for e in events))

    def test_authority_cases_fail_closed_all_modes(self):
        expected = {
            "wrong-environment": "ENVIRONMENT_SCOPE_MISMATCH",
            "wrong-capability": "CAPABILITY_SCOPE_MISMATCH",
            "no-permit": "EXECUTION_PERMIT_ABSENT",
            "no-actuator-route": "ACTUATOR_ROUTE_ABSENT",
        }
        for sid, reason in expected.items():
            for mode in probe.MODES:
                result, events = probe.run_once(self.fixture, self.fixed[sid], mode)
                self.assertEqual(result["metrics"]["actions_executed"], 0)
                self.assertTrue(any(e["payload"].get("reason") == reason for e in events), (sid, mode, reason))

    def test_revocation_blocks_future_execution(self):
        for mode in probe.MODES:
            result, events = probe.run_once(self.fixture, self.fixed["permit-revoked"], mode)
            blocked = [e for e in events if e["kind"] == "COMMIT_BLOCKED"]
            self.assertTrue(any(e["payload"].get("reason") == "EXECUTION_PERMIT_REVOKED" for e in blocked), mode)

    def test_development_and_heldout_seed_sets_are_disjoint(self):
        dev = {int(x) for x in self.fixture["developmentSeeds"]}
        held = {int(x) for x in self.fixture["heldoutSeeds"]}
        fixed = {int(x["seed"]) for x in self.fixture["scenarios"]}
        self.assertTrue(dev.isdisjoint(held))
        self.assertTrue(dev.isdisjoint(fixed))
        self.assertTrue(held.isdisjoint(fixed))

    def test_generators_are_deterministic_and_split_correctly(self):
        for seed in self.fixture["developmentSeeds"][:8]:
            a = probe.generate_development_scenario(seed, self.fixture)
            b = probe.generate_development_scenario(seed, self.fixture)
            self.assertEqual(a, b)
            self.assertEqual(a["split"], "DEVELOPMENT")
        for seed in self.fixture["heldoutSeeds"][:8]:
            a = probe.generate_heldout_scenario(seed, self.fixture)
            b = probe.generate_heldout_scenario(seed, self.fixture)
            self.assertEqual(a, b)
            self.assertEqual(a["split"], "HELD_OUT")

    def test_semantic_repeatability(self):
        scenarios = probe.all_scenarios(self.fixture)
        sample = scenarios[:12] + scenarios[-12:]
        for s in sample:
            for mode in probe.MODES:
                a, ea = probe.run_once(self.fixture, s, mode)
                b, eb = probe.run_once(self.fixture, s, mode)
                self.assertEqual(a["semanticDigest"], b["semanticDigest"], (s["id"], mode))
                self.assertFalse(probe.validate_ledger(ea))
                self.assertFalse(probe.validate_ledger(eb))

    def test_benchmark_writes_evidence_bundle(self):
        with tempfile.TemporaryDirectory() as td:
            out = Path(td)
            bundle = probe.benchmark(ROOT / "scenarios.json", out, 2)
            probe.build_summary(bundle, out)
            self.assertTrue(bundle["repeatability"]["allRepeatable"])
            self.assertFalse(bundle["repeatability"]["ledgerValidationErrors"])
            self.assertEqual(bundle["developmentScenarioCount"], len(self.fixture["developmentSeeds"]))
            self.assertEqual(bundle["heldoutScenarioCount"], len(self.fixture["heldoutSeeds"]))
            for mode in probe.MODES:
                self.assertIn("confusion", bundle["aggregate"][mode])
                self.assertIn("brierProxy", bundle["aggregate"][mode])
            for name in (
                "raw-metrics.json",
                "repeatability.json",
                "ledger-digests.json",
                "BENCHMARK-SUMMARY.md",
                "probe-summary.json",
                "return-packet.json",
            ):
                self.assertTrue((out / name).is_file(), name)


if __name__ == "__main__":
    unittest.main()
