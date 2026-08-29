import unittest
import probe

class ProbeTests(unittest.TestCase):
    def custom(self, world, events, commits=None):
        return {"seed":0,"family":"custom","T":len(world),"world":world,"events":events,
                "commits":commits or {t:("HIGH" if t==2 else "LOW") for t in range(2,len(world))}}

    def e(self,t,kind,claim,root,confidence=.9,provenance=True,view=0):
        return {"t":t,"kind":kind,"claim":claim,"root":root,"confidence":confidence,
                "provenance":provenance,"view":view}

    def test_generation_is_deterministic(self):
        self.assertEqual(probe.make_scenario(34001),probe.make_scenario(34001))

    def test_alias_views_do_not_create_independent_roots(self):
        sc=self.custom(["A"]*6,[self.e(2,"source","B","same",view=i) for i in range(5)])
        self.assertEqual(probe.run_mode(sc,"ROOT_AWARE").get("preempts",0),0)

    def test_two_roots_can_preempt_root_aware(self):
        sc=self.custom(["A","A","B","B","B","B"],[self.e(2,"source","B","s1"),self.e(2,"source","B","s2")])
        self.assertEqual(probe.run_mode(sc,"ROOT_AWARE").get("preempts",0),1)

    def test_causal_mode_keeps_same_plan_reaction(self):
        sc=self.custom(["A","A","B","B","B","B"],[self.e(2,"source","B","s1"),self.e(2,"source","B","s2")])
        self.assertEqual(probe.run_mode(sc,"ROOT_AWARE").get("preempts",0),
                         probe.run_mode(sc,"ROOT_CAUSAL_GATE").get("preempts",0))

    def test_high_commit_holds_while_challenge_open(self):
        sc=self.custom(["A","A","B","B","B","B"],[self.e(2,"source","B","s1"),self.e(2,"source","B","s2")])
        self.assertEqual(probe.run_mode(sc,"ROOT_CAUSAL_GATE").get("holds",0),1)

    def test_matching_consequence_resolves_challenge(self):
        sc=self.custom(["A","A","B","B","B","B"],[
            self.e(2,"source","B","s1"),self.e(2,"source","B","s2"),self.e(3,"outcome","B","o1")])
        r=probe.run_mode(sc,"ROOT_CAUSAL_GATE")
        self.assertEqual(r.get("challengeConfirmed",0),1)

    def test_contradicting_consequence_reverts(self):
        sc=self.custom(["A"]*6,[
            self.e(2,"source","B","s1"),self.e(2,"source","B","s2"),self.e(3,"outcome","A","o1")])
        r=probe.run_mode(sc,"ROOT_CAUSAL_GATE")
        self.assertEqual(r.get("challengeRejected",0),1)
        self.assertEqual(r.get("correct",0),1)

    def test_wrong_causal_consequence_can_still_fool(self):
        sc=self.custom(["A"]*6,[
            self.e(2,"source","B","s1"),self.e(2,"source","B","s2"),self.e(3,"outcome","B","o_bad")])
        r=probe.run_mode(sc,"ROOT_CAUSAL_GATE")
        self.assertEqual(r.get("correct",1),0)

    def test_timeout_is_not_truth_and_only_releases_hold(self):
        sc=self.custom(["A","A","B","B","B","B","B"],[
            self.e(2,"source","B","s1"),self.e(2,"source","B","s2")],
            {2:"HIGH",3:"HIGH",4:"HIGH",5:"HIGH"})
        r=probe.run_mode(sc,"ROOT_CAUSAL_GATE")
        self.assertGreater(r.get("challengeTimeouts",0),0)

    def test_guard_requires_permit(self):
        self.assertEqual(probe.execution_guard(None),"HOLD_NO_PERMIT")

    def test_guard_refuses_wrong_scope(self):
        p={"revoked":False,"environment":"host","capability":"sim.commit","actuator":"sim.actuator"}
        self.assertEqual(probe.execution_guard(p),"REFUSE_ENVIRONMENT")

    def test_reasoning_hold_does_not_create_authority(self):
        p={"revoked":False,"environment":"sim","capability":"sim.commit","actuator":"sim.actuator"}
        self.assertEqual(probe.execution_guard(p,reasoning_hold=True),"HOLD_REASONING_UNRESOLVED")
        self.assertEqual(probe.execution_guard(p),"ALLOW")

if __name__=="__main__":
    unittest.main()
