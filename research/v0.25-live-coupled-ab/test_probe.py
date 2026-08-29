from __future__ import annotations

import dataclasses
import json
import tempfile
import unittest
from pathlib import Path

import probe


ROOT = Path(__file__).parent


class LiveCoupledABProbeTests(unittest.TestCase):
    @classmethod
    def setUpClass(cls) -> None:
        cls.fixture = json.loads((ROOT / "scenarios.json").read_text(encoding="utf-8"))
        cls.scenarios = {s["id"]: s for s in cls.fixture["scenarios"]}

    def pair(self, scenario_id: str):
        scenario = self.scenarios[scenario_id]
        a, a_events = probe.run_once(self.fixture, scenario, "SEQUENTIAL_A")
        b, b_events = probe.run_once(self.fixture, scenario, "COUPLED_B")
        return a, a_events, b, b_events

    def test_same_fixture_and_capabilities_feed_both_modes(self) -> None:
        for scenario_id in self.scenarios:
            a, _ae, b, _be = self.pair(scenario_id)
            self.assertEqual(a["inputDigest"], b["inputDigest"])
            self.assertEqual(a["seed"], b["seed"])
            self.assertEqual(a["authority"], "NONE")
            self.assertEqual(b["authority"], "NONE")
            self.assertFalse(a["canon"])
            self.assertFalse(b["canon"])

    def test_early_shift_is_cancelled_before_coupled_execution(self) -> None:
        a, _ae, b, _be = self.pair("target-shift-early")
        am, bm = a["metrics"], b["metrics"]
        self.assertGreaterEqual(bm["builder_proposals_revised_before_execution"], 1)
        self.assertLess(bm["contradictionToCancelTicks"], am["contradictionToCancelTicks"])
        self.assertLess(bm["contradictionToCorrectionTicks"], am["contradictionToCorrectionTicks"])
        self.assertLess(bm["unnecessary_actions"], am["unnecessary_actions"])
        self.assertTrue(b["outcome"]["correctFinalPlacement"])

    def test_coupling_can_be_worse_and_result_is_preserved(self) -> None:
        a, _ae, b, _be = self.pair("missing-evidence")
        self.assertGreater(a["outcome"]["qualityScore"], b["outcome"]["qualityScore"])
        self.assertGreaterEqual(b["metrics"]["holds"], 1)
        noisy_a, _nae, noisy_b, _nbe = self.pair("measurement-disagreement")
        self.assertGreater(noisy_b["metrics"]["events"], noisy_a["metrics"]["events"])
        self.assertGreaterEqual(noisy_b["metrics"]["false_contradictions"], 1)

    def test_revocation_stops_future_actions_without_rewriting_past(self) -> None:
        _a, a_events, _b, b_events = self.pair("permit-revoked")
        for events in (a_events, b_events):
            before = [
                e for e in events
                if e["logicalTime"] < 9 and (
                    e["kind"] == "ACTION_EXECUTED"
                    or (e["kind"] == "ACTION_RESULT" and e["payload"].get("outcome") == "EXECUTED")
                )
            ]
            refused = [e for e in events if e["kind"] == "REFUSED" and e["payload"].get("reason") == "EXECUTION_PERMIT_REVOKED"]
            refused += [e for e in events if e["kind"] == "ACTION_RESULT" and e["payload"].get("reason") == "EXECUTION_PERMIT_REVOKED"]
            after = [
                e for e in events
                if e["logicalTime"] >= 9 and (
                    e["kind"] == "ACTION_EXECUTED"
                    or (e["kind"] == "ACTION_RESULT" and e["payload"].get("outcome") == "EXECUTED")
                )
            ]
            self.assertTrue(before)
            self.assertTrue(refused)
            self.assertFalse(after)
            revocation = [e for e in events if e["kind"] == "PERMIT_REVOKED"]
            self.assertEqual(len(revocation), 1)

    def test_exact_authority_envelope_fail_closed(self) -> None:
        expectations = {
            "wrong-environment": ("ENVIRONMENT_SCOPE_MISMATCH", 1),
            "wrong-capability": ("CAPABILITY_SCOPE_MISMATCH", 1),
            "no-permit": ("EXECUTION_PERMIT_ABSENT", 0),
            "no-actuator-route": ("ACTUATOR_ROUTE_ABSENT", 0),
        }
        for scenario_id, (reason, violations) in expectations.items():
            for mode in ("SEQUENTIAL_A", "COUPLED_B"):
                result, events = probe.run_once(self.fixture, self.scenarios[scenario_id], mode)
                self.assertEqual(result["metrics"]["actions_executed"], 0)
                self.assertEqual(result["metrics"]["authority_violations_attempted"], violations)
                self.assertEqual(result["metrics"]["authority_violations_refused"], violations)
                self.assertTrue(any(e["payload"].get("reason") == reason for e in events))

    def test_reasoning_cannot_expand_immutable_permit(self) -> None:
        runtime = probe.CoupledRuntime(self.fixture, self.scenarios["no-permit"])
        self.assertFalse(runtime.guard.permit.present)
        with self.assertRaises(dataclasses.FrozenInstanceError):
            runtime.guard.permit.present = True  # type: ignore[misc]
        runtime.ledger.append(0, "BUILDER", "AUTHORITY_ESCALATION_REQUEST", {"permit": "VALID"})
        outcome, reason = runtime.guard.check(probe.ENVIRONMENT, probe.CAPABILITY)
        self.assertEqual((outcome, reason), ("HOLD", "EXECUTION_PERMIT_ABSENT"))

    def test_causal_ledgers_are_complete_and_digest_valid(self) -> None:
        for scenario_id in self.scenarios:
            for mode in ("SEQUENTIAL_A", "COUPLED_B"):
                result, events = probe.run_once(self.fixture, self.scenarios[scenario_id], mode)
                self.assertEqual(probe.validate_ledger(events, mode, scenario_id), [])
                receipt = result["evidenceReceipt"]
                self.assertEqual(receipt["completeness"], 1.0)
                self.assertTrue(receipt["validCausalReferences"])

    def test_stale_writes_and_duplicate_events_are_bounded(self) -> None:
        result, _events = probe.run_once(self.fixture, self.scenarios["target-shift-early"], "COUPLED_B")
        self.assertGreater(result["metrics"]["stale_writes_rejected"], 0)
        runtime = probe.CoupledRuntime(self.fixture, self.scenarios["stable-short"])
        runtime.schedule(1, "OBSERVATION_PUBLISHED", "PERCEPTION", {}, dedupe_key="same")
        runtime.schedule(1, "OBSERVATION_PUBLISHED", "PERCEPTION", {}, dedupe_key="same")
        self.assertEqual(runtime.metrics.duplicate_events_suppressed, 1)
        self.assertLessEqual(len(runtime.queue), self.fixture["limits"]["maxQueueDepth"])

    def test_semantic_repeatability_excludes_machine_timing(self) -> None:
        for scenario_id in self.scenarios:
            for mode in ("SEQUENTIAL_A", "COUPLED_B"):
                first, _ = probe.run_once(self.fixture, self.scenarios[scenario_id], mode)
                second, _ = probe.run_once(self.fixture, self.scenarios[scenario_id], mode)
                self.assertEqual(first["semanticDigest"], second["semanticDigest"])
                self.assertIn("wallClockNanoseconds", first["resourceObservation"])

    def test_benchmark_writes_all_required_evidence(self) -> None:
        with tempfile.TemporaryDirectory() as tmp:
            output = Path(tmp)
            bundle = probe.benchmark(ROOT / "scenarios.json", output, repeats=2)
            probe.build_summary(bundle, output)
            probe.finalize_artifacts(bundle, ROOT, output)
            required = {
                "raw-metrics.json",
                "repeatability.json",
                "failure-evidence.json",
                "BENCHMARK-SUMMARY.md",
                "probe-summary.json",
                "return-packet.json",
                "artifact-manifest.json",
            }
            self.assertTrue(required.issubset({p.name for p in output.iterdir()}))
            self.assertTrue((output / "causal-ledgers").is_dir())


if __name__ == "__main__":
    unittest.main()
