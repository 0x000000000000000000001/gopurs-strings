// Compare the actual JS FFI and native FFI. UTF-16 wire values preserve lone
// surrogates that Go's encoding/json would otherwise replace with U+FFFD.
import { execFileSync } from 'node:child_process';
import { copyFileSync, mkdtempSync, readFileSync, rmSync, writeFileSync } from 'node:fs';
import { tmpdir } from 'node:os';
import { join } from 'node:path';
import { fileURLToPath } from 'node:url';
const ffi = await import('data:text/javascript;base64,' + Buffer.from(readFileSync(new URL('../src/Data/String/Regex.js', import.meta.url))).toString('base64'));
const units = s => Array.from({length:s.length}, (_, i) => s.charCodeAt(i));
const wire = x => typeof x === 'string' ? {utf16:units(x)} : Array.isArray(x) ? x.map(wire) : x && typeof x === 'object' ? Object.fromEntries(Object.entries(x).map(([k,v]) => [k,wire(v)])) : x ?? null;
const cases = [];
const groups = [
  ['a(?=b)', '', ['ab','ac','']], ['a(?!b)', '', ['ac','ab']],
  ['(?<=a)b','',['ab','cb']], ['(?<!a)b','',['ab','cb']],
  ['(a)\\1','',['aa','ab']], ['(a)?b\\1','',['b','aba']],
  ['(?<first>x)(y)\\k<first>','',['xyx','xyz']],
  ['(a*)()(?<tail>b)?','',['aa','aab','']],
  ['.', '', ['😀','é','\r','\n','\u2028','\u2029','\ud800']],
  ['.', 'u', ['😀','é','\r','\ud800']], ['.', 's', ['\n','\r','😀']],
  ['.', 'su', ['\n','😀']], ['.', 'g', ['😀','ab','']], ['.', 'gu', ['😀','ab','']],
  ['a$', '', ['a','a\n','a\r','a\r\n']], ['^a$', 'm', ['x\na\ny','x\ra\ry','a\u2028']],
  ['a', 'yi', ['A','bA','aa']], ['a', 'gy', ['aa a','baa']],
  ['a', 'gi', ['Aa a','b']], ['(?:)', 'g', ['ab','😀','']], ['(?:)', 'gu', ['😀','']],
  ['b', 'u', ['😀b','éab']], ['b', '', ['😀b','éab']],
  ['\\u{1F600}','u',['😀','x']], ['😀','',['😀','x']], ['😀+','u',['😀😀']],
  ['\\p{Lu}+','u',['ÉA','é']], ['\\d+','',['١2','123']],
  ['\\s','',['\ufeff','\u00a0','\u0085']], ['\\w','',['é','A','K']],
  ['(a)|(b)','',['a','b']], ['\\A','',['A','']], ['\\z','',['z','']], ['\\uD83D\\uDE00','u',['😀']], ['[a-z]+','i',['aZ','123']], ['a/b','',['a/b']],
];
function add(pattern, flags, input, op, replacement='') {
  const r = new RegExp(pattern, flags);
  let want;
  switch(op) {
    case 'test': want = [ffi.test(r)(input), ffi.test(r)(input)]; break;
    case 'match': want = ffi._match(x=>x)(null)(r)(input); break;
    case 'search': want = ffi._search(x=>x)(null)(r)(input); break;
    case 'replace': want = ffi.replace(r)(replacement)(input); break;
    case 'replaceBy': want = ffi._replaceBy(x=>x)(null)(r)(m=>gs=>JSON.stringify([m,gs]))(input); break;
    case 'split': want = ffi.split(r)(input); break;
    case 'source': want = [ffi.source(r),ffi.showRegexImpl(r),ffi.flagsImpl(r)]; break;
  }
  // Array String cannot represent the undefined inserted by JS split for an
  // absent capture. The native FFI deliberately raises an explicit exception.
  const rejectSplit = op === 'split' && want.some(x=>x===undefined);
  cases.push({pattern:units(pattern),flags,input:units(input),op,replacement:units(replacement),want:wire(want),rejectSplit});
}
for(const [p,f,inputs] of groups) for(const input of inputs) {
  for(const op of ['test','match','search','source','replaceBy','split']) add(p,f,input,op);
  for(const replacement of ['<$&>','$$-$`-$\'','$1/$2/$12/$01/$0','$<first>:$<tail>:$<absent>']) add(p,f,input,'replace',replacement);
}
for(const [p,f] of [['',''],['a/b','mig'],['\n',''],['\\/','']]) add(p,f,'','source');
const invalid = [['+',''],['(a|(b))+',''],['(?:a(b)?)+',''],['\\A','u'],['a','gg'],['a','q'],['(?>a)',''],['(?i:a)',''],['(?(1)a|b)',''],['(?<x>a)(?<x>b)',''],['\\k<missing>','u']];
const workspace = mkdtempSync(join(tmpdir(),'gopurs-regex-native-'));
try {
  for(const [source,dest] of [['../src/Data/String/Regex.go','regex.go'],['./regex-native_test.go','regex_test.go'],['./regex-go.mod','go.mod'],['./regex-go.sum','go.sum']]) copyFileSync(fileURLToPath(new URL(source,import.meta.url)),join(workspace,dest));
  writeFileSync(join(workspace,'fixtures.json'),JSON.stringify(cases));
  writeFileSync(join(workspace,'invalid.json'),JSON.stringify(invalid));
  execFileSync('go',['test','-mod=readonly','-count=1','-race','-timeout=30s','./...'],{cwd:workspace,stdio:'inherit'});
  console.log(`JS/native regex: ${cases.length} operation cases, ${invalid.length} rejected syntax cases; timeout and concurrent use checked.`);
} finally { rmSync(workspace,{recursive:true,force:true}); }
