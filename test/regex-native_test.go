package Regex

import (
	"encoding/json"
	"fmt"
	"os"
	"reflect"
	"strings"
	"sync"
	"testing"
	"time"
)

func fixtureString(units []uint16) string {
	rs := make([]rune, len(units))
	for i, c := range units {
		rs[i] = rune(c)
	}
	return regexString(rs)
}
func fixtureWire(v any) any {
	switch x := v.(type) {
	case string:
		rs := regexRunes(x, false)
		u := make([]any, len(rs))
		for i, c := range rs {
			u[i] = float64(c)
		}
		return map[string]any{"utf16": u}
	case []string:
		out := make([]any, len(x))
		for i, v := range x {
			out[i] = fixtureWire(v)
		}
		return out
	case []any:
		out := make([]any, len(x))
		for i, v := range x {
			out[i] = fixtureWire(v)
		}
		return out
	case map[string]any:
		out := make(map[string]any)
		for k, v := range x {
			out[k] = fixtureWire(v)
		}
		return out
	case int:
		return float64(x)
	default:
		return v
	}
}
func fixtureJSON(v any) string {
	// encoding/json escapes U+2028 and U+2029 unlike JSON.stringify, and replaces
	// WTF-8 surrogates. Construct the JS string spelling for callback results.
	switch x := v.(type) {
	case nil:
		return "null"
	case string:
		var b strings.Builder
		b.WriteByte('"')
		for _, c := range regexRunes(x, true) {
			switch c {
			case '"', '\\':
				b.WriteByte('\\')
				b.WriteRune(c)
			case '\b':
				b.WriteString(`\b`)
			case '\f':
				b.WriteString(`\f`)
			case '\n':
				b.WriteString(`\n`)
			case '\r':
				b.WriteString(`\r`)
			case '\t':
				b.WriteString(`\t`)
			default:
				if c < 32 || c >= 0xd800 && c <= 0xdfff {
					fmt.Fprintf(&b, `\u%04x`, c)
				} else {
					b.WriteRune(c)
				}
			}
		}
		b.WriteByte('"')
		return b.String()
	case []any:
		parts := make([]string, len(x))
		for i, v := range x {
			parts[i] = fixtureJSON(v)
		}
		return "[" + strings.Join(parts, ",") + "]"
	default:
		panic("unexpected callback value")
	}
}
func compileFixture(t *testing.T, p, f string) *GoRegex {
	t.Helper()
	return RegexImpl(func(e string) any { t.Fatalf("compile %q/%s: %s", p, f, e); return nil }, func(v any) any { return v }, p, f).(*GoRegex)
}
func TestJavaScriptOperations(t *testing.T) {
	data, e := os.ReadFile("fixtures.json")
	if e != nil {
		t.Fatal(e)
	}
	var fixtures []struct {
		Pattern, Input, Replacement []uint16
		Flags, Op                   string
		Want                        any
		RejectSplit                 bool
	}
	if e = json.Unmarshal(data, &fixtures); e != nil {
		t.Fatal(e)
	}
	for n, c := range fixtures {
		t.Run(fmt.Sprint(n), func(t *testing.T) {
			p, s, repl := fixtureString(c.Pattern), fixtureString(c.Input), fixtureString(c.Replacement)
			r := compileFixture(t, p, c.Flags)
			var got any
			if c.RejectSplit {
				defer func() {
					if e := recover(); e == nil || !strings.Contains(fmt.Sprint(e), "unmatched captures") {
						t.Fatalf("expected explicit split limitation, got %v", e)
					}
				}()
				Split(r, s)
				return
			}
			switch c.Op {
			case "test":
				got = []any{Test(r, s), Test(r, s)}
			case "match":
				got = _Match(func(v any) any { return v }, nil, r, s)
			case "search":
				got = _Search(func(v any) any { return v }, nil, r, s)
			case "replace":
				got = Replace(r, repl, s)
			case "replaceBy":
				got = _ReplaceBy(func(v any) any { return v }, nil, r, func(m string) func([]any) string { return func(gs []any) string { return fixtureJSON([]any{m, gs}) } }, s)
			case "split":
				got = Split(r, s)
			case "source":
				got = []any{Source(r), ShowRegexImpl(r), FlagsImpl(r)}
			}
			if wire := fixtureWire(got); !reflect.DeepEqual(wire, c.Want) {
				t.Errorf("%s %q/%s input %q replacement %q\ngot  %#v\nwant %#v", c.Op, p, c.Flags, s, repl, wire, c.Want)
			}
		})
	}
}
func TestRejectUnsupportedSyntax(t *testing.T) {
	data, _ := os.ReadFile("invalid.json")
	var cases [][2]string
	json.Unmarshal(data, &cases)
	for _, c := range cases {
		rejected := false
		RegexImpl(func(string) any { rejected = true; return nil }, func(any) any { return nil }, c[0], c[1])
		if !rejected {
			t.Errorf("accepted %q/%s", c[0], c[1])
		}
	}
}
func TestTimeoutRaisesError(t *testing.T) {
	r := compileFixture(t, `^(a+)+$`, "")
	r.Re.MatchTimeout = time.Millisecond
	defer func() {
		e := recover()
		if e == nil || !strings.Contains(fmt.Sprint(e), "timeout") {
			t.Fatalf("expected timeout exception, got %v", e)
		}
	}()
	Test(r, strings.Repeat("a", 100)+"!")
}
func TestConcurrentRegex(t *testing.T) {
	r := compileFixture(t, `(?<a>x)(y)\k<a>`, "g")
	var wg sync.WaitGroup
	for n := 0; n < 8; n++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for i := 0; i < 100; i++ {
				if !Test(r, "xyx") {
					t.Error("concurrent match failed")
					return
				}
			}
		}()
	}
	wg.Wait()
}
func TestUTF16RoundTrip(t *testing.T) {
	s := "A😀B"
	if got := regexString(regexRunes(s, false)); got != s {
		t.Fatal(got)
	}
}
