// Compare the actual JS and Go FFI without compiling PureScript.
import { execFileSync } from "node:child_process";
import { copyFileSync, mkdirSync, mkdtempSync, readFileSync, rmSync, writeFileSync } from "node:fs";
import { tmpdir } from "node:os";
import { join } from "node:path";
import { fileURLToPath } from "node:url";
import * as units from "../src/Data/String/CodeUnits.js";
import * as unsafe from "../src/Data/String/Unsafe.js";

function hex(value) {
  const bytes = [];
  for (const char of value) {
    const cp = char.codePointAt(0);
    if (cp >= 0xd800 && cp <= 0xdfff) bytes.push(0xed, 0x80 | ((cp >> 6) & 63), 0x80 | (cp & 63));
    else bytes.push(...Buffer.from(char));
  }
  return Buffer.from(bytes).toString("hex");
}
const maybeHex = value => value === null ? null : hex(value);
const just = x => x;
const inputs = ["", "a", "ab", "x", "ÿ", "≠", "·", "aÿ≠·z", "aéx漢", "💻", "a💻z", "💻𝄞", "\ud800", "\udfff", "a\ud800z", "\ud800\ud800\udc00\udfff", "\0", "\uffff", "éé", "𝄞a𝄞"];
const cases = inputs.map(input => {
  const indices = Array.from({ length: input.length + 5 }, (_, i) => i - 2);
  const slices = [];
  for (const start of indices) for (const end of indices) slices.push({ start, end, want: hex(units.slice(start)(end)(input)) });
  const searches = [];
  for (const pattern of ["", "a", "x", "ÿ", "≠", "·", "💻", "\ud83d", "\udcbb", "\ud800", "zz"])
    searches.push({ pattern: hex(pattern), first: units._indexOf(just)(null)(pattern)(input), last: units._lastIndexOf(just)(null)(pattern)(input), starts: indices.map(index => ({ index, first: units._indexOfStartingAt(just)(null)(pattern)(index)(input), last: units._lastIndexOfStartingAt(just)(null)(pattern)(index)(input) })) });
  let unsafeChar;
  try { unsafeChar = hex(unsafe.char(input)); } catch { unsafeChar = null; }
  return { input: hex(input), length: units.length(input), char: maybeHex(units._toChar(just)(null)(input)), unsafeChar,
    chars: units.toCharArray(input).map(hex), joined: hex(units.fromCharArray(units.toCharArray(input))), countPrefix: units.countPrefix(c => c !== "x")(input),
    indices: indices.map(index => { const split = units.splitAt(index)(input); let unsafeAt; try { unsafeAt = hex(unsafe.charAt(index)(input)); } catch { unsafeAt = null; } return { index, char: maybeHex(units._charAt(just)(null)(index)(input)), unsafeAt, take: hex(units.take(index)(input)), drop: hex(units.drop(index)(input)), before: hex(split.before), after: hex(split.after) }; }), slices, searches };
});
const workspace = mkdtempSync(join(tmpdir(), "gopurs-code-units-"));
try {
  writeFileSync(join(workspace, "CodeUnits.go"), "package codeunits\n" + readFileSync(new URL("../src/Data/String/CodeUnits.go", import.meta.url), "utf8"));
  writeFileSync(join(workspace, "Unsafe.go"), "package codeunits\nimport \"gopurs/output/gopurs_runtime\"\n" + readFileSync(new URL("../src/Data/String/Unsafe.go", import.meta.url), "utf8"));
  copyFileSync(fileURLToPath(new URL("./code-units-native_test.go", import.meta.url)), join(workspace, "code_units_test.go"));
  const runtime = join(workspace, "gopurs_runtime");
  mkdirSync(runtime);
  copyFileSync(fileURLToPath(new URL("../../gopurs/runtime/runtime.go", import.meta.url)), join(runtime, "runtime.go"));
  writeFileSync(join(workspace, "go.mod"), "module gopurs/output\n\ngo 1.22\n");
  writeFileSync(join(workspace, "fixtures.json"), JSON.stringify(cases));
  execFileSync("go", ["test", "-count=1", "."], { cwd: workspace, stdio: "inherit" });
  console.log(`JS/Go UTF-16 parity: ${cases.length} inputs, including b8x TAST characters, supplementary characters and isolated surrogates.`);
} finally {
  rmSync(workspace, { recursive: true, force: true });
}
