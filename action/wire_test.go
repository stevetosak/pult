package action

import (
	"encoding/base64"
	"maps"
	"os/exec"
	"regexp"
	"slices"
	"strings"
	"testing"
)

var safeCharacters = regexp.MustCompile(`^[A-Za-z0-9_-]+$`)

// Every argument of the command the Core sends holds only A-Z a-z 0-9 _ -,
// whatever the values hold (ADR 0002, amendment).
func TestRunArgsHoldOnlySafeCharacters(t *testing.T) {
	a := valuesAction()
	for _, v := range append(shellSyntax, strings.Repeat("€", 200)) {
		given := map[string]string{"text": v, "vault": "work", "hours": "12"}
		args, err := RunArgs(a, given)
		if err != nil {
			t.Fatalf("%q: %v", v, err)
		}
		for _, arg := range args {
			if !safeCharacters.MatchString(arg) {
				t.Errorf("%q gives the argument %q", v, arg)
			}
		}
		if want := []string{"run", "v", "--expect", Pin(a), "--params"}; !slices.Equal(args[:5], want) {
			t.Errorf("args = %q, want them to start with %q", args, want)
		}
		back, err := DecodeValues(args[5])
		if err != nil {
			t.Fatalf("DecodeValues: %v", err)
		}
		if !maps.Equal(back, given) {
			t.Errorf("values came back as %q, want %q", back, given)
		}
	}
}

func TestRunArgsWithoutValues(t *testing.T) {
	a := goldenActions()[0]
	args, err := RunArgs(a, nil)
	if err != nil {
		t.Fatal(err)
	}
	if want := []string{"run", "lock-screen", "--expect", Pin(a)}; !slices.Equal(args, want) {
		t.Errorf("args = %q, want %q", args, want)
	}
}

func TestRunArgsRefuses(t *testing.T) {
	if _, err := RunArgs(Action{ID: "Bad id", Title: "T", Run: "x"}, nil); err == nil {
		t.Error("an invalid Action was accepted")
	}
	if _, err := RunArgs(valuesAction(), map[string]string{"text": "a"}); err == nil {
		t.Error("a missing value was accepted")
	}
}

// The command string passes through each login shell on the Machine with its
// arguments unchanged. A shell function named pult prints what it receives.
func TestRunArgsThroughLoginShells(t *testing.T) {
	args, err := RunArgs(valuesAction(), map[string]string{"text": "$(touch pwned) `id` ; '\"", "vault": "work"})
	if err != nil {
		t.Fatal(err)
	}
	command := "pult " + strings.Join(args, " ")
	ran := 0
	for _, shell := range []string{"sh", "dash", "bash", "zsh"} {
		path, err := exec.LookPath(shell)
		if err != nil {
			continue
		}
		ran++
		script := `pult() { for a in "$@"; do printf '%s\n' "$a"; done; }; ` + command
		cmd := exec.Command(path, "-c", script)
		cmd.Dir = t.TempDir()
		out, err := cmd.Output()
		if err != nil {
			t.Fatalf("%s: %v", shell, err)
		}
		if got := strings.Split(strings.TrimSuffix(string(out), "\n"), "\n"); !slices.Equal(got, args) {
			t.Errorf("%s passed %q, want %q", shell, got, args)
		}
	}
	if ran == 0 {
		t.Skip("no shell found")
	}
}

func TestDecodeValuesRefuses(t *testing.T) {
	enc := base64.RawURLEncoding.EncodeToString
	tests := map[string]string{
		"empty":        "",
		"padding":      enc([]byte(`{"a":"b"}`)) + "=",
		"plus":         "eyJhIjoiYiJ9+",
		"slash":        "eyJhIjoiYiJ9/",
		"space":        "eyJhIjoi YiJ9",
		"newline":      "eyJhIjoi\nYiJ9",
		"not base64":   "a",
		"not JSON":     enc([]byte(`a=b`)),
		"JSON array":   enc([]byte(`["a"]`)),
		"JSON null":    enc([]byte(`null`)),
		"number value": enc([]byte(`{"a":1}`)),
		"trailing":     enc([]byte(`{"a":"b"} x`)),
		"too long":     strings.Repeat("A", maxEncodedValues+1),
	}
	for name, arg := range tests {
		if _, err := DecodeValues(arg); err == nil {
			t.Errorf("%s: DecodeValues(%.40q) was accepted", name, arg)
		}
	}
}
