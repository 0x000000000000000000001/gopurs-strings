package Regex

import (
	"encoding/json"
	"os"
	"reflect"
	"testing"
)

func TestCSTRegexMatchesJavaScript(t *testing.T) {
	data, err := os.ReadFile("fixtures.json")
	if err != nil {
		t.Fatal(err)
	}
	var fixtures []struct {
		Pattern string
		Flags   string
		Input   string
		Want    any
		Fast    bool
	}
	if err := json.Unmarshal(data, &fixtures); err != nil {
		t.Fatal(err)
	}
	for _, fixture := range fixtures {
		var compileError string
		result := RegexImpl(func(message string) any {
			compileError = message
			return nil
		}, func(regex any) any { return regex }, fixture.Pattern, fixture.Flags)
		if compileError != "" {
			t.Fatalf("compile %q: %s", fixture.Pattern, compileError)
		}
		regex := result.(*GoRegex)
		if regex.Source != fixture.Pattern || regex.Flags != fixture.Flags {
			t.Fatal("regex source or flags changed")
		}
		if fixture.Fast && regex.cst == nil {
			t.Fatalf("missing qualified fast path: %q", fixture.Pattern)
		}
		got := _Match(func(value any) any { return value }, nil, regex, fixture.Input)
		if regex.cst != nil {
			regex.cst = nil
			general := _Match(func(value any) any { return value }, nil, regex, fixture.Input)
			if !reflect.DeepEqual(general, fixture.Want) {
				t.Fatalf("general pattern %q input %q: got %#v want %#v", fixture.Pattern, fixture.Input, general, fixture.Want)
			}
		}
		if !reflect.DeepEqual(got, fixture.Want) {
			t.Fatalf("pattern %q input %q: got %#v, want %#v", fixture.Pattern, fixture.Input, got, fixture.Want)
		}
	}
}
