package action

import (
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"
	"testing"
)

// shellSyntax are values that a shell would run as code if they became
// command text.
var shellSyntax = []string{
	`$(touch pwned)`,
	"`touch pwned`",
	`; touch pwned`,
	`&& touch pwned`,
	`| touch pwned`,
	`' ; touch pwned ; '`,
	`" ; touch pwned ; "`,
	`${IFS}touch${IFS}pwned`,
	`> pwned`,
	`*`,
	`~`,
	`\`,
	`$PATH`,
	`-rf /`,
	`Ω ünïcödé`,
}

func valuesAction() Action {
	return Action{
		ID: "v", Title: "V", Run: `printf '%s' "$PULT_TEXT"`,
		Params: []Param{
			{Name: "text", Title: "Text", Type: TypeString},
			{Name: "hours", Title: "Hours", Type: TypeInt, Min: ptr[int64](1), Max: ptr[int64](12), Default: ptr("2")},
			{Name: "lid", Title: "Lid", Type: TypeBool, Default: ptr("false")},
			{Name: "vault", Title: "Vault", Type: TypeChoice, Options: []string{"personal", "work"}},
			{Name: "code", Title: "Code", Type: TypeString, Max: ptr[int64](4), Pattern: `[0-9]+`, Default: ptr("1")},
		},
	}
}

func TestBindFillsDefaultsInOrder(t *testing.T) {
	got, err := valuesAction().Bind(map[string]string{"vault": "work", "text": "hi"})
	if err != nil {
		t.Fatal(err)
	}
	want := []Binding{{"text", "hi"}, {"hours", "2"}, {"lid", "false"}, {"vault", "work"}, {"code", "1"}}
	if !slices.Equal(got, want) {
		t.Errorf("Bind = %v, want %v", got, want)
	}
}

func TestBindFaults(t *testing.T) {
	tests := []struct {
		name  string
		given map[string]string
		want  []string
	}{
		{"missing", map[string]string{"text": "a"}, []string{"vault: a value is required"}},
		{"unknown", map[string]string{"text": "a", "vault": "work", "zz": "1", "PATH": "/tmp"},
			[]string{"PATH: the Action v has no Parameter with this name", "zz: the Action v has no Parameter with this name"}},
		{"int above", map[string]string{"text": "a", "vault": "work", "hours": "13"}, []string{"hours: the value is above the maximum 12"}},
		{"int below", map[string]string{"text": "a", "vault": "work", "hours": "0"}, []string{"hours: the value is below the minimum 1"}},
		{"int form", map[string]string{"text": "a", "vault": "work", "hours": "+3"}, []string{"hours: the value is not a whole number in decimal"}},
		{"int space", map[string]string{"text": "a", "vault": "work", "hours": "3 "}, []string{"hours: the value is not a whole number in decimal"}},
		{"bool", map[string]string{"text": "a", "vault": "work", "lid": "yes"}, []string{"lid: the value is not true or false"}},
		{"choice", map[string]string{"text": "a", "vault": "Work"}, []string{"vault: the value is not one of the options"}},
		{"newline", map[string]string{"text": "a\nb", "vault": "work"}, []string{"text: the value holds a control character"}},
		{"ESC", map[string]string{"text": "a\x1b[2J", "vault": "work"}, []string{"text: the value holds a control character"}},
		{"NUL", map[string]string{"text": "a\x00", "vault": "work"}, []string{"text: the value holds a control character"}},
		{"bidi", map[string]string{"text": "a‮", "vault": "work"}, []string{"text: the value holds a control character"}},
		{"not UTF-8", map[string]string{"text": "a\xff", "vault": "work"}, []string{"text: the value is not valid UTF-8"}},
		{"long", map[string]string{"text": strings.Repeat("ü", 201), "vault": "work"}, []string{"text: the value is longer than 200 characters"}},
		{"pattern", map[string]string{"text": "a", "vault": "work", "code": "12a"}, []string{`code: the value does not match the pattern "[0-9]+"`}},
		{"pattern is whole", map[string]string{"text": "a", "vault": "work", "code": "a1"}, []string{`code: the value does not match the pattern "[0-9]+"`}},
		{"several", map[string]string{"hours": "x", "lid": "1"},
			[]string{"text: a value is required", "hours: the value is not a whole number in decimal",
				"lid: the value is not true or false", "vault: a value is required"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := valuesAction().Bind(tt.given)
			if got := faultStrings(t, err); !slices.Equal(got, tt.want) {
				t.Errorf("faults:\n  got  %q\n  want %q", got, tt.want)
			}
		})
	}
}

func TestBindKeepsShellSyntaxAsText(t *testing.T) {
	for _, v := range shellSyntax {
		got, err := valuesAction().Bind(map[string]string{"text": v, "vault": "work"})
		if err != nil {
			t.Errorf("%q: %v", v, err)
			continue
		}
		if got[0].Value != v {
			t.Errorf("Bind changed %q to %q", v, got[0].Value)
		}
	}
}

func TestEnviron(t *testing.T) {
	base := []string{"HOME=/home/x", "PULT_OLD=1", "PULT_TEXT=inherited", "PATH=/bin"}
	got := Environ(base, []Binding{{"text", "hi"}, {"hours", "2"}})
	want := []string{"HOME=/home/x", "PATH=/bin", "PULT_TEXT=hi", "PULT_HOURS=2"}
	if !slices.Equal(got, want) {
		t.Errorf("Environ = %q, want %q", got, want)
	}
	if EnvName("max_hours") != "PULT_MAX_HOURS" {
		t.Errorf("EnvName(max_hours) = %s", EnvName("max_hours"))
	}
}

// A value with shell syntax reaches the command as data: sh -c prints it
// unchanged, and nothing in it runs.
func TestShellSyntaxValuesStayData(t *testing.T) {
	sh, err := exec.LookPath("sh")
	if err != nil {
		t.Skip("no sh")
	}
	a := valuesAction()
	for _, v := range shellSyntax {
		dir := t.TempDir()
		bindings, err := a.Bind(map[string]string{"text": v, "vault": "work"})
		if err != nil {
			t.Fatal(err)
		}
		cmd := exec.Command(sh, "-c", a.Run)
		cmd.Dir = dir
		cmd.Env = Environ([]string{"PATH=" + os.Getenv("PATH")}, bindings)
		out, err := cmd.Output()
		if err != nil {
			t.Fatalf("%q: %v", v, err)
		}
		if string(out) != v {
			t.Errorf("sh printed %q, want %q", out, v)
		}
		if _, err := os.Stat(filepath.Join(dir, "pwned")); err == nil {
			t.Errorf("%q ran as code", v)
		}
	}
}
