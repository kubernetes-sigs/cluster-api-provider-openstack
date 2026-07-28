#!/usr/bin/env python3
"""Remove JSON tag options from generated API reference field names."""

import re
import sys
from pathlib import Path


FIELD_TAG = re.compile(
    r"(<code>[A-Za-z_][A-Za-z0-9_]*)(?:,[A-Za-z_][A-Za-z0-9_]*)+(</code><br/>)"
)


def strip_tag_options(document: str) -> str:
    """Return generated API documentation with field tag options removed."""
    return FIELD_TAG.sub(r"\1\2", document)


def main() -> int:
    if len(sys.argv) < 2:
        print(f"usage: {Path(sys.argv[0]).name} FILE [FILE ...]", file=sys.stderr)
        return 2

    for filename in sys.argv[1:]:
        path = Path(filename)
        original = path.read_text(encoding="utf-8")
        updated = strip_tag_options(original)
        if updated != original:
            path.write_text(updated, encoding="utf-8")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
