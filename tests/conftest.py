"""Make `spec/patch_spec.py` importable as `patch_spec` from tests.

`spec/` is a script directory, not a package, so it isn't on `sys.path`
when pytest collects tests. Prepend it so `import patch_spec` works
without restructuring the source tree.
"""

from __future__ import annotations

import sys
from pathlib import Path

_SPEC_DIR = Path(__file__).resolve().parent.parent / "spec"
if str(_SPEC_DIR) not in sys.path:
    sys.path.insert(0, str(_SPEC_DIR))
