package codeunits

import (
 "encoding/hex"
 "encoding/json"
 "os"
 "reflect"
 "testing"
)

type indexCase struct { Index int; Char, UnsafeAt *string; Take, Drop, Before, After string }
type searchCase struct { Pattern string; First, Last *int; Starts []struct { Index int; First, Last *int } }
type unitCase struct { Input string; Length int; Char, UnsafeChar *string; Chars []string; Joined string; CountPrefix int; Indices []indexCase; Slices []struct { Start, End int; Want string }; Searches []searchCase }
func decodeHex(t *testing.T, s string) string { t.Helper(); b,e:=hex.DecodeString(s);if e!=nil {t.Fatal(e)};return string(b) }
func maybeStringHex(v any) *string { if v==nil{return nil}; s:=hex.EncodeToString([]byte(v.(string)));return &s }
func callUnsafe(f func() any) (result *string) {defer func(){if recover()!=nil {result=nil}}();return maybeStringHex(f())}
func check(t *testing.T, name string, got, want any) { t.Helper();if !reflect.DeepEqual(got,want){t.Fatalf("%s: got %#v want %#v",name,got,want)} }
func TestCodeUnitsAgainstJavaScript(t *testing.T) {
 data,e:=os.ReadFile("fixtures.json");if e!=nil{t.Fatal(e)};var cases []unitCase;if e=json.Unmarshal(data,&cases);e!=nil {t.Fatal(e)}
 for _,c:=range cases {t.Run(c.Input,func(t *testing.T){
  input:=decodeHex(t,c.Input);justS:=func(s string)any{return s};justI:=func(i int)any{return i}
  check(t,"length",Length(input),c.Length)
  check(t,"toChar",maybeStringHex(_ToChar(justS,nil,input)),c.Char)
  check(t,"unsafe char",callUnsafe(func()any{return Char(input)}),c.UnsafeChar)
  chars:=ToCharArray(input);gotChars:=make([]string,len(chars));for i,s:=range chars {gotChars[i]=hex.EncodeToString([]byte(s))};check(t,"toCharArray",gotChars,c.Chars)
  check(t,"fromCharArray",hex.EncodeToString([]byte(FromCharArray(chars))),c.Joined)
  check(t,"countPrefix",CountPrefix(func(s string)bool{return s!="x"},input),c.CountPrefix)
  // CoreFn.Json.decodeLiteral checks UTF-16 length before taking the first Char.
  var literal any;if Length(input)==1 {literal=chars[0]};check(t,"TAST CharLiteral",maybeStringHex(literal),c.Char)
  for _,i:=range c.Indices {
   check(t,"charAt",maybeStringHex(_CharAt(justS,nil,i.Index,input)),i.Char)
   check(t,"unsafe charAt",callUnsafe(func()any{return CharAt(i.Index).(func(any)any)(input)}),i.UnsafeAt)
   check(t,"take",hex.EncodeToString([]byte(Take(i.Index,input))),i.Take)
   check(t,"drop",hex.EncodeToString([]byte(Drop(i.Index,input))),i.Drop)
   split:=SplitAt(i.Index,input);check(t,"splitAt.before",hex.EncodeToString([]byte(split["before"].(string))),i.Before);check(t,"splitAt.after",hex.EncodeToString([]byte(split["after"].(string))),i.After)
  }
  for _,s:=range c.Slices {check(t,"slice",hex.EncodeToString([]byte(Slice(s.Start,s.End,input))),s.Want)}
  checkIndex:=func(label string,got any,want *int){var actual *int;if got!=nil {v:=got.(int);actual=&v};check(t,label,actual,want)}
  for _,s:=range c.Searches {p:=decodeHex(t,s.Pattern);checkIndex("indexOf",_IndexOf(justI,nil,p,input),s.First);checkIndex("lastIndexOf",_LastIndexOf(justI,nil,p,input),s.Last)
   for _,i:=range s.Starts {checkIndex("indexOfStartingAt",_IndexOfStartingAt(justI,nil,p,i.Index,input),i.First);checkIndex("lastIndexOfStartingAt",_LastIndexOfStartingAt(justI,nil,p,i.Index,input),i.Last)}
  }
 })}
}
