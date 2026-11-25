# Clear Signal 0.13.1 (draft; delete after publishing)

A fix release for `clrsig conform` filters.

## Read this if you use a path with a parameter in `--paths` or `--max-response-time-for`

**A run that passed can now fail**, because something that was silently left out is now done:

- **`--paths`** matched operations and negative probes against two different spellings. With `--paths '/items/{id}'`, as the spec writes it, **no operation** was selected while the probes ran. With `--paths '/items/{{id}}'`, the operations ran and **no probe** did, and the probe summary said `PASSED  0 probes`.
- **`--max-response-time-for`** matched only the `{{id}}` spelling. A limit written as the spec writes it, such as `/items/{id}=500`, **did nothing**, without a message. It now applies.

Both now take either spelling, and so do `conform --list` and the Check Conformance page. A command never selects less than before. Prefixes without a parameter (`/items/`) behave as they did. If one path is given both spellings with two different time limits, the `{{id}}` one wins, as it did when it was the only one that applied.

These problems are in 0.13.0 and 0.12.0; older versions were not checked.

**Full Changelog**: https://github.com/fuvito/api-workbench/compare/v0.13.0...v0.13.1
