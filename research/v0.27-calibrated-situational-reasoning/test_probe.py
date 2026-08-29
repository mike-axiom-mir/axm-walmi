from __future__ import annotations

import json
import tempfile
import unittest
from pathlib import Path

import probe

ROOT = Path(__file__).parent


class CalibratedReasoningTests(unittest.TestCase):
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

    def test_single_low_confidence_false_signal_does_not_preempt_calibrated(self):
        d = self.run_mode("single-low-confidence-false", "CALIBRATED_D")
        self.assertEqual(d["metrics"]["false_positive_preemptions"], 0)
        self.assertTrue(d["final"]["correctFinalPlan"])

    def test_persistent_weak_real_change_can_recruit_deliberative_burst(self):
        d = self.run_mode("persistent-weak-real-change", "CALIBRATED_D")
        c = self.run_mode("persistent-weak-real-change", "HYBRID_C")
        self.assertGreaterEqual(d["metrics"]["deliberative_bursts"], 1)
        self.assertTrue(d["final"]["correctFinalPlan"])
        self.assertLess(d["metrics"]["false_negative_delay_ticks"], c["metrics"]["false_negative_delay_ticks"])

    def test_independent_corroboration_can_escalate(self):
        d = self.run_mode("corroborated-real-change", "CALIBRATED_D")
        self.assertGreaterEqual(d["metrics"]["coupled_activations"], 1)
        self.assertTrue(d["final"]["correctFinalPlan"])

    def test_strong_real_change_is_immediate_enough(self):
        d = self.run_mode("strong-real-change", "CALIBRATED_D")
        self.assertEqual(d["metrics"]["false_negative_delay_ticks"], 0)
        self.assertTrue(d["final"]["correctFinalPlan"])

    def test_missing_evidence_is_not_execution_authority(self):
        sim = probe.Simulator(self.fixture, self.fixed["missing-only"], "CALIBRATED_D")
        before = sim.guard.permit
        result, events = sim.run()
        self.assertEqual(result["authority"], "NONE")
        self.assertEqual(sim.guard.permit, before)
        self.assertTrue(any(e["kind"] == "COMMIT_HOLD" for e in events))

    def test_retains_real_negative_cases(self):
        low = self.run_mode("genuine-low-confidence-single", "CALIBRATED_D")
        high_false = self.run_mode("high-confidence-false-single", "CALIBRATED_D")
        # D is intentionally not tuned to make every designed case win.
        self.assertFalse(low["final"]["correctFinalPlan"])
        self.assertGreaterEqual(high_false["metrics"]["false_positive_preemptions"], 1)

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

    def test_semantic_repeatability(self):
        scenarios = probe.all_scenarios(self.fixture)
        for s in scenarios[:10] + scenarios[-10:]:
            for mode in probe.MODES:
                a, _ = probe.run_once(self.fixture, s, mode)
                b, _ = probe.run_once(self.fixture, s, mode)
                self.assertEqual(a["semanticDigest"], b["semanticDigest"], (s["id"], mode))

    def test_heldout_seed_set_is_disjoint_and_generator_deterministic(self):
        fixed_seeds = {int(x["seed"]) for x in self.fixture["scenarios"]}
        held = {int(x) for x in self.fixture["heldoutSeeds"]}
        self.assertTrue(fixed_seeds.isdisjoint(held))
        for seed in list(sorted(held))[:8]:
            a = probe.generate_heldout_scenario(seed, self.fixture)
            b = probe.generate_heldout_scenario(seed, self.fixture)
            self.assertEqual(a, b)
            self.assertEqual(a["split"], "HELD_OUT")

    def test_benchmark_writes_calibration_evidence(self):
        with tempfile.TemporaryDirectory() as td:
            out = Path(td)
            bundle = probe.benchmark(ROOT / "scenarios.json", out, 2)
            probe.build_summary(bundle, out)
            self.assertTrue(bundle["repeatability"]["allRepeatable"])
            self.assertFalse(bundle["repeatability"]["ledgerValidationErrors"])
            self.assertEqual(bundle["heldoutScenarioCount"], len(self.fixture["heldoutSeeds"]))
            for mode in probe.MODES:
                self.assertIn("confusion", bundle["aggregate"][mode])
                self.assertIn("calibrationCurve", bundle["aggregate"][mode])
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
