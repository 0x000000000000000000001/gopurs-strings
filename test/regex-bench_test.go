package Regex

import (
 "os"
 "testing"
)
var regexBenchSink any
func BenchmarkCSTSuffixes(b *testing.B) {
 data,err:=os.ReadFile("Lexer.purs");if err!=nil{b.Fatal(err)}
 text:=string(data)
 for _,c:=range []struct{name,prefix,pattern string}{
  {"identifier","lexerIdentifier ",`^(?:[\p{Ll}_][\p{L}0-9_']*)`},
  {"proper","LexerIdentifier ",`^(?:\p{Lu}[\p{L}0-9_']*)`},
  {"symbols","→+= ",`^(?:(?:[:!#$%&*+./<=>?@\\^|~-]|(?!\p{P})\p{S})+)`},
  {"comment","{- a comment -} ",`^(?:\{-(-(?!\})|[^-]+)*(-\}|$))`},
  {"space","      ",`^(?: +)`},
 }{
  b.Run(c.name,func(b *testing.B){r:=RegexImpl(func(e string)any{b.Fatal(e);return nil},func(v any)any{return v},c.pattern,"u").(*GoRegex);input:=c.prefix+text;b.ReportAllocs();b.ResetTimer();for n:=0;n<b.N;n++{regexBenchSink=_Match(func(v any)any{return v},nil,r,input)}})
 }
}
