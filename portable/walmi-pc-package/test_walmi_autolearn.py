from __future__ import annotations

import json
import unittest
from pathlib import Path
from tempfile import TemporaryDirectory
from unittest.mock import patch

import walmi_autolearn


class NormalizeProjectionTests(unittest.TestCase):
    def test_positive_response_becomes_user_assistant_conversation(self) -> None:
        row = walmi_autolearn.normalize_projection(
            {
                "schema": walmi_autolearn.LEARNING_SCHEMA,
                "trainingReady": True,
                "prompt": "What happened?",
                "targetResponse": "Use the observed result.",
                "learningRecordSha256": "a" * 64,
                "outcomeSignal": "HELPFUL",
            },
            1,
        )
        self.assertEqual(
            row["messages"],
            [
                {"role": "user", "content": "What happened?"},
                {"role": "assistant", "content": "Use the observed result."},
            ],
        )

    def test_failed_attempt_remains_tool_evidence(self) -> None:
        row = walmi_autolearn.normalize_projection(
            {
                "schema": walmi_autolearn.TRAJECTORY_SCHEMA,
                "sourceReceiptSha256": "b" * 64,
                "outcomeSignal": "INCONCLUSIVE",
                "targetKind": "OUTCOME_CONDITIONED_REFLECTION",
                "supervisedRoles": ["assistant"],
                "messages": [
                    {"role": "user", "content": "Build it."},
                    {"role": "tool", "content": "The attempt stopped."},
                    {"role": "assistant", "content": "Resume from the missing capability."},
                ],
            },
            1,
        )
        self.assertEqual(row["messages"][1]["role"], "tool")
        self.assertEqual(row["messages"][-1]["role"], "assistant")

    def test_failed_attempt_cannot_be_added_to_supervised_roles(self) -> None:
        with self.assertRaisesRegex(ValueError, "supervise only the assistant"):
            walmi_autolearn.normalize_projection(
                {
                    "schema": walmi_autolearn.TRAJECTORY_SCHEMA,
                    "sourceReceiptSha256": "c" * 64,
                    "targetKind": "OUTCOME_CONDITIONED_REFLECTION",
                    "supervisedRoles": ["tool", "assistant"],
                    "messages": [
                        {"role": "user", "content": "Build it."},
                        {"role": "tool", "content": "Failed attempt."},
                        {"role": "assistant", "content": "Repair lesson."},
                    ],
                },
                1,
            )

    def test_growth_compose_masks_non_assistant_roles(self) -> None:
        value = walmi_autolearn.compose(3, "experience/walmi", 2, 0.001)
        stage = value["stages"][0]
        self.assertEqual(stage["objective"], "assistant-response-modeling")
        self.assertEqual(stage["conversation"]["supervised_roles"], ["assistant"])
        self.assertEqual(value["interaction"]["template"], "user-assistant-v1")

    def test_each_growth_generation_gets_a_new_corpus_identity(self) -> None:
        first = walmi_autolearn.generation_corpus(1)
        second = walmi_autolearn.generation_corpus(2)
        self.assertEqual(first, "experience/walmi-generation-000001")
        self.assertEqual(second, "experience/walmi-generation-000002")
        self.assertNotEqual(first, second)
        with self.assertRaisesRegex(ValueError, "positive"):
            walmi_autolearn.generation_corpus(0)

    def test_menu_index_gap_gets_bounded_weight_only_in_derived_view(self) -> None:
        curriculum = walmi_autolearn.normalize_projection(
            {
                "schema": walmi_autolearn.TRAJECTORY_SCHEMA,
                "sourceReceiptSha256": "d" * 64,
                "outcomeSignal": "HARMFUL",
                "targetKind": "OUTCOME_CONDITIONED_REFLECTION",
                "supervisedRoles": ["assistant"],
                "messages": [
                    {
                        "role": "user",
                        "content": "WALMI_MENU_INDEX_V1\nworld=fixture\nChoose one visible action. Return one digit only.\n0=A\n1=B",
                    },
                    {"role": "tool", "content": "player failure: invalid menu index; repair selected B"},
                    {"role": "assistant", "content": "1"},
                ],
            },
            1,
        )
        base = walmi_autolearn.normalize_projection(
            {
                "schema": walmi_autolearn.LEARNING_SCHEMA,
                "trainingReady": True,
                "prompt": "Normal experience",
                "targetResponse": "Normal lesson",
                "learningRecordSha256": "e" * 64,
            },
            2,
        )
        original_id = curriculum["id"]
        expanded, replay_count, excluded_count = walmi_autolearn.expand_training_view(
            [base, curriculum]
        )
        self.assertEqual(len(expanded), 1 + walmi_autolearn.MENU_INDEX_CURRICULUM_REPLAYS)
        self.assertEqual(replay_count, walmi_autolearn.MENU_INDEX_CURRICULUM_REPLAYS - 1)
        self.assertEqual(excluded_count, 0)
        self.assertEqual(curriculum["id"], original_id)
        curriculum_rows = [
            row for row in expanded if row["curriculumCategory"] == "SIMULATOR_MENU_INDEX_CONTRACT_GAP"
        ]
        self.assertEqual(len({row["id"] for row in curriculum_rows}), len(curriculum_rows))
        self.assertEqual([row["curriculumReplay"] for row in curriculum_rows], [1, 2, 3, 4])
        self.assertNotIn(walmi_autolearn.CURRICULUM_REPLAY_MARKER, curriculum_rows[0]["messages"][-2]["content"])
        self.assertEqual(
            [row["messages"][-1]["content"] for row in curriculum_rows],
            ["1", "1", "1", "1"],
        )
        replay_evidence = [row["messages"][-2]["content"] for row in curriculum_rows]
        self.assertEqual(len(set(replay_evidence)), len(replay_evidence))
        self.assertIn("DERIVED_REPLAY=4/4", replay_evidence[-1])
        self.assertNotIn(walmi_autolearn.CURRICULUM_REPLAY_MARKER, curriculum["messages"][-2]["content"])

    def test_neutral_menu_self_choice_stays_direct_but_leaves_derived_training_view(self) -> None:
        neutral = walmi_autolearn.normalize_projection(
            {
                "schema": walmi_autolearn.TRAJECTORY_SCHEMA,
                "sourceReceiptSha256": "f" * 64,
                "outcomeSignal": "INCONCLUSIVE",
                "targetKind": "OUTCOME_CONDITIONED_REFLECTION",
                "supervisedRoles": ["assistant"],
                "messages": [
                    {
                        "role": "user",
                        "content": "WALMI_MENU_INDEX_V1\nworld=fixture\nChoose one visible action. Return one digit only.\n0=A\n1=B\n4=E",
                    },
                    {"role": "tool", "content": "turn 1: accepted visible action E"},
                    {"role": "assistant", "content": "4"},
                ],
            },
            1,
        )
        base = walmi_autolearn.normalize_projection(
            {
                "schema": walmi_autolearn.LEARNING_SCHEMA,
                "trainingReady": True,
                "prompt": "Grounded helpful experience",
                "targetResponse": "Keep its verified lesson.",
                "learningRecordSha256": "1" * 64,
                "outcomeSignal": "HELPFUL",
            },
            2,
        )
        original = json.loads(json.dumps(neutral))
        expanded, replay_count, excluded_count = walmi_autolearn.expand_training_view(
            [base, neutral]
        )
        self.assertEqual([row["id"] for row in expanded], [base["id"]])
        self.assertEqual(replay_count, 0)
        self.assertEqual(excluded_count, 1)
        self.assertEqual(neutral, original)
        self.assertEqual(
            walmi_autolearn.curriculum_category(neutral),
            walmi_autolearn.NEUTRAL_MENU_SELF_CHOICE,
        )

    def test_current_run_corpus_records_reads_persisted_bom(self) -> None:
        with TemporaryDirectory() as temporary:
            model_root = Path(temporary)
            model = model_root / "walmi"
            run_directory = model / "runs" / "0001-test"
            run_directory.mkdir(parents=True)
            (model / "MODEL-BOM.json").write_text(
                json.dumps(
                    {
                        "current_run_id": "run-1",
                        "runs": [{"id": "run-1", "run_bom": "runs/0001-test/RUN-BOM.json"}],
                    }
                ),
                encoding="utf-8",
            )
            (run_directory / "RUN-BOM.json").write_text(
                json.dumps({"corpus_bom": {"totals": {"docs": 17}}}),
                encoding="utf-8",
            )
            self.assertEqual(walmi_autolearn.current_run_corpus_records(model_root, "walmi"), 17)

    def test_external_data_home_keeps_personal_state_outside_runtime(self) -> None:
        with TemporaryDirectory() as temporary:
            root = Path(temporary)
            runtime = root / "runtime"
            data = root / "travel-data"
            script_host = runtime.resolve()
            with patch.dict(
                "os.environ",
                {"WALMI_HOME": str(script_host), "WALMI_DATA_HOME": str(data)},
                clear=False,
            ):
                resolved = Path(__import__("os").environ["WALMI_DATA_HOME"]).resolve()
            self.assertEqual(resolved, data.resolve())
            self.assertNotEqual(resolved, script_host)


if __name__ == "__main__":
    unittest.main()
