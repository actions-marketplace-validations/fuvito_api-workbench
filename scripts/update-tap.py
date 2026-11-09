#!/usr/bin/env python3
"""
Update Formula/clrsig.rb and bucket/clrsig.json with the correct SHA256 hashes
for a given release tag. Run from the repo root after a release is published:

    python3 scripts/update-tap.py v0.1.0
"""
import hashlib
import json
import re
import sys
import urllib.request

TAG = sys.argv[1]
VERSION = TAG.lstrip("v")
BASE = f"https://github.com/fuvito/api-workbench/releases/download/{TAG}"

BINARIES = {
    "darwin-arm64":  f"{BASE}/clrsig-darwin-arm64",
    "darwin-amd64":  f"{BASE}/clrsig-darwin-amd64",
    "linux-amd64":   f"{BASE}/clrsig-linux-amd64",
    "windows-amd64": f"{BASE}/clrsig-windows-amd64.exe",
}


def sha256_url(url: str) -> str:
    print(f"  fetching {url} ...", flush=True)
    with urllib.request.urlopen(url) as r:
        return hashlib.sha256(r.read()).hexdigest()


print(f"Computing SHA256s for {TAG}:")
hashes = {k: sha256_url(v) for k, v in BINARIES.items()}
for k, v in hashes.items():
    print(f"  {k}: {v}")

# ---- Homebrew formula ----
with open("Formula/clrsig.rb") as f:
    formula = f.read()

formula = re.sub(r'version "[^"]+"', f'version "{VERSION}"', formula)
formula = re.sub(
    r'(clrsig-darwin-arm64"\n\s+sha256 ")[0-9a-f]{64}',
    rf'\g<1>{hashes["darwin-arm64"]}', formula
)
formula = re.sub(
    r'(clrsig-darwin-amd64"\n\s+sha256 ")[0-9a-f]{64}',
    rf'\g<1>{hashes["darwin-amd64"]}', formula
)
formula = re.sub(
    r'(clrsig-linux-amd64"\n\s+sha256 ")[0-9a-f]{64}',
    rf'\g<1>{hashes["linux-amd64"]}', formula
)

with open("Formula/clrsig.rb", "w") as f:
    f.write(formula)
print("Updated Formula/clrsig.rb")

# ---- Scoop manifest ----
with open("bucket/clrsig.json") as f:
    manifest = json.load(f)

manifest["version"] = VERSION
manifest["architecture"]["64bit"]["url"] = BINARIES["windows-amd64"]
manifest["architecture"]["64bit"]["hash"] = hashes["windows-amd64"]

with open("bucket/clrsig.json", "w") as f:
    json.dump(manifest, f, indent=4)
    f.write("\n")
print("Updated bucket/clrsig.json")
