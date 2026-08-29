import unittest
from probe import *

class V032(unittest.TestCase):
    def test_discovery_candidate_not_change(self):
        out=translate(sample_payloads()[0]); self.assertEqual(out["reasoningEffect"],"COUPLED_UNCERTAINTY"); self.assertFalse(out["executionPermissionCreated"])
    def test_discipline_is_context(self): self.assertEqual(translate(sample_payloads()[1])["reasoningEffect"],"COUPLED_CONTEXT")
    def test_intersection_is_context(self): self.assertEqual(translate(sample_payloads()[2])["reasoningEffect"],"COUPLED_CONTEXT")
    def test_template_verified_mechanic_not_truth(self):
        out=translate(sample_payloads()[3]); self.assertEqual(out["reasoningEffect"],"COUPLED_CONTEXT"); self.assertIn("VERIFIED_MECHANIC_REFERENCE",[x["class"] for x in out["candidates"]])
    def test_current_bound_contradiction_can_pass_reasoning_change(self):
        x={"schema":VERIFIER_SCHEMA,"verdict":"CONTRADICTS_CURRENT","freshness":"CURRENT","provenanceVerified":True,"verifierId":"native-1","authority":"NONE"}
        self.assertEqual(translate(x)["reasoningEffect"],"PASS_CHANGE")
    def test_current_support_confirms(self):
        x={"schema":VERIFIER_SCHEMA,"verdict":"SUPPORTS_CURRENT","freshness":"CURRENT","provenanceVerified":True,"verifierId":"native-1","authority":"NONE"}
        self.assertEqual(translate(x)["reasoningEffect"],"CONFIRM_CURRENT")
    def test_stale_verifier_damped(self):
        x={"schema":VERIFIER_SCHEMA,"verdict":"CONTRADICTS_CURRENT","freshness":"STALE","provenanceVerified":True,"verifierId":"native-1","authority":"NONE"}
        self.assertEqual(translate(x)["reasoningEffect"],"COUPLED_UNCERTAINTY")
    def test_unbound_verifier_damped(self):
        x={"schema":VERIFIER_SCHEMA,"verdict":"CONTRADICTS_CURRENT","freshness":"CURRENT","provenanceVerified":False,"verifierId":"native-1","authority":"NONE"}
        self.assertEqual(translate(x)["reasoningEffect"],"COUPLED_UNCERTAINTY")
    def test_donor_authority_widening_fails(self):
        p=sample_payloads()[0]; p["authority"]="WRITE"
        with self.assertRaises(BindingError): translate(p)
    def test_nested_eye_authority_widening_fails(self):
        p=sample_payloads()[0]; p["topCandidates"][0]["authority"]={"workspaceMutation":True}
        with self.assertRaises(BindingError): translate(p)
    def test_unknown_schema_fails(self):
        with self.assertRaises(BindingError): translate({"schema":"nope","authority":"NONE"})
    def test_unknown_discovery_state_fails(self):
        p=sample_payloads()[0]; p["topCandidates"][0]["state"]="MAGIC"
        with self.assertRaises(BindingError): translate(p)
    def test_no_packet_can_execute_without_permit(self):
        self.assertEqual(execution_guard(translate(sample_payloads()[0]),None),"HOLD_NO_PERMIT")
    def test_same_execution_guard_remains_fail_closed(self):
        packet=translate(sample_payloads()[0])
        self.assertEqual(execution_guard(packet,{"revoked":True,"environment":"sim","capability":"sim.commit"}),"REFUSE_REVOKED")
        self.assertEqual(execution_guard(packet,{"revoked":False,"environment":"prod","capability":"sim.commit"}),"REFUSE_ENVIRONMENT")
        self.assertEqual(execution_guard(packet,{"revoked":False,"environment":"sim","capability":"wrong"}),"REFUSE_CAPABILITY")
        self.assertEqual(execution_guard(packet,{"revoked":False,"environment":"sim","capability":"sim.commit"}),"ALLOW")

if __name__=="__main__": unittest.main()
