"""Entry point. `main` is thin; `run` is the composition root.

Everything the program needs is constructed here and passed down, so nothing further in
carries a global or reaches back for a dependency.
"""

import sys
from collections.abc import Sequence


def run(argv: Sequence[str]) -> int:
    """Wire the program up and do the work. Returns the process exit code."""
    print("{{name}}")
    return 0


def main() -> int:
    return run(sys.argv[1:])


# The one file where running something on import is the point.
if __name__ == "__main__":
    raise SystemExit(main())
