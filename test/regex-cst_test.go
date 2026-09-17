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
		got := _Match(func(value any) any { return value }, nil, regex, fixture.Input)
		if !reflect.DeepEqual(got, fixture.Want) {
			t.Fatalf("pattern %q input %q: got %#v, want %#v", fixture.Pattern, fixture.Input, got, fixture.Want)
		}
	}
}

func TestBlockCommentMultilineKeepsExistingUnsupportedBehavior(t *testing.T) {
	// The CST-specific reduction is invalid with multiline $. Keep this
	// unsupported variant out of it, rather than silently returning "{-".
	regex := RegexImpl(func(message string) any {
		t.Fatal(message)
		return nil
	}, func(value any) any { return value }, `^(?:\{-(-(?!\})|[^-]+)*(-\}|$))`, "um").(*GoRegex)
	if got := _Match(func(value any) any { return value }, nil, regex, "{-\na-}"); got != nil {
		t.Fatalf("multiline expression incorrectly used CST reduction: %#v", got)
	}
}
