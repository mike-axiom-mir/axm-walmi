from __future__ import annotations

import os
import unittest
from pathlib import Path
from tempfile import TemporaryDirectory
from unittest.mock import patch

import walmi_portable_bootstrap


class PortableBootstrapTests(unittest.TestCase):
    def test_inside_accepts_child_and_rejects_sibling(self) -> None:
        with TemporaryDirectory() as temporary:
            root = Path(temporary).resolve()
            self.assertTrue(walmi_portable_bootstrap.inside(root, root / "state"))
            self.assertFalse(
                walmi_portable_bootstrap.inside(root, root.parent / "other")
            )

    def test_external_data_home_can_be_selected_explicitly(self) -> None:
        with TemporaryDirectory() as temporary:
            root = Path(temporary)
            runtime = (root / "runtime").resolve()
            data = (root / "data").resolve()
            with patch.dict(
                os.environ,
                {"WALMI_HOME": str(runtime), "WALMI_DATA_HOME": str(data)},
                clear=False,
            ):
                self.assertEqual(
                    Path(os.environ["WALMI_DATA_HOME"]).resolve(), data
                )


if __name__ == "__main__":
    unittest.main()
