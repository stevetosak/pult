package main

import (
	"testing"

	"github.com/stevetosak/pult/action"
)

func TestListTable(t *testing.T) {
	fx := newFixture(t, testFile)
	fx.wantExit(fx.pult("list"), exitOK)
	want := "ID     KIND      GROUP  TITLE\n" +
		"greet  output    Demo   Greet\n" +
		"shell  terminal         A shell\n"
	if fx.stdout.String() != want {
		t.Errorf("list:\n%s\nwant:\n%s", fx.stdout.String(), want)
	}
}

// The Core decodes the listing and computes the pins that pult run checks.
func TestListJSONIsWhatTheCoreDecodes(t *testing.T) {
	fx := newFixture(t, testFile)
	fx.wantExit(fx.pult("list", "--json"), exitOK)
	l, err := action.DecodeListing(fx.stdout.Bytes())
	if err != nil {
		t.Fatalf("DecodeListing: %v\n%s", err, fx.stdout.String())
	}
	f, err := action.Parse([]byte(testFile))
	if err != nil {
		t.Fatal(err)
	}
	for i, a := range l.Actions {
		if action.Pin(a) != action.Pin(f.Actions[i]) {
			t.Errorf("%s: the listing gives another pin", a.ID)
		}
	}
}

func TestListRefusesABrokenFile(t *testing.T) {
	fx := newFixture(t, "version: 2\n")
	fx.wantExit(fx.pult("list", "--json"), exitFaults)
	fx.wantStderr(fx.file+":1:10: version 2 is not supported", "pult check shows every fault")
	if fx.stdout.Len() != 0 {
		t.Errorf("stdout = %q, want nothing", fx.stdout.String())
	}
	fx.wantExit(fx.pult("list", "extra"), exitUsage)
}
