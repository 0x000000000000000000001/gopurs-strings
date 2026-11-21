// Exercise the native FFI directly, without rebuilding the PureScript compiler.
import assert from "node:assert/strict";
import { execFileSync } from "node:child_process";
import { copyFileSync, mkdtempSync, readFileSync, rmSync, writeFileSync } from "node:fs";
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
  cases.push({ pattern, flags: new RegExp(pattern, flags).flags, input, want: match ? Array.from(match, x => x ?? null) : null });
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
add(blockPattern, "um", "{-\na-}");
for (const input of ["{-hello-} after", "{- a --} after", "{-a-}\n", "{-𐍈-}", "{-a\r", "{-😀-->\n"])
  add(blockPattern, "u", input);
for (const [pattern, flags, input] of [
  ["^(a+)(b)?", "", "aaa"],
  [String.raw`^(?:\p{Lu}[\p{L}0-9_']*)`, "u", "Éclair_2"],
  [String.raw`^\u{1f600}`, "u", "😀!"],
  ["^hello", "i", "HELLO there"],
]) add(pattern, flags, input);


// Closed list copied from language-cst-parser 0.14.1. Test every admitted fast
// path against JS and the general engine, including real PureScript suffixes.
const lexerPatterns = [
 String.raw`^(?:--[^\r\n]*)`, String.raw`^(?:#![^\r\n]*)`, '^(?: +)', '^(?:\n+)', '^(?:(?:\r\n)+)',
 String.raw`^(?:(?:(?:\p{Lu}[\p{L}0-9_']*)\.)*)`, String.raw`^(?:\p{Lu}[\p{L}0-9_']*)`, String.raw`^(?:[\p{Ll}_][\p{L}0-9_']*)`,
 String.raw`^(?:[a-fA-F0-9]{1,6})`, String.raw`^(?:\\[ \r\n]+\\)`, String.raw`^(?:[^"\\]+)`, String.raw`^(?:""""{0,2}([^"]+"{1,2})*[^"]*""")`,
 String.raw`^(?:(0|[1-9][0-9_]*))`, String.raw`^(?:[0-9_]+)`, String.raw`^(?:[a-fA-F0-9]+)`, symbolPattern, blockPattern,
];
const pursSource = readFileSync(new URL('../src/Data/String/Regex.purs', import.meta.url),'utf8');
const lexerInputs = ['', 'Module.Name.', 'Éclair.Δelta.x', "foo'", '\\ \r\n\\', '"""abc""" tail', '"""a"b""c"""', '0123_4', '0xabc', '😀→+ trailing', '{-nested{-comment-}-}', '\n\n', '\r\n\r\n', '--abc\rnext', '#!abc\nnext'];
for(let i=0;i<pursSource.length;i+=53) lexerInputs.push(pursSource.slice(i));
let seed=173;
for(let i=0;i<200;i++){let value='';for(let n=0;n<24;n++){seed=(Math.imul(seed,1664525)+1013904223)>>>0;value+=alphabet[seed%alphabet.length];}lexerInputs.push(value);}
for(const pattern of lexerPatterns) for(const input of lexerInputs){ add(pattern,'u',input); cases[cases.length-1].fast=true; }

const workspace = mkdtempSync(join(tmpdir(), "gopurs-regex-cst-"));
try {
  copyFileSync(fileURLToPath(new URL("../src/Data/String/Regex.go", import.meta.url)), join(workspace, "regex.go"));
  copyFileSync(fileURLToPath(new URL("./regex-cst_test.go", import.meta.url)), join(workspace, "regex_test.go"));
  writeFileSync(join(workspace, "fixtures.json"), JSON.stringify(cases));
  for (const name of ["mod", "sum"]) copyFileSync(fileURLToPath(new URL(`./regex-go.${name}`, import.meta.url)), join(workspace, `go.${name}`));
  execFileSync("go", ["test", "-mod=readonly", "regex.go", "regex_test.go", "-count=1"], { cwd: workspace, stdio: "inherit" });
  console.log(`JS/Go regex parity: ${cases.length} cases; symbol reduction checked for every Unicode scalar.`);
} finally {
  rmSync(workspace, { recursive: true, force: true });
}
