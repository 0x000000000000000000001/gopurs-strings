# Native regular expressions

The Go port uses `github.com/dlclark/regexp2` **v1.11.5**, with `ECMAScript`
mode and `Unicode` mode when the PureScript `u` flag is set. That version's
[go.mod](https://github.com/dlclark/regexp2/blob/v1.11.5/go.mod) requires Go 1.13;
the port retains gopurs' Go 1.22 minimum. The v2 module requires a newer Go
version and is not selected implicitly.

`go-dependencies.json` pins the module. Before generating an application, prepare
its output module, including the requirements of its local ports:

```sh
node ../gopurs/tools/library-go-dependencies.mjs output ../gopurs-strings
```

Run this from the application directory, adjusting sibling paths as needed.
Dependency preparation must precede generation because the native FFI parser
resolves imported Go types. `bin/test` and the compiler's isolated library test
harness perform this preparation. The test fixtures also pin their own `go.mod`
and `go.sum`; `go mod tidy` alone is not a version selection policy.

## Behavior covered by the differential tests

The Go interface retains ordinary typed functions and an opaque `*GoRegex`.
It does not expose an `Any` registry or require a change to the TAST.

- Positive and negative lookahead/lookbehind, numeric backreferences and named
  backreferences with JavaScript capture ordering.
- Empty captures distinguished from absent captures, including callbacks and
  `$1`, `$nn`, `$<name>`, `$$`, `$&`, prefix and suffix replacements.
- `g`, `i`, `m`, `s`, `u` and `y`, duplicate/unknown flag rejection, stateless
  repeated `test` calls, global empty matches and sticky matching from index 0.
- UTF-16 search indices; UTF-16 matching without `u`, scalar matching with `u`,
  and canonical WTF-8 preservation of isolated surrogates.
- JavaScript line terminators for dot and multiline anchors, source/flag
  display, and split boundaries, empty matches and participating captures.

Errors from compilation return through `regexImpl`'s existing `Left` callback.
An unsupported expression is never silently replaced by `^$`. For example,
`a(?=b)` now matches `ab`; before this change its stored Go expression was `^$`
and the same input incorrectly returned false.

## Deliberate limits

This is a tested compatibility subset, not a complete ECMAScript implementation.

- Captures nested inside a repeated group are rejected. regexp2 retains captures
  from earlier repetitions where JavaScript clears them; `(a|(b))+` on `ba`
  exposed that difference. Rejecting the pattern prevents stale captures and
  incorrect backreferences. Noncapturing grouping without nested captures and
  the qualified CST expressions remain supported.
- Capture names currently use ASCII letters, digits, `_` and `$`, with a
  nondigit first character. Duplicate names, .NET-only group constructs and
  flags outside `gimsuy` are rejected.
- Full Unicode case folding is not established. In particular, regexp2 v1.11.5
  does not match long s (`ſ`) with `[a-z]` under `iu`, unlike JavaScript. Unicode
  property membership also follows the Go toolchain's tables and may differ
  from the JavaScript engine's Unicode version. Applications relying on these
  cases need a dedicated compatibility check.
- `split` raises an explicit exception if a separator has an absent capture:
  JavaScript inserts `undefined`, which cannot inhabit the declared PureScript
  result `Array String`. Participating captures, including empty ones, work.
- Native operations start at index 0 and expose no mutable `lastIndex`. They do
  not reproduce state left on a JavaScript RegExp by successive sticky `match`
  or `replace` calls. The separately documented `test` operation stays stateless.
- regexp2 is a backtracking engine. Each match has an approximate one-second
  timeout; runtime failures raise an exception instead of returning false,
  `Nothing`, or partial output. It is not a strict whole-operation deadline:
  a global operation can perform multiple matches. See the upstream
  [timeout contract](https://github.com/dlclark/regexp2/blob/v1.11.5/README.md#catastrophic-backtracking-and-timeouts).

## Lexer paths and measurements

Using regexp2 for every lexer attempt converted the entire remaining source to
runes at every token. A closed list of the 17 expressions from
`language-cst-parser` 0.14.1 therefore keeps Go's linear regex engine for `match`,
only when the flags are exactly `u`. Successful RE2 compilation is **not** used
as a general dispatch rule. General expressions use regexp2.

These exact expressions use syntax with a direct translation. The symbol
lookahead can be removed because Unicode punctuation and symbol categories are
disjoint. The block-comment reduction tries the closing delimiter before
repeating and retains both capture groups. The tests compare every admitted
path with JavaScript and the general engine, including generated comments,
Unicode tokens and actual PureScript source suffixes. Changed flags and changed
patterns do not enter the qualified path.

Measured on 2026-11-21, Apple M4 Pro, Go 1.27.0. Medians of three interleaved
150 ms benchmark samples; input is a representative token followed by the
actual `PureScript/CST/Lexer.purs` source. Baseline commit:
`a6cec255b458e43c4b075c43a66fd6190af21318`.

| Lexer operation | Before, ns/op | Qualified path, ns/op | General engine only, ns/op | Before / qualified B/op |
| --- | ---: | ---: | ---: | ---: |
| Identifier | 342.9 | 344.8 | 34909 | 72 / 72 |
| Proper name | 387.2 | 344.2 | 33672 | 72 / 72 |
| Symbol | 182.4 | 192.7 | 33609 | 72 / 72 |
| Block comment | 875.9 | 847.4 | 34039 | 232 / 168 |
| Spaces | 182.2 | 182.9 | 32437 | 72 / 72 |

The qualified paths avoid the large suffix conversion regression and remain in
the original timing range. These microbenchmarks do not establish a compiler
or application speedup. The official
[altbak.pub baselines](../../../altbak.pub/README.md) were consulted: the strings
compilation row records approximately 12.680 s for the JavaScript compiler,
4.489 s for the native Go compiler, and 3.824 s for its dev profile. Those are
whole compilation timings with different workloads; they cannot be compared
numerically with the matching timings above.

## Reproduce

```sh
node test/regex-native.mjs
GOTOOLCHAIN=go1.22.12 node test/regex-native.mjs
node test/regex-cst.mjs
node test/regex-bench.mjs \
  ../gopurs/.spago/p/language-cst-parser-0.14.1/src/PureScript/CST/Lexer.purs \
  a6cec255b458e43c4b075c43a66fd6190af21318
```

The first command executes the actual JavaScript FFI as its oracle, including
884 operation cases, explicit rejection cases, timeout propagation, concurrent
use and Go's race detector. It passed with Go 1.27.0 and Go 1.22.12.
UTF-16 wire fixtures preserve isolated surrogates.
The second checks 13,494 JS/Go/general-engine cases; the symbol reduction is
also checked in JavaScript for every Unicode scalar. The benchmark uses private
temporary directories and compares the baseline, qualified paths and general
engine. Supply the lexer path installed in your workspace if it differs.

`test/Test/Data/String/Regex.purs` adds end-to-end assertions for lookarounds,
backreferences, captures, Unicode indices, flags, replacement and split. The
complete strings suite passed through the typed frontend and gopurs release
and dev generation in an isolated source family, including Go `-N -l` for dev.
