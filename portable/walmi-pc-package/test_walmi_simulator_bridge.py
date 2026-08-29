from __future__ import annotations

import importlib.util
import argparse
import json
import os
from pathlib import Path
import tempfile
import unittest
from unittest import mock
import zipfile


MODULE_PATH = Path(__file__).with_name("walmi_simulator_bridge.py")
SPEC = importlib.util.spec_from_file_location("walmi_simulator_bridge", MODULE_PATH)
bridge = importlib.util.module_from_spec(SPEC)
assert SPEC.loader is not None
SPEC.loader.exec_module(bridge)


class SimulatorBridgeContractTest(unittest.TestCase):
    def test_data_home_never_silently_splits_inner_state(self):
        with tempfile.TemporaryDirectory() as directory:
            host = Path(directory) / "host"
            for name in ("bin", "models", "state", "tools"):
                (host / name).mkdir(parents=True, exist_ok=True)
            self.assertEqual(
                bridge.bundled_data_home(host / "tools" / "walmi_simulator_bridge.py"),
                host.resolve(),
            )
        with mock.patch.dict(
            os.environ, {"WALMI_DATA_HOME": "", "WALMI_HOME": ""}
        ):
            with self.assertRaisesRegex(RuntimeError, "data home is ambiguous"):
                bridge.default_data_home()

    def observation(self):
        return {
            "schema": "axm.walmi.simulator-observation/v1",
            "world": "fixture",
            "stateDigest": "a" * 64,
            "visible": {"score": 1},
            "actions": [
                {"id": "a", "label": "A", "description": "first"},
                {"id": "b", "label": "B", "description": "second"},
            ],
        }

    def test_player_response_is_exact_and_menu_bounded(self):
        accepted = {"schema": bridge.PLAYER_RESPONSE_SCHEMA, "actionId": "a"}
        self.assertEqual(bridge.validate_player_response(accepted, self.observation()), accepted)
        for refused in (
            {"schema": bridge.PLAYER_RESPONSE_SCHEMA, "actionId": "hidden"},
            {"schema": bridge.PLAYER_RESPONSE_SCHEMA, "actionId": "a", "payload": {}},
            {"schema": "wrong", "actionId": "a"},
        ):
            with self.assertRaises(ValueError):
                bridge.validate_player_response(refused, self.observation())

    def test_deterministic_fixture_repeats(self):
        request = {
            "episodeId": "episode-1",
            "world": "fixture",
            "seed": "repeatable-seed",
            "turn": 2,
            "observation": self.observation(),
        }
        self.assertEqual(bridge.deterministic_player(request), bridge.deterministic_player(request))
        changed_episode = {**request, "episodeId": "episode-2"}
        self.assertEqual(bridge.deterministic_player(request), bridge.deterministic_player(changed_episode))

    def test_contract_gap_repair_stays_visible_and_preserves_failure(self):
        request = {
            "episodeId": "episode-gap",
            "world": "fixture",
            "seed": "repair-seed",
            "turn": 1,
            "observation": self.observation(),
        }
        repair = bridge.repair_player_gap(request, "model returned invalid JSON")
        self.assertEqual(repair["schema"], bridge.DETERMINISTIC_REPAIR_SCHEMA)
        self.assertIn(repair["response"]["actionId"], {"a", "b"})
        self.assertEqual(repair["failure"], "model returned invalid JSON")
        self.assertFalse(any(repair["authority"].values()))

    def test_capability_gap_record_is_stable_and_closed_authority(self):
        episode = {
            "world": "fixture",
            "player": {"kind": "model", "id": "walmi"},
            "receiptSha256": "a" * 64,
            "observedAt": "2026-08-29T00:00:00Z",
        }
        step = {
            "turn": 1,
            "requestSha256": "b" * 64,
            "outcome": {"ok": True},
            "deterministicRepair": {
                "failure": "invalid JSON",
                "hand": "deterministic-visible-action-selector/v1",
                "response": {"schema": bridge.PLAYER_RESPONSE_SCHEMA, "actionId": "a"},
            },
        }
        first = bridge.capability_gap_record(episode, step)
        second = bridge.capability_gap_record(episode, step)
        self.assertEqual(first, second)
        self.assertEqual(first["schema"], bridge.CAPABILITY_GAP_SCHEMA)
        self.assertEqual(first["gapType"], "CONTRACT")
        self.assertTrue(first["repair"]["worldActionApplied"])
        self.assertFalse(any(first["authority"].values()))

        fixture_path = MODULE_PATH.with_name("examples") / "simulator-capability-gap-v1.json"
        fixture = json.loads(fixture_path.read_text(encoding="utf-8"))
        self.assertEqual(set(fixture), set(first))
        self.assertEqual(fixture["schema"], bridge.CAPABILITY_GAP_SCHEMA)
        self.assertEqual(set(fixture["evidence"]), set(first["evidence"]))
        self.assertEqual(set(fixture["repair"]), set(first["repair"]))
        self.assertEqual(set(fixture["authority"]), set(first["authority"]))

        described = bridge.describe_attempt(step)
        self.assertIn("player failure: invalid JSON", described)
        self.assertIn("deterministic repair selected", described)
        self.assertEqual(
            bridge.experience_lesson(True, [first], [step]),
            '{"actionId":"a","schema":"axm.walmi.simulator-player-response/v1"}',
        )

    def test_model_codec_is_small_deterministic_and_translates_only_a_visible_digit(self):
        observation = self.observation()
        observation["actions"] = [
            {"id": f"action-{index}", "label": "Long visible label " + ("x" * 80)}
            for index in range(12)
        ]
        request = {
            "episodeId": "episode-codec",
            "world": "fixture",
            "seed": "codec-seed",
            "turn": 3,
            "observation": observation,
        }
        codec = bridge.model_player_codec(request)
        self.assertEqual(codec, bridge.model_player_codec(request))
        codec_fixture = json.loads(
            (MODULE_PATH.with_name("examples") / "simulator-menu-index-codec-v1.json").read_text(
                encoding="utf-8"
            )
        )
        self.assertEqual(set(codec), set(codec_fixture))
        self.assertEqual(codec["schema"], codec_fixture["schema"])
        self.assertEqual(len(codec["menuActionIds"]), bridge.MAX_MODEL_MENU_ACTIONS)
        self.assertLessEqual(len(codec["prompt"].encode("utf-8")), bridge.MAX_MODEL_PROMPT_BYTES)
        self.assertEqual(
            bridge.decode_model_choice("0", codec),
            {"schema": bridge.PLAYER_RESPONSE_SCHEMA, "actionId": codec["menuActionIds"][0]},
        )
        for refused in ("", "zero", "0\n1", "8", '{"actionId":"action-0"}'):
            with self.assertRaises(ValueError):
                bridge.decode_model_choice(refused, codec)

        short_request = {**request, "observation": self.observation()}
        fixed_codec = bridge.model_player_codec(short_request)
        self.assertEqual(len(fixed_codec["menuActionIds"]), bridge.MAX_MODEL_MENU_ACTIONS)
        self.assertEqual(set(fixed_codec["menuActionIds"]), {"a", "b"})
        decoded = bridge.decode_model_choice("7", fixed_codec)
        self.assertIn(decoded["actionId"], {"a", "b"})
        self.assertLessEqual(
            len(fixed_codec["prompt"].encode("utf-8")), bridge.MAX_MODEL_PROMPT_BYTES
        )

        request["playerCodec"] = codec
        repair = bridge.repair_player_gap(request, "model missed the index contract")
        self.assertIn(repair["response"]["actionId"], codec["menuActionIds"])
        step = {
            "turn": 3,
            "actionId": repair["response"]["actionId"],
            "outcome": {"ok": True},
            "playerCodec": codec,
            "deterministicRepair": repair,
        }
        self.assertEqual(
            bridge.experience_lesson(True, [], [step]),
            str(codec["menuActionIds"].index(step["actionId"])),
        )

    def test_canonical_digest_ignores_dictionary_insertion_order(self):
        self.assertEqual(bridge.digest({"a": 1, "b": 2}), bridge.digest({"b": 2, "a": 1}))

    def test_bundled_waldo_is_inferred_from_data_home(self):
        with tempfile.TemporaryDirectory() as directory:
            data_home = Path(directory)
            binary = data_home / "bin" / "waldo.exe"
            binary.parent.mkdir()
            binary.write_bytes(b"fixture")
            self.assertEqual(bridge.infer_waldo_bin(None, data_home), binary.resolve())
            explicit = data_home / "other-waldo.exe"
            self.assertEqual(
                bridge.infer_waldo_bin(explicit, data_home), explicit.resolve()
            )
            runtime_host = data_home / "runtime-host"
            waldo_bin = runtime_host / "bin" / "waldo.exe"
            environment = bridge.waldo_environment(data_home, waldo_bin)
            self.assertEqual(
                Path(environment["WALDO_CONFIG"]),
                (data_home / "state" / "waldo-config.json").resolve(),
            )
            self.assertEqual(
                Path(environment["PYTHONHOME"]),
                (runtime_host / "runtime" / "python").resolve(),
            )
            self.assertEqual(Path(environment["WALMI_HOME"]), runtime_host.resolve())

    def test_named_save_slot_is_sealed_resumable_and_never_overwritten(self):
        with tempfile.TemporaryDirectory() as directory:
            data_home = Path(directory)
            created = bridge.open_save_slot(
                data_home, "walmi", "theme-park", "main", "SAVE-SEED", True
            )
            self.assertFalse(created["resumed"])
            self.assertEqual(created["metadata"]["schema"], bridge.SAVE_SLOT_SCHEMA)
            self.assertEqual(
                bridge.digest({**created["metadata"], "metadataSha256": ""}),
                created["metadata"]["metadataSha256"],
            )
            bridge.write_slot_metadata(created, {"stateDigest": "state-a", "totalTurns": 2})
            resumed = bridge.open_save_slot(
                data_home, "walmi", "theme-park", "main", None, False
            )
            self.assertTrue(resumed["resumed"])
            self.assertEqual(resumed["seed"], "SAVE-SEED")
            self.assertEqual(resumed["metadata"]["totalTurns"], 2)
            with self.assertRaises(FileExistsError):
                bridge.open_save_slot(
                    data_home, "walmi", "theme-park", "main", "SAVE-SEED", True
                )
            with self.assertRaises(ValueError):
                bridge.open_save_slot(
                    data_home, "../escape", "theme-park", "main", None, False
                )

    def test_every_observed_turn_gets_its_own_projection(self):
        with tempfile.TemporaryDirectory() as directory:
            root = Path(directory)
            projection_path = root / "projections.jsonl"
            episode = {
                "episodeId": "episode-turn-projection",
                "receiptSha256": "a" * 64,
                "observedAt": "2026-08-29T00:00:00Z",
                "world": "fixture",
                "steps": [
                    {"turn": 1, "actionId": "a", "outcome": {"ok": True, "reason": "accepted"}},
                    {"turn": 2, "actionId": "b", "outcome": {"ok": True, "reason": "accepted"}},
                ],
            }
            receipt = bridge.project_episode_turns(
                episode,
                [],
                argparse.Namespace(waldo_projection_supported=False),
                root / "records.jsonl",
                projection_path,
                root,
            )
            lines = [json.loads(line) for line in projection_path.read_text(encoding="utf-8").splitlines()]
            self.assertEqual(receipt["turnsProjected"], 2)
            self.assertEqual(len(lines), 2)
            self.assertEqual(
                [line["id"] for line in lines],
                ["episode-turn-projection-turn-0001", "episode-turn-projection-turn-0002"],
            )

    def test_run_compaction_is_verified_and_keeps_full_episode_name(self):
        with tempfile.TemporaryDirectory() as directory:
            root = Path(directory)
            run = root / "sim-theme-20260829T120000.123456Z-abcdef12"
            run.mkdir()
            (run / "state.json").write_text('{"ok":true}\n', encoding="utf-8")
            archive, count, expanded = bridge.compact_run(run)
            self.assertEqual(archive.name, run.name + ".zip")
            self.assertFalse(run.exists())
            self.assertEqual(count, 1)
            self.assertGreater(expanded, 0)
            with zipfile.ZipFile(archive) as packed:
                self.assertIsNone(packed.testzip())
                self.assertIn(f"{run.name}/state.json", packed.namelist())


if __name__ == "__main__":
    unittest.main()
