package action

import (
	"errors"
	"slices"
	"testing"
)

func TestCheck(t *testing.T) {
	src := `version: 1
actions:
  - id: a
    title: A
    params:
      - {name: v, title: V, type: string}
      - {name: w, title: W, type: string}
      - {name: unused, title: U, type: bool}
    run: echo "$PULT_V" "${PULT_W}x" "$PULT_TYPO" "$PULT_TYPO"
  - id: b
    title: B
    run: |
      broken
`
	f, err := Parse([]byte(src))
	if err != nil {
		t.Fatal(err)
	}
	faults, warnings := f.Check(func(a Action) error {
		if a.ID == "b" {
			return errors.New("b: 1: Syntax error")
		}
		return nil
	})
	wantFaults := []string{
		"9:10: run uses $PULT_TYPO, but no Parameter of a declares it",
		"12:10: run has a syntax fault: b: 1: Syntax error",
	}
	if got := faultStrings(t, faults.err()); !slices.Equal(got, wantFaults) {
		t.Errorf("faults:\n  got  %q\n  want %q", got, wantFaults)
	}
	wantWarnings := []string{"8:16: run never names $PULT_UNUSED, the variable of Parameter unused"}
	if got := faultStrings(t, warnings.err()); !slices.Equal(got, wantWarnings) {
		t.Errorf("warnings:\n  got  %q\n  want %q", got, wantWarnings)
	}
}

// A File that was not parsed has no positions; its findings name paths.
func TestCheckWithoutPositions(t *testing.T) {
	f := &File{Version: 1, Actions: []Action{{ID: "a", Title: "A", Run: "echo $PULT_X"}}}
	faults, _ := f.Check(nil)
	want := []string{"actions[0].run: run uses $PULT_X, but no Parameter of a declares it"}
	if got := faultStrings(t, faults.err()); !slices.Equal(got, want) {
		t.Errorf("faults = %q, want %q", got, want)
	}
}
