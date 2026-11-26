"""Tests of check_bindings.py. Run: python3 -m unittest discover -s scripts"""

import unittest

from check_bindings import check, parse_models, parse_types

MODELS = """export namespace domain {

\texport class Other {
\t    ignored: string;

\t    constructor(source: any = {}) {
\t    }
\t}

}

export namespace main {

\texport class ItemIO {
\t    name: string;
\t    note?: string;

\t    static createFrom(source: any = {}) {
\t        return new ItemIO(source);
\t    }
\t}
\texport class CachedIO {
\t    name: string;
\t    at: string;

\t    constructor(source: any = {}) {
\t    }
\t}

}

export namespace workspaceinfo {

\texport class Elsewhere {
\t    x: string;

\t    constructor(source: any = {}) {
\t    }
\t}

}
"""

TYPES = """export interface Item {
  name: string
  // a comment
  note?: string
}
export type Shown = {
  name: string
}
export interface Cached extends Shown {
  at: string
}
"""


def run(models=MODELS, types=TYPES, aliases=None, unchecked=None, ts_only=None):
    return check(parse_models(models), parse_types(types), aliases or {}, unchecked or {}, ts_only or {})


class CheckBindings(unittest.TestCase):
    def test_matching_types_pass_and_only_the_main_namespace_is_read(self):
        self.assertEqual(sorted(parse_models(MODELS)), ["CachedIO", "ItemIO"])
        self.assertEqual(run(), ([], 2))

    def test_a_field_added_in_go_is_reported(self):
        problems, _ = run(models=MODELS.replace("\t    note?: string;\n", "\t    note?: string;\n\t    added: number;\n"))
        self.assertEqual(len(problems), 1)
        self.assertIn("Go sends 'added'", problems[0])

    def test_a_field_removed_or_renamed_in_go_is_reported(self):
        problems, _ = run(models=MODELS.replace("note?: string;", "remark?: string;"))
        self.assertEqual(len(problems), 2, problems)

    def test_inherited_fields_count(self):
        problems, _ = run(types=TYPES.replace("  name: string\n}\nexport interface Cached", "  name: string\n  extra?: string\n}\nexport interface Cached"))
        self.assertEqual(problems, ["CachedIO -> Cached: the TypeScript type has 'extra', which Go does not send"])
        allowed, _ = run(types=TYPES.replace("  at: string\n", "  at: string\n  extra?: string\n"), ts_only={"CachedIO": {"extra"}})
        self.assertEqual(allowed, [])

    def test_a_new_go_type_cannot_slip_through(self):
        problems, _ = run(types=TYPES.replace("export interface Item {", "export interface Renamed {"))
        self.assertTrue(any(p.startswith("ItemIO: no hand-written type found") for p in problems), problems)
        self.assertEqual(run(types=TYPES.replace("export interface Item {", "export interface Renamed {"), aliases={"ItemIO": "Renamed"})[0], [])
        self.assertEqual(run(types=TYPES.replace("export interface Item {", "export interface Renamed {"), unchecked={"ItemIO": "why"})[0], [])

    def test_stale_entries_are_reported(self):
        self.assertIn("remove the entry", run(unchecked={"GoneIO": "why"})[0][0])
        self.assertIn("remove it from UNCHECKED", run(unchecked={"ItemIO": "why"})[0][0])
        self.assertIn("does not exist", run(aliases={"ItemIO": "Nope"})[0][0])
        self.assertIn("no longer only in", run(ts_only={"CachedIO": {"name"}})[0][0])

    def test_reading_nothing_is_a_failure_not_a_pass(self):
        self.assertIn("wails generate module", run(models="")[0][0])
        self.assertIn("nothing could be compared", run(types="")[0][0])


if __name__ == "__main__":
    unittest.main()
