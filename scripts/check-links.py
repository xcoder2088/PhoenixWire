#!/usr/bin/env python3
"""Reports markdown links to files that do not exist (run from anywhere: python3 scripts/check-links.py)."""
import os, re, subprocess, sys
root = subprocess.run(["git", "rev-parse", "--show-toplevel"], capture_output=True, text=True, cwd=os.path.dirname(os.path.abspath(__file__))).stdout.strip()
bad = 0
for path in subprocess.run(["git", "ls-files", "*.md"], capture_output=True, text=True, cwd=root).stdout.split():
    if "node_modules" in path:
        continue
    text = open(os.path.join(root, path), encoding="utf-8").read()
    for target in re.findall(r"\]\(([^)#\s]+)(?:#[^)]*)?\)", text):
        if re.match(r"^[a-z]+:", target) or target.startswith("mailto:"):
            continue
        full = os.path.normpath(os.path.join(root, os.path.dirname(path), target))
        if not os.path.exists(full):
            print(f"{path}: broken link -> {target}")
            bad += 1
sys.exit(1 if bad else 0)
