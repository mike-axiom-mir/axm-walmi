import unittest
import probe as p
from probe import *

class V033(unittest.TestCase):
    def test_same_root_many_views_not_independent(self):
        reps=[report("eye","NEW","STRONG",8,provenance="root-X"),report("discipline","NEW","STRONG",5,provenance="root-X")]
        self.assertEqual(p._candidate_counts(reps)["NEW"],13)
        self.assertEqual(p._root_counts(reps)["NEW"],1)
    def test_two_roots_are_independent_for_counting(self):
        reps=[report("a","NEW","MODERATE",1),report("b","NEW","MODERATE",1)]
        self.assertEqual(p._root_counts(reps)["NEW"],2)
    def test_weak_views_do_not_become_corroboration(self):
        reps=[report("a","NEW","WEAK",20),report("b","NEW","WEAK",20)]
        self.assertEqual(p._root_counts(reps),{})
    def test_root_aware_avoids_short_fanout_false(self):
        s=scenario(33000)  # seed%12 == 0
        self.assertEqual(s["family"],"fanout-false")
        self.assertTrue(run_one(s,"ROOT_AWARE_MEMBRANE")["finalCorrect"])
        self.assertFalse(run_one(s,"RAW_COUPLED")["finalCorrect"])
    def test_independent_real_passes(self):
        seed=next(x for x in range(33000,33100) if scenario(x)["family"]=="independent-real")
        s=scenario(seed); self.assertTrue(run_one(s,"ROOT_AWARE_MEMBRANE")["finalCorrect"])
    def test_persistence_fallback_is_bounded_and_visible(self):
        seed=next(x for x in range(33000,33100) if scenario(x)["family"]=="single-real-long")
        r=run_one(scenario(seed),"ROOT_AWARE_MEMBRANE")
        self.assertGreaterEqual(r["fallbackActivations"],1); self.assertTrue(r["finalCorrect"])
    def test_independent_false_is_retained_failure(self):
        seed=next(x for x in range(33000,33100) if scenario(x)["family"]=="independent-false")
        r=run_one(scenario(seed),"ROOT_AWARE_MEMBRANE")
        self.assertGreater(r["falsePreempts"],0)
    def test_verified_false_is_retained_failure(self):
        seed=next(x for x in range(33000,33100) if scenario(x)["family"]=="verified-false")
        r=run_one(scenario(seed),"ROOT_AWARE_MEMBRANE")
        self.assertGreater(r["falsePreempts"],0)
    def test_no_mode_changes_authority(self):
        for seed in range(33001,33013):
            for m in MODES:
                for row in run_one(scenario(seed),m)["ledger"]:
                    self.assertEqual(row["authority"],"NONE")
    def test_repeatability(self):
        seeds=list(range(33001,33049)); self.assertEqual(semantic_digest(seeds),semantic_digest(seeds))

if __name__=="__main__": unittest.main()
