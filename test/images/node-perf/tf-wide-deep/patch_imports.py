#!/usr/bin/env python3
"""Patch Python files to insert tf_compat import after __future__ imports.

This script modifies Python source files to inject the TF1 compatibility
shim import. It inserts the import after any 'from __future__' imports
to avoid SyntaxErrors (future imports must be at the start of the file).
"""
import sys
import glob

COMPAT_IMPORT = "import sys; sys.path.insert(0, \"/\"); import tf_compat\n"


def patch_file(filepath):
    """Add tf_compat import to a Python file after __future__ imports."""
    with open(filepath, "r") as f:
        lines = f.readlines()

    # Find the last line that starts with "from __future__"
    last_future_line = -1
    for i, line in enumerate(lines):
        if line.strip().startswith("from __future__"):
            last_future_line = i

    if last_future_line >= 0:
        # Insert after the last __future__ import
        insert_pos = last_future_line + 1
        lines.insert(insert_pos, COMPAT_IMPORT)
    else:
        # No __future__ imports found, insert at beginning
        lines.insert(0, COMPAT_IMPORT)

    with open(filepath, "w") as f:
        f.writelines(lines)
    print(f"Patched: {filepath}")


if __name__ == "__main__":
    for pattern in sys.argv[1:]:
        for filepath in glob.glob(pattern):
            patch_file(filepath)
