// A focused lexer workload, not a replacement for altbak's end-to-end baselines.
// Usage: node test/regex-bench.mjs /path/to/PureScript/CST/Lexer.purs [baseline-revision]
import { execFileSync } from 'node:child_process';
import { copyFileSync, mkdtempSync, mkdirSync, readFileSync, rmSync, writeFileSync } from 'node:fs';
import { tmpdir } from 'node:os';
import { join } from 'node:path';
import { fileURLToPath } from 'node:url';
const lexer = process.argv[2];
if (!lexer) throw new Error('Pass the language-cst-parser Lexer.purs source path');
const baseline = process.argv[3];
const root=fileURLToPath(new URL('../',import.meta.url));
const current=readFileSync(new URL('../src/Data/String/Regex.go',import.meta.url),'utf8');
const variants=[['qualified',current],['general',current.replace('cst: regexCST(source, canonicalFlags)','cst: nil')]];
if(baseline) variants.unshift(['baseline',execFileSync('git',['show',`${baseline}:src/Data/String/Regex.go`],{cwd:root,encoding:'utf8'})]);
const work=mkdtempSync(join(tmpdir(),'gopurs-regex-bench-'));
try{
 for(const [name,source] of variants){
  const dir=join(work,name);mkdirSync(dir);
  writeFileSync(join(dir,'regex.go'),source);
  for(const [src,dest] of [['regex-bench_test.go','regex_test.go'],['regex-go.mod','go.mod'],['regex-go.sum','go.sum']])copyFileSync(fileURLToPath(new URL(src,import.meta.url)),join(dir,dest));
  copyFileSync(lexer,join(dir,'Lexer.purs'));
  execFileSync('go',['test','-mod=readonly','-c','-o','bench'],{cwd:dir,stdio:'inherit'});
 }
 for(let sample=1;sample<=3;sample++)for(const [name]of variants){
  console.log(`variant=${name} sample=${sample}`);
  execFileSync(join(work,name,'bench'),['-test.run=^$','-test.bench=.','-test.benchmem','-test.benchtime=150ms'],{cwd:join(work,name),stdio:'inherit'});
 }
}finally{rmSync(work,{recursive:true,force:true});}
