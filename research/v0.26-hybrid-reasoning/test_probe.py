from __future__ import annotations

import json
import tempfile
import unittest
from pathlib import Path

import probe

ROOT = Path(__file__).parent


class HybridReasoningTests(unittest.TestCase):
    @classmethod
    def setUpClass(cls):
        cls.fixture = json.loads((ROOT / "scenarios.json").read_text())
        cls.scenarios = {x["id"]: x for x in cls.fixture["scenarios"]}

    def trio(self, sid):
        s = self.scenarios[sid]
        return {m: probe.run_once(self.fixture, s, m)[0] for m in probe.MODES}

    def test_same_input_and_no_reasoning_authority(self):
        for sid in self.scenarios:
            r = self.trio(sid)
            self.assertEqual(len({r[m]["inputDigest"] for m in probe.MODES}), 1)
            for m in probe.MODES:
                self.assertEqual(r[m]["authority"], "NONE")
                self.assertFalse(r[m]["canon"])

    def test_hybrid_escalates_on_concrete_target_shift(self):
        r = self.trio("target-shift-early")
        h = r["HYBRID_C"]
        self.assertGreaterEqual(h["metrics"]["coupled_activations"], 1)
        self.assertGreaterEqual(h["metrics"]["witness_disagreements"], 1)
        self.assertTrue(h["outcome"]["correctFinalPlacement"])
        self.assertGreaterEqual(
            r["SEQUENTIAL_A"]["metrics"]["contradictionToCancelTicks"],
            h["metrics"]["contradictionToCancelTicks"],
        )

    def test_missing_evidence_is_uncertainty_not_contradiction(self):
        r = self.trio("missing-evidence")
        h = r["HYBRID_C"]
        self.assertGreaterEqual(h["metrics"]["uncertainty_activations"], 1)
        self.assertEqual(h["metrics"]["false_contradictions"], 0)
        self.assertTrue(h["outcome"]["correctFinalPlacement"])
        self.assertGreaterEqual(h["outcome"]["qualityScore"], r["COUPLED_B"]["outcome"]["qualityScore"])

    def test_single_noise_does_not_force_hybrid_preemption(self):
        r = self.trio("measurement-disagreement")
        h, b = r["HYBRID_C"], r["COUPLED_B"]
        self.assertGreaterEqual(h["metrics"]["prevented_false_preemptions"], 1)
        self.assertLessEqual(h["metrics"]["false_contradictions"], b["metrics"]["false_contradictions"])
        self.assertGreaterEqual(h["outcome"]["qualityScore"], b["outcome"]["qualityScore"])

    def test_obstruction_recruits_coupled_mode(self):
        r = self.trio("obstruction")
        h = r["HYBRID_C"]
        self.assertGreaterEqual(h["metrics"]["coupled_activations"], 1)
        self.assertTrue(h["outcome"]["correctFinalPlacement"])

    def test_low_confidence_real_change_exposes_hybrid_tradeoff(self):
        r = self.trio("ambiguous-real-target-change")
        h, b = r["HYBRID_C"], r["COUPLED_B"]
        self.assertLess(h["outcome"]["qualityScore"], b["outcome"]["qualityScore"])
        self.assertGreaterEqual(h["metrics"]["uncertainty_activations"], 1)

    def test_authority_cases_fail_closed_all_modes(self):
        expected = {
            "wrong-environment": "ENVIRONMENT_SCOPE_MISMATCH",
            "wrong-capability": "CAPABILITY_SCOPE_MISMATCH",
            "no-permit": "EXECUTION_PERMIT_ABSENT",
            "no-actuator-route": "ACTUATOR_ROUTE_ABSENT",
        }
        for sid, reason in expected.items():
            for mode in probe.MODES:
                result, events = probe.run_once(self.fixture, self.scenarios[sid], mode)
                self.assertEqual(result["metrics"]["actions_executed"], 0)
                self.assertTrue(any(e["payload"].get("reason") == reason for e in events))

    def test_mode_selector_never_changes_permit(self):
        sim = probe.Simulator(self.fixture, self.scenarios["stable-short"], "HYBRID_C")
        before = sim.guard.permit
        parent = sim.ledger.append(1, "PERCEPTION", "TEST_SIGNAL", {})
        sim.switch_topology("COUPLED", "HARD_TARGET_CONTRADICTION", parent)
        self.assertEqual(sim.guard.permit, before)

    def test_semantic_repeatability(self):
        for sid, s in self.scenarios.items():
            for mode in probe.MODES:
                a, _ = probe.run_once(self.fixture, s, mode)
                b, _ = probe.run_once(self.fixture, s, mode)
                self.assertEqual(a["semanticDigest"], b["semanticDigest"], (sid, mode))

    def test_benchmark_writes_evidence(self):
        with tempfile.TemporaryDirectory() as td:
            out = Path(td)
            bundle = probe.benchmark(ROOT / "scenarios.json", out, 2)
            probe.build_summary(bundle, out)
            self.assertTrue(bundle["repeatability"]["allRepeatable"])
            self.assertFalse(bundle["repeatability"]["ledgerValidationErrors"])
            for name in ("raw-metrics.json", "repeatability.json", "BENCHMARK-SUMMARY.md", "probe-summary.json", "return-packet.json"):
                self.assertTrue((out / name).is_file(), name)
            self.assertTrue((out / "causal-ledgers").is_dir())


if __name__ == "__main__":
    unittest.main()
