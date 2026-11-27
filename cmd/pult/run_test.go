package main

import (
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/stevetosak/pult/action"
)

func parsedAction(t *testing.T, src, id string) action.Action {
	t.Helper()
	f, err := action.Parse([]byte(src))
	if err != nil {
		t.Fatal(err)
	}
	a, ok := f.Action(id)
	if !ok {
		t.Fatalf("no Action %s", id)
	}
	return a
}

func TestRunStartsTheCommand(t *testing.T) {
	fx := newFixture(t, testFile)
	fx.wantExit(fx.pult("run", "greet", "--param", "who=$(id)"), exitOK)
	if len(fx.execs) != 1 {
		t.Fatalf("%d execs, want 1", len(fx.execs))
	}
	c := fx.execs[0]
	a := parsedAction(t, testFile, "greet")
	if c.path != "/bin/sh" || !slices.Equal(c.argv, []string{"sh", "-c", a.Run, "greet"}) {
		t.Errorf("exec %s %q", c.path, c.argv)
	}
	if c.dir != fx.home || !c.noInput {
		t.Errorf("dir %q, noInput %v; want the home and no input", c.dir, c.noInput)
	}
	want := []string{"HOME=" + fx.home, "PATH=/usr/bin:/bin", "PULT_WHO=$(id)", "PULT_TIMES=1"}
	if !slices.Equal(c.env, want) {
		t.Errorf("env = %q, want %q", c.env, want)
	}
}

func TestRunTerminalKeepsTheInput(t *testing.T) {
	fx := newFixture(t, testFile)
	fx.wantExit(fx.pult("run", "shell"), exitOK)
	if fx.execs[0].noInput {
		t.Error("a terminal Run got /dev/null as input")
	}
}

// The arguments the Core builds start the Run with the Operator's values.
func TestRunTakesTheCoreArguments(t *testing.T) {
	fx := newFixture(t, testFile)
	a := parsedAction(t, testFile, "greet")
	args, err := action.RunArgs(a, map[string]string{"who": `"; rm -rf ~ #`, "times": "3"})
	if err != nil {
		t.Fatal(err)
	}
	fx.wantExit(fx.pult(args...), exitOK)
	env := fx.execs[0].env
	if !slices.Contains(env, `PULT_WHO="; rm -rf ~ #`) || !slices.Contains(env, "PULT_TIMES=3") {
		t.Errorf("env = %q", env)
	}
}

func TestRunRefusesBeforeAnythingStarts(t *testing.T) {
	a := parsedAction(t, testFile, "greet")
	encoded, err := action.RunArgs(a, map[string]string{"who": "x"})
	if err != nil {
		t.Fatal(err)
	}
	params := encoded[len(encoded)-1]
	tests := []struct {
		name string
		args []string
		want string
	}{
		{"no id", []string{"run"}, "the Action id comes first"},
		{"flag first", []string{"run", "--expect", "x", "greet"}, "the Action id comes first"},
		{"unknown id", []string{"run", "nope"}, `has no Action "nope"`},
		{"wrong pin", []string{"run", "greet", "--expect", "v1-0000"}, "the Action greet changed since it was pinned"},
		{"value above", []string{"run", "greet", "--param", "times=4"}, "times: the value is above the maximum 3"},
		{"undeclared", []string{"run", "greet", "--param", "PATH=/tmp"}, "PATH: the Action greet has no Parameter with this name"},
		{"twice", []string{"run", "greet", "--param", "who=a", "--params", params}, "a value for who is given twice"},
		{"no equals", []string{"run", "greet", "--param", "who"}, "--param takes name=value"},
		{"bad params", []string{"run", "greet", "--params", "e30="}, "--params holds a character outside"},
		{"extra", []string{"run", "greet", "extra"}, `unexpected argument "extra"`},
		{"unknown flag", []string{"run", "greet", "--force"}, "flag provided but not defined: -force"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fx := newFixture(t, testFile)
			fx.wantExit(fx.pult(tt.args...), exitRefuse)
			fx.wantStderr("pult: refused: ", tt.want)
			if len(fx.execs) > 0 {
				t.Errorf("the command started: %+v", fx.execs[0])
			}
		})
	}
}

