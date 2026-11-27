package action

import (
	"encoding/json"
	"strings"
	"testing"
)

// The Core decodes what pult list --json prints, and computes the same pins.
func TestListingRoundTrip(t *testing.T) {
	f, err := Parse([]byte(exampleFile))
	if err != nil {
		t.Fatal(err)
	}
	data, err := json.Marshal(NewListing(f))
	if err != nil {
		t.Fatal(err)
	}
	l, err := DecodeListing(data)
	if err != nil {
		t.Fatalf("DecodeListing: %v", err)
	}
	if len(l.Actions) != len(f.Actions) {
		t.Fatalf("%d Actions, want %d", len(l.Actions), len(f.Actions))
	}
	for i := range f.Actions {
		if Pin(l.Actions[i]) != Pin(f.Actions[i]) {
			t.Errorf("%s: the pin changed in the listing", f.Actions[i].ID)
		}
	}
}

func TestNewListingOfAnEmptyFile(t *testing.T) {
	f, err := Parse([]byte("version: 1\n"))
	if err != nil {
		t.Fatal(err)
	}
	data, _ := json.Marshal(NewListing(f))
	if string(data) != `{"version":1,"actions":[]}` {
		t.Errorf("listing = %s", data)
	}
	if _, err := DecodeListing(data); err != nil {
		t.Errorf("DecodeListing: %v", err)
	}
}

func TestDecodeListingRefuses(t *testing.T) {
	ok := `{"id":"a","title":"A","run":"x"}`
	tests := map[string]string{
		"not JSON":      `version: 1`,
		"two values":    `{"version":1,"actions":[]} {}`,
		"version 2":     `{"version":2,"actions":[]}`,
		"no version":    `{"actions":[]}`,
		"unknown field": `{"version":1,"actions":[{"id":"a","title":"A","run":"x","status":"y"}]}`,
		"top field":     `{"version":1,"actions":[],"pins":[]}`,
		"invalid":       `{"version":1,"actions":[{"id":"A","title":"A","run":"x"}]}`,
		"id twice":      `{"version":1,"actions":[` + ok + `,` + ok + `]}`,
		"too large":     `{"version":1,"actions":[],"x":"` + strings.Repeat("a", maxListing) + `"}`,
	}
	for name, data := range tests {
		if _, err := DecodeListing([]byte(data)); err == nil {
			t.Errorf("%s: accepted", name)
		}
	}
}
