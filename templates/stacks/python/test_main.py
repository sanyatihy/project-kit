"""The installed entry points work outside the source checkout."""

import subprocess
import sys
from pathlib import Path

import pytest


@pytest.mark.parametrize(
    "command",
    [
        [sys.executable, "-m", "{{pkg}}"],
        ["{{name}}"],
    ],
)
def test_installed_entry_points(command: list[str], tmp_path: Path) -> None:
    result = subprocess.run(
        command, cwd=tmp_path, capture_output=True, text=True, check=True, timeout=10
    )
    assert result.stdout == "{{name}}\n"
    assert result.stderr == ""
