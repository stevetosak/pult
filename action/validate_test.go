package action

import (
	"encoding/json"
	"slices"
	"testing"
)

// An Action that arrives as JSON has no lines: its faults name fields by path.
func TestValidateNamesFieldsByPath(t *testing.T) {
	src := `{"id":"a","title":"A","run":"x","params":[
		{"name":"v","title":"V","type":"string","max":0},
		{"name":"v","title":"W","type":"choice","options":["x","x"]}]}`
	var a Action
	if err := json.Unmarshal([]byte(src), &a); err != nil {
		t.Fatal(err)
	}
	want := []string{
		"params[0].max: max must be from 1 to 1024 for a string, not 0",
		"params[1].options[1]: option \"x\" is already listed at params[1].options[0]",
		"params[1].name: Parameter name \"v\" is already used at params[0].name",
	}
	if got := faultStrings(t, Validate(a)); !slices.Equal(got, want) {
		t.Errorf("faults:\n  got  %q\n  want %q", got, want)
	}
}

func FuzzParse(f *testing.F) {
	f.Add([]byte(exampleFile))
	f.Add([]byte(file("title: A", "run: x")))
	f.Add([]byte("version: 1\nactions:\n  - &a {id: a, title: A, run: x}\n  - *a\n"))
	f.Fuzz(func(t *testing.T, src []byte) {
		file, err := Parse(src)
		if err != nil {
			return
		}
		for _, a := range file.Actions {
			if err := Validate(a); err != nil {
				t.Fatalf("Parse accepted an Action that Validate refuses: %v", err)
			}
			if Pin(a) != Pin(a) {
				t.Fatal("Pin is not stable")
			}
		}
	})
}

func FuzzDecodeValues(f *testing.F) {
	f.Add("eyJhIjoiYiJ9")
	f.Add("e30")
	f.Fuzz(func(t *testing.T, arg string) {
		values, err := DecodeValues(arg)
		if err != nil {
			return
		}
		again, err := encodeValues(values)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := DecodeValues(again); err != nil {
			t.Fatalf("a value set does not decode again: %v", err)
		}
	})
}
