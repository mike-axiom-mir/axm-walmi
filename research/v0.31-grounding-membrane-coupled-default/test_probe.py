from __future__ import annotations

import json
import unittest
from pathlib import Path

import probe

ROOT=Path(__file__).parent

class GroundingMembraneTests(unittest.TestCase):
    @classmethod
    def setUpClass(cls):
        cls.fx=json.loads((ROOT/"scenarios.json").read_text())
        cls.fixed={x["id"]:x for x in cls.fx["scenarios"]}

    def runm(self,sid,mode):
        return probe.run_once(self.fx,self.fixed[sid],mode)

    def test_default_is_coupled_membrane_with_sequential_fallback(self):
        self.assertEqual(self.fx["defaultArchitecture"],"COUPLED_MEMBRANE_FALLBACK_D")
        self.assertEqual(self.fx["fallbackArchitecture"],"SEQUENTIAL_A")

    def test_authority_is_none_everywhere(self):
        for mode in probe.MODES:
            r,_=self.runm("stable-clean",mode)
            self.assertEqual(r["authority"],"NONE")
            self.assertFalse(r["canon"])

    def test_missing_is_damped_not_contradiction(self):
        raw,_=self.runm("missing-only","COUPLED_B")
        grounded,events=self.runm("missing-only","COUPLED_MEMBRANE_FALLBACK_D")
        self.assertTrue(any(e["kind"]=="GROUNDING_PACKET" and e["payload"]["class"]=="MISSING_NOT_CONTRADICTION" for e in events))
        self.assertEqual(grounded["metrics"].get("false_positive_preemptions",0),0)
        self.assertTrue(grounded["final"]["correctFinalPlan"])

    def test_low_noise_is_damped_while_raw_coupling_preempts(self):
        raw,_=self.runm("single-low-noise","COUPLED_B")
        grounded,events=self.runm("single-low-noise","COUPLED_MEMBRANE_FALLBACK_D")
        self.assertGreaterEqual(raw["metrics"].get("false_positive_preemptions",0),1)
        self.assertTrue(any(e["kind"]=="GROUNDING_PACKET" and e["payload"]["class"]=="DAMP_TO_UNCERTAINTY" for e in events))
        self.assertEqual(grounded["metrics"].get("false_positive_preemptions",0),0)

    def test_strong_real_change_is_not_damped(self):
        grounded,events=self.runm("strong-real-change","COUPLED_MEMBRANE_FALLBACK_D")
        self.assertTrue(grounded["final"]["correctFinalPlan"])
        self.assertTrue(any(e["kind"]=="GROUNDED_COUPLED_PREEMPT" for e in events))
        self.assertEqual(grounded["metrics"].get("false_negative_delay_ticks",0),0)

    def test_independent_corroboration_passes_change(self):
        grounded,events=self.runm("corroborated-real-change","COUPLED_MEMBRANE_FALLBACK_D")
        self.assertTrue(grounded["final"]["correctFinalPlan"])
        self.assertTrue(any(e["kind"]=="GROUNDED_COUPLED_PREEMPT" and e["payload"].get("reason")=="independent-corroboration" for e in events))

    def test_fallback_can_recover_weak_persistent_change(self):
        grounded,events=self.runm("weak-real-change-needs-fallback","COUPLED_MEMBRANE_FALLBACK_D")
        self.assertTrue(grounded["final"]["correctFinalPlan"])
        self.assertGreaterEqual(grounded["metrics"].get("fallback_activations",0),1)
        self.assertTrue(any(e["kind"]=="FALLBACK_REVIEW" for e in events))

    def test_verified_anchor_does_not_freeze_reality(self):
        grounded,events=self.runm("stale-anchor-real-change","COUPLED_MEMBRANE_FALLBACK_D")
        self.assertTrue(grounded["final"]["correctFinalPlan"])
        self.assertEqual(grounded["final"]["world"],8)
        self.assertEqual(grounded["final"]["plan"],8)

    def test_grounding_packets_are_non_authoritative(self):
        _,events=self.runm("single-high-false","COUPLED_MEMBRANE_FALLBACK_D")
        packets=[e for e in events if e["kind"]=="GROUNDING_PACKET"]
        self.assertTrue(packets)
        self.assertTrue(all(e["payload"].get("authority")=="NONE" for e in packets))

    def test_authority_cases_fail_closed_all_modes(self):
        expected={
            "wrong-environment":"ENVIRONMENT_SCOPE_MISMATCH",
            "wrong-capability":"CAPABILITY_SCOPE_MISMATCH",
            "no-permit":"EXECUTION_PERMIT_ABSENT",
            "no-actuator":"ACTUATOR_ROUTE_ABSENT",
        }
        for sid,reason in expected.items():
            for mode in probe.MODES:
                r,events=self.runm(sid,mode)
                self.assertEqual(r["metrics"].get("actions_executed",0),0,(sid,mode))
                self.assertTrue(any(e["kind"]=="COMMIT_BLOCKED" and e["payload"].get("reason")==reason for e in events),(sid,mode))

    def test_revocation_blocks_future_execution(self):
        for mode in probe.MODES:
            r,events=self.runm("permit-revoked",mode)
            self.assertTrue(any(e["kind"]=="COMMIT_BLOCKED" and e["payload"].get("reason")=="EXECUTION_PERMIT_REVOKED" for e in events),mode)

    def test_semantic_repeatability(self):
        scenarios=list(self.fixed.values())[:10]
        for s in scenarios:
            for mode in probe.MODES:
                a,_=probe.run_once(self.fx,s,mode)
                b,_=probe.run_once(self.fx,s,mode)
                self.assertEqual(a["semanticDigest"],b["semanticDigest"],(s["id"],mode))

    def test_development_generator_is_deterministic(self):
        for seed in self.fx["developmentSeeds"][:12]:
            a=probe.generate_scenario(seed,self.fx,"DEVELOPMENT")
            b=probe.generate_scenario(seed,self.fx,"DEVELOPMENT")
            self.assertEqual(a,b)
            self.assertEqual(a["split"],"DEVELOPMENT")

if __name__=="__main__":
    unittest.main()
