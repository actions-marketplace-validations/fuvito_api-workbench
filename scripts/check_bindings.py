#!/usr/bin/env python3
"""Fail when the TypeScript types the desktop app uses no longer match the Go structs it receives.

Wails generates frontend/wailsjs/go/models.ts from the bound Go methods (the file is not committed:
run `wails generate module` in apps/desktop/wails first). The app does not use those classes: its
types are written by hand, mostly in frontend/src/lib/bridge.ts, and the results are cast to them,
so the compiler cannot see a field that was added, removed or renamed in Go. This compares the
field names of each Go type with its hand-written type.

Every generated type of the `main` package must be compared or be listed in UNCHECKED with the
reason, so a new one cannot slip through. Only names are compared, not types or optionality, and
fields of types written inline inside another type are not looked into.
"""

import argparse
import re
import sys
from pathlib import Path

FRONTEND = Path(__file__).resolve().parent.parent / "apps" / "desktop" / "frontend"

# Go type -> hand-written type, where the name is not the same (with or without the IO suffix).
ALIASES = {
    "AuthIO": "AuthPayload",
    "CoverageIO": "ScenarioCoverage",
    "EnvironmentInfoIO": "EnvInfo",
    "QualityIO": "QualityReport",
    "RequestInput": "RequestPayload",
    "ResponseCacheIO": "ResponseCacheData",
    "ResponseOutput": "ResponseData",
    "TimingIO": "TimingData",
    "VariableIO": "ScenarioVariable",
}

# Go types with no hand-written type of their own to compare with, and why.
_INLINE = "written inline inside the types that contain it"
UNCHECKED = {
    "BodyIO": _INLINE,
    "CliExplanationIO": _INLINE,
    "CliPartIO": _INLINE,
    "DashEnvVarIO": _INLINE,
    "DashLastRunIO": _INLINE,
    "HeaderIO": _INLINE,
    "KVPairIO": _INLINE,
    "OperationResultIO": _INLINE,
    "QueryParamIO": _INLINE,
}

# Optional fields a TypeScript type has that this Go type never sends, on purpose.
TS_ONLY = {
    # A cached response is shown as a response, but the request as sent is not stored in the cache.
    "ResponseCacheIO": {"error", "requestBody", "requestHeaders", "requestMethod", "requestUrl"},
}


def parse_models(text: str) -> dict:
    """name -> set of fields, for the classes of namespace main."""
    start = re.search(r"^export namespace main \{$", text, re.M)
    if not start:
        return {}
    end = re.search(r"^export namespace \w+ \{$", text[start.end():], re.M)
    body = text[start.end(): start.end() + end.start() if end else len(text)]
    classes = {}
    for m in re.finditer(r"^\texport class (\w+) \{\n(.*?)^\t    (?:static |constructor)", body, re.M | re.S):
        classes[m.group(1)] = set(re.findall(r"^\t    (\w+)\??:", m.group(2), re.M))
    return classes


def parse_types(text: str) -> dict:
    """name -> set of fields, for exported interfaces and object types, with what they extend."""
    text = text.replace("\r\n", "\n")
    raw = {}
    pattern = r"^export (?:interface (\w+)(?: extends ([\w, ]+?))? \{|type (\w+) = \{)\n(.*?)^\}"
    for m in re.finditer(pattern, text, re.M | re.S):
        name = m.group(1) or m.group(3)
        parents = [p.strip() for p in (m.group(2) or "").split(",") if p.strip()]
        raw[name] = (set(re.findall(r"^  (\w+)\??:", m.group(4), re.M)), parents)

    def fields(name, seen=()):
        own, parents = raw[name]
        out = set(own)
        for parent in parents:
            if parent in raw and parent not in seen:
                out |= fields(parent, (*seen, name))
        return out

    return {name: fields(name) for name in raw}


def counterpart(go_name: str, types: dict, aliases: dict):
    if go_name in aliases:
        return aliases[go_name]
    for candidate in (go_name, go_name[:-2] if go_name.endswith("IO") else None):
        if candidate and candidate in types:
            return candidate
    return None


def check(classes: dict, types: dict, aliases: dict, unchecked: dict, ts_only: dict) -> tuple:
    """Returns (problems, number of types compared)."""
    if not classes:
        return ["no generated types were found: run `wails generate module` in apps/desktop/wails first"], 0
    if not types:
        return ["no hand-written types were found, so nothing could be compared"], 0
    problems, compared = [], 0
    for name in sorted(set(aliases) | set(unchecked) | set(ts_only)):
        if name not in classes:
            problems.append(f"{name} is listed in this script but Go no longer generates it: remove the entry")
    for name in sorted(classes):
        if name in unchecked:
            if counterpart(name, types, aliases):
                problems.append(f"{name} is listed as unchecked but a type for it exists: remove it from UNCHECKED")
            continue
        other = counterpart(name, types, aliases)
        if other is None:
            problems.append(
                f"{name}: no hand-written type found. Add one named like it, add the pair to ALIASES, "
                "or list it in UNCHECKED with the reason"
            )
            continue
        if other not in types:
            problems.append(f"{name}: ALIASES points at {other}, which does not exist")
            continue
        compared += 1
        for field in sorted(classes[name] - types[other]):
            problems.append(f"{name} -> {other}: Go sends '{field}', which the TypeScript type does not have")
        allowed = ts_only.get(name, set())
        for field in sorted(allowed - (types[other] - classes[name])):
            problems.append(f"{name}: TS_ONLY allows '{field}', which is no longer only in {other}: remove it")
        for field in sorted(types[other] - classes[name] - allowed):
            problems.append(f"{name} -> {other}: the TypeScript type has '{field}', which Go does not send")
    return problems, compared


def main(argv=None) -> int:
    parser = argparse.ArgumentParser(description=__doc__.splitlines()[0])
    parser.add_argument("--models", type=Path, default=FRONTEND / "wailsjs" / "go" / "models.ts")
    parser.add_argument(
        "--types", type=Path, nargs="+",
        default=[FRONTEND / "src" / "lib" / "bridge.ts", FRONTEND / "src" / "lib" / "dynamicFunctions.ts"],
    )
    args = parser.parse_args(argv)
    if not args.models.is_file():
        print(f"{args.models} does not exist: run `wails generate module` in apps/desktop/wails first", file=sys.stderr)
        return 1
    types = {}
    for path in args.types:
        types.update(parse_types(path.read_text(encoding="utf-8")))
    classes = parse_models(args.models.read_text(encoding="utf-8"))
    problems, compared = check(classes, types, ALIASES, UNCHECKED, TS_ONLY)
    for problem in problems:
        print(problem)
    print(f"{compared} of {len(classes)} Go types compared, {len(UNCHECKED)} listed as unchecked, {len(problems)} problem(s)")
    return 1 if problems else 0


if __name__ == "__main__":
    sys.exit(main())
