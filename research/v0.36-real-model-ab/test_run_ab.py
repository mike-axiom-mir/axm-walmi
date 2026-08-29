from __future__ import annotations

import json
import contextlib
import io
import tempfile
import unittest
from pathlib import Path

import run_ab


ROOT = Path(__file__).resolve().parent


def trace(hold: bool) -> dict:
    return {
        "rowCount": 1,
        "parseError": None,
        "sha256": "trace",
        "lastRow": {
            "schema": "axm.waldo.live-neural-hybrid-trace/v0.35",
            "runtime": "v0.35",
            "authority": "NONE",
            "delegate_called": True,
            "prompt_preserved_exact": True,
            "grounding_applied_at": "RESPONSE_BOUNDARY",
            "deterministic_fallback": "HOLD_UNRESOLVED_HIGH_CONSEQUENCE" if hold else "NONE",
        },
    }


def receipt(mode: str, text: str, finish: str, hold: bool = False) -> dict:
    value = {
        "mode": mode,
        "returnCode": 0,
        "timedOut": False,
        "durationNs": 1_000_000,
        "parseError": None,
        "finishReason": finish,
        "answerText": text,
    }
    if mode == run_ab.HYBRID_MODE:
        value["trace"] = trace(hold)
    return value


class ContractTests(unittest.TestCase):
    def setUp(self) -> None:
        self.contract = json.loads((ROOT / "contract.json").read_text(encoding="utf-8"))
        self.policy = json.loads((ROOT / "scoring-policy.json").read_text(encoding="utf-8"))
        self.case_set = json.loads((ROOT / "development-cases.json").read_text(encoding="utf-8"))

    def test_checked_in_contracts_validate(self) -> None:
        run_ab.validate_contract(self.contract)
        run_ab.validate_scoring_policy(self.policy)
        run_ab.validate_cases(self.case_set, "development")

    def test_development_fixture_is_balanced_and_contains_both_costs(self) -> None:
        cases = self.case_set["cases"]
        self.assertEqual(len(cases), 18)
        self.assertEqual(sum(case["pairOrder"] == "RAW_FIRST" for case in cases), 9)
        self.assertEqual(sum(case["pairOrder"] == "HYBRID_FIRST" for case in cases), 9)
        self.assertGreater(sum(run_ab.expected_hold(case["grounding"]) for case in cases), 0)
        self.assertGreater(sum(not run_ab.expected_hold(case["grounding"]) for case in cases), 0)

    def test_hold_rule_matches_v035(self) -> None:
        for state in ("STABLE", "UNCERTAIN", "CONFLICT"):
            for consequence in ("LOW", "HIGH"):
                got = run_ab.expected_hold({"state": state, "consequence": consequence})
                self.assertEqual(got, consequence == "HIGH" and state in {"UNCERTAIN", "CONFLICT"})

    def test_mode_changes_environment_not_command(self) -> None:
        command = run_ab.build_chat_command(Path("/tmp/waldo"), "model-a", "same prompt", {"maxTokens": 64, "temperature": 0.0, "topP": 1.0, "seed": 7})
        raw = run_ab.mode_environment({"KEEP": "yes"}, run_ab.RAW_MODE, Path("g.json"), Path("t.jsonl"))
        hybrid = run_ab.mode_environment({"KEEP": "yes"}, run_ab.HYBRID_MODE, Path("g.json"), Path("t.jsonl"))
        self.assertEqual(command, run_ab.build_chat_command(Path("/tmp/waldo"), "model-a", "same prompt", {"maxTokens": 64, "temperature": 0.0, "topP": 1.0, "seed": 7}))
        self.assertEqual(raw["WALDO_AXM_HYBRID"], "raw")
        self.assertNotIn("WALDO_AXM_GROUNDING_FILE", raw)
        self.assertEqual(hybrid["WALDO_AXM_HYBRID"], "hybrid")
        self.assertEqual(hybrid["WALDO_AXM_GROUNDING_FILE"], "g.json")
        self.assertEqual(raw["KEEP"], hybrid["KEEP"])

    def test_normalization_is_bounded_and_explicit(self) -> None:
        self.assertEqual(run_ab.normalize_answer("  DELTA\n"), "delta")
        self.assertTrue(run_ab.score_oracle("Paris", {"kind": "NORMALIZED_EXACT", "value": "PARIS"}))
        self.assertFalse(run_ab.score_oracle("Paris.", {"kind": "NORMALIZED_EXACT", "value": "Paris"}))

    def test_correct_high_answer_withheld_is_suppression_not_correctness(self) -> None:
        case = next(case for case in self.case_set["cases"] if case["id"] == "dev-10-missing-consequence-high")
        result = run_ab.evaluate_case(
            case,
            {
                run_ab.RAW_MODE: receipt(run_ab.RAW_MODE, "81", "max_tokens"),
                run_ab.HYBRID_MODE: receipt(run_ab.HYBRID_MODE, run_ab.HOLD_TEXT, "axm-grounding-hold", True),
            },
        )
        self.assertTrue(result["rawCorrect"])
        self.assertTrue(result["observedHold"])
        self.assertIsNone(result["hybridCorrectWhenExposed"])
        metrics = run_ab.summarize([result])
        self.assertEqual(metrics["correctHighAnswersSuppressed"], 1)
        self.assertEqual(metrics["hybridOracleCorrectWhenExposed"], 0)

    def test_incorrect_high_answer_withheld_is_prevented_exposure(self) -> None:
        case = next(case for case in self.case_set["cases"] if case["id"] == "dev-11-conflict-high")
        result = run_ab.evaluate_case(
            case,
            {
                run_ab.RAW_MODE: receipt(run_ab.RAW_MODE, "9", "max_tokens"),
                run_ab.HYBRID_MODE: receipt(run_ab.HYBRID_MODE, run_ab.HOLD_TEXT, "axm-grounding-hold", True),
            },
        )
        metrics = run_ab.summarize([result])
        self.assertEqual(metrics["incorrectHighAnswersWithheld"], 1)
        self.assertEqual(metrics["correctHighAnswersSuppressed"], 0)

    def test_low_passthrough_compares_exact_output(self) -> None:
        case = next(case for case in self.case_set["cases"] if case["id"] == "dev-01-stable-low")
        result = run_ab.evaluate_case(
            case,
            {
                run_ab.RAW_MODE: receipt(run_ab.RAW_MODE, "42", "max_tokens"),
                run_ab.HYBRID_MODE: receipt(run_ab.HYBRID_MODE, "42", "max_tokens", False),
            },
        )
        self.assertTrue(result["exactPassthroughMatch"])
        self.assertFalse(result["policyFailures"])

    def test_recovery_pair_requires_hold_then_release(self) -> None:
        before_case = next(case for case in self.case_set["cases"] if case["id"] == "dev-15-change-missing-high")
        after_case = next(case for case in self.case_set["cases"] if case["id"] == "dev-16-change-recovered-high")
        before = run_ab.evaluate_case(
            before_case,
            {
                run_ab.RAW_MODE: receipt(run_ab.RAW_MODE, "CANCELLED", "max_tokens"),
                run_ab.HYBRID_MODE: receipt(run_ab.HYBRID_MODE, run_ab.HOLD_TEXT, "axm-grounding-hold", True),
            },
        )
        after = run_ab.evaluate_case(
            after_case,
            {
                run_ab.RAW_MODE: receipt(run_ab.RAW_MODE, "CANCELLED", "max_tokens"),
                run_ab.HYBRID_MODE: receipt(run_ab.HYBRID_MODE, "CANCELLED", "max_tokens", False),
            },
        )
        metrics = run_ab.summarize([before, after])
        self.assertEqual(metrics["recoveryPairsEvaluated"], 1)
        self.assertEqual(metrics["recoveryPairsSatisfied"], 1)

    def test_output_path_is_never_overwritten(self) -> None:
        with tempfile.TemporaryDirectory() as directory:
            with self.assertRaises(run_ab.ContractError):
                run_ab.prepare_output(Path(directory))

    def test_heldout_freeze_rejects_development_cases(self) -> None:
        inputs = {"contractSha256": "c", "scoringPolicySha256": "s", "runnerSha256": "r", "casesSha256": "dev"}
        freeze = {
            "schema": run_ab.FREEZE_SCHEMA,
            "authority": "NONE",
            "policyFrozenBeforeHeldout": True,
            "heldoutCaseCountAtFreeze": 0,
            "contractSha256": "c",
            "scoringPolicySha256": "s",
            "runnerSha256": "r",
            "developmentCasesSha256": "dev",
            "waldoBinarySha256": "w",
            "modelIdentity": {"model": "m"},
            "generationOptions": {"seed": 1},
        }
        with self.assertRaises(run_ab.ContractError):
            run_ab.validate_freeze(freeze, inputs, "dev", "w", {"model": "m"}, {"seed": 1})

    def test_full_runner_with_contract_faithful_fake_session(self) -> None:
        answers = {case["prompt"]: case["oracle"]["value"] for case in self.case_set["cases"]}
        fake_source = f'''#!/usr/bin/env python3
import hashlib, json, os, sys
answers = {answers!r}
args = sys.argv[1:]
if "summary" in args:
    print(json.dumps({{"model":"test-model","identity":"summary-v1"}}, sort_keys=True))
    raise SystemExit(0)
if "bom" in args:
    print(json.dumps({{"model":"test-model","identity":"bom-v1"}}, sort_keys=True))
    raise SystemExit(0)
chat = args.index("chat")
prompt = args[chat + 2]
answer = answers[prompt]
hold = False
if os.environ.get("WALDO_AXM_HYBRID") == "hybrid":
    grounding = json.load(open(os.environ["WALDO_AXM_GROUNDING_FILE"], encoding="utf-8"))
    hold = grounding["consequence"] == "HIGH" and grounding["state"] in ("UNCERTAIN", "CONFLICT")
    trace = {{
        "schema":"axm.waldo.live-neural-hybrid-trace/v0.35",
        "runtime":"v0.35",
        "authority":"NONE",
        "delegate_called":True,
        "prompt_preserved_exact":True,
        "grounding_applied_at":"RESPONSE_BOUNDARY",
        "deterministic_fallback":"HOLD_UNRESOLVED_HIGH_CONSEQUENCE" if hold else "NONE",
        "neural_output_sha256":hashlib.sha256(answer.encode()).hexdigest()
    }}
    with open(os.environ["WALDO_AXM_HYBRID_TRACE"], "a", encoding="utf-8") as handle:
        handle.write(json.dumps(trace, sort_keys=True) + "\\n")
if hold:
    answer = {run_ab.HOLD_TEXT!r}
    finish = "axm-grounding-hold"
else:
    finish = "max_tokens"
print(json.dumps({{"model":"test-model","prompt":prompt,"result":{{"text":answer,"finish_reason":finish}}}}, sort_keys=True))
'''
        with tempfile.TemporaryDirectory() as directory:
            root = Path(directory)
            fake = root / "waldo"
            fake.write_text(fake_source, encoding="utf-8")
            fake.chmod(0o700)
            output = root / "run"
            with contextlib.redirect_stdout(io.StringIO()):
                code = run_ab.main(["--waldo", str(fake), "--model", "test-model", "--output", str(output), "--timeout-seconds", "5"])
            self.assertEqual(code, 0)
            public = json.loads((output / "PUBLIC-SUMMARY.json").read_text(encoding="utf-8"))
            self.assertEqual(public["status"], "COMPLETED")
            self.assertEqual(public["metrics"]["expectedHolds"], 7)
            self.assertEqual(public["metrics"]["observedHolds"], 7)
            self.assertEqual(public["metrics"]["correctHighAnswersSuppressed"], 7)
            self.assertEqual(public["metrics"]["lowPassthroughCompared"], 9)
            self.assertEqual(public["metrics"]["lowPassthroughExactMatch"], 9)
            self.assertEqual(public["metrics"]["recoveryPairsSatisfied"], 1)
            self.assertTrue((output / "preheldout-freeze.json").is_file())


if __name__ == "__main__":
    unittest.main()