func TestRunRefusesABrokenFile(t *testing.T) {
	fx := newFixture(t, testFile+"  - id: Bad\n    title: B\n    run: x\n")
	fx.wantExit(fx.pult("run", "greet"), exitRefuse)
	fx.wantStderr(`id "Bad" must match`, "pult cannot use the Action file")
	if len(fx.execs) > 0 {
		t.Error("a Run started from a file with faults")
	}
}

func TestRunReportsAFailedExec(t *testing.T) {
	fx := newFixture(t, testFile)
	fx.sys.exec = func(command) error { return errors.New("no such file") }
	fx.wantExit(fx.pult("run", "greet"), exitExec)
	fx.wantStderr("pult: cannot start /bin/sh: no such file")
}

const execFile = `version: 1
actions:
  - id: probe
    title: Probe
    params:
      - {name: v, title: V, type: string}
    run: printf 'v=%s old=%s input=%s dir=%s zero=%s' "$PULT_V" "$PULT_OLD" "$(cat)" "$(pwd -P)" "$0"
  - id: echo-input
    title: Echo the input
    kind: terminal
    run: printf 'input=%s' "$(cat)"
  - id: fail
    title: Fail
    run: exit 7
  - id: mark
    title: Mark
    run: touch "$HOME/ran"
`

// runReal starts this test binary as pult, so that run replaces a real
// process with sh.
func runReal(t *testing.T, home, stdin string, args ...string) (string, int) {
	t.Helper()
	cmd := exec.Command(os.Args[0], args...)
	cmd.Env = []string{"GO_PULT_AS_MAIN=1", "HOME=" + home, "PATH=" + os.Getenv("PATH"), "PULT_OLD=inherited"}
	cmd.Stdin = strings.NewReader(stdin)
	cmd.Dir = t.TempDir()
	out, err := cmd.CombinedOutput()
	var exitErr *exec.ExitError
	switch {
	case errors.As(err, &exitErr):
		return string(out), exitErr.ExitCode()
	case err != nil:
		t.Fatal(err)
	}
	return string(out), 0
}

func TestRunReplacesItselfWithTheCommand(t *testing.T) {
	fx := newFixture(t, execFile)
	home, err := filepath.EvalSymlinks(fx.home)
	if err != nil {
		t.Fatal(err)
	}

	out, code := runReal(t, fx.home, "typed", "run", "probe", "--param", "v=$(touch pwned)")
	want := "v=$(touch pwned) old= input= dir=" + home + " zero=probe"
	if code != 0 || out != want {
		t.Errorf("probe: exit %d, output %q\nwant exit 0, output %q", code, out, want)
	}
	if _, err := os.Stat(filepath.Join(fx.home, "pwned")); err == nil {
		t.Error("the value ran as code")
	}

	if out, code := runReal(t, fx.home, "typed", "run", "echo-input"); code != 0 || out != "input=typed" {
		t.Errorf("terminal Run: exit %d, output %q; want the input passed through", code, out)
	}

	if _, code := runReal(t, fx.home, "", "run", "fail"); code != 7 {
		t.Errorf("fail: exit %d, want the command's own 7", code)
	}
}

func TestRunWithAWrongPinStartsNothing(t *testing.T) {
	fx := newFixture(t, execFile)
	marker := filepath.Join(fx.home, "ran")

	out, code := runReal(t, fx.home, "", "run", "mark", "--expect", "v1-0000")
	if code != exitRefuse {
		t.Errorf("exit %d, want %d: %s", code, exitRefuse, out)
	}
	if _, err := os.Stat(marker); err == nil {
		t.Fatal("the command ran with a wrong pin")
	}

	pin := action.Pin(parsedAction(t, execFile, "mark"))
	if out, code := runReal(t, fx.home, "", "run", "mark", "--expect", pin); code != 0 {
		t.Fatalf("exit %d with the right pin: %s", code, out)
	}
	if _, err := os.Stat(marker); err != nil {
		t.Error("the command did not run with the right pin")
	}
}
