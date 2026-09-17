// Exercise the native FFI directly, without rebuilding the PureScript compiler.
import assert from "node:assert/strict";
import { execFileSync } from "node:child_process";
import { copyFileSync, mkdtempSync, rmSync, writeFileSync } from "node:fs";
import { tmpdir } from "node:os";
import { join } from "node:path";
import { fileURLToPath } from "node:url";

const symbolPattern = String.raw`^(?:(?:[:!#$%&*+./<=>?@\\^|~-]|(?!\p{P})\p{S})+)`;
const simplified = symbolPattern.replace(String.raw`(?!\p{P})\p{S}`, String.raw`\p{S}`);
const originalRegex = new RegExp(symbolPattern, "u");
const simplifiedRegex = new RegExp(simplified, "u");

// Unicode categories P and S are disjoint. Check every scalar in the JS engine
// as well as representative complete tokens in both JS and the actual Go FFI.
for (let scalar = 0; scalar <= 0x10ffff; scalar++) {
  if (scalar >= 0xd800 && scalar <= 0xdfff) continue;
  const input = String.fromCodePoint(scalar);
  assert.equal(originalRegex.test(input), simplifiedRegex.test(input));
}

const cases = [];
function add(pattern, flags, input) {
  const match = new RegExp(pattern, flags).exec(input);
  cases.push({ pattern, flags, input, want: match ? Array.from(match, x => x ?? null) : null });
}
const alphabet = ["=", "+", "-", ":", "\\", "^", "~", "|", "a", "_", "{", "}", ",", " ", "\n", "→", "∑", "€", "©", "😀", "。", "́"];
for (const first of ["", ...alphabet]) {
  add(symbolPattern, "u", first);
  for (const second of alphabet) add(symbolPattern, "u", first + second + " trailing");
}
for (const input of ["=1", "->", "<=", "::", "+++", "😀→= rest", "arity=1", "-- comment"])
  add(symbolPattern, "u", input);
const blockPattern = String.raw`^(?:\{-(-(?!\})|[^-]+)*(-\}|$))`;
function addComments(input, remaining) {
  add(blockPattern, "u", input);
  add(blockPattern, "u", "{-" + input);
  if (remaining > 0)
    for (const char of ["-", "}", "{", "a", "\n"]) addComments(input + char, remaining - 1);
}
addComments("", 5);
for (const input of ["{-hello-} after", "{- a --} after", "{-a-}\n", "{-𐍈-}", "{-a\r", "{-😀-->\n"])
  add(blockPattern, "u", input);
for (const [pattern, flags, input] of [
  ["^(a+)(b)?", "", "aaa"],
  [String.raw`^(?:\p{Lu}[\p{L}0-9_']*)`, "u", "Éclair_2"],
  [String.raw`^\u{1f600}`, "u", "😀!"],
  ["^hello", "i", "HELLO there"],
]) add(pattern, flags, input);

const workspace = mkdtempSync(join(tmpdir(), "gopurs-regex-cst-"));
try {
  copyFileSync(fileURLToPath(new URL("../src/Data/String/Regex.go", import.meta.url)), join(workspace, "regex.go"));
  copyFileSync(fileURLToPath(new URL("./regex-cst_test.go", import.meta.url)), join(workspace, "regex_test.go"));
  writeFileSync(join(workspace, "fixtures.json"), JSON.stringify(cases));
  execFileSync("go", ["test", "regex.go", "regex_test.go", "-count=1"], { cwd: workspace, stdio: "inherit" });
  console.log(`JS/Go regex parity: ${cases.length} cases; symbol reduction checked for every Unicode scalar.`);
} finally {
  rmSync(workspace, { recursive: true, force: true });
}
