package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// The tests that exec a real Run start this test binary as pult.
func TestMain(m *testing.M) {
	if os.Getenv("GO_PULT_AS_MAIN") == "1" {
		main()
	}
	os.Exit(m.Run())
}

const testFile = `version: 1
actions:
  - id: greet
    title: Greet
    group: Demo
    params:
      - {name: who, title: Who, type: string, default: world}
      - {name: times, title: Times, type: int, min: 1, max: 3, default: 1}
    run: printf '%s %s' "$PULT_WHO" "$PULT_TIMES"
  - id: shell
    title: A shell
    kind: terminal
    run: exec "$SHELL" -l
`

// fixture is a home directory with an Action file, and a system whose exec
// records the command instead of starting it.
type fixture struct {
	t              *testing.T
	home, file     string
	env            map[string]string
	stdout, stderr bytes.Buffer
	execs          []command
	sys            *system
}

func newFixture(t *testing.T, actions string) *fixture {
	t.Helper()
	home := t.TempDir()
	dir := filepath.Join(home, ".config", "pult")
	if err := os.MkdirAll(dir, 0o700); err != nil {
		t.Fatal(err)
	}
	fx := &fixture{t: t, home: home, file: filepath.Join(dir, "actions.yaml"), env: map[string]string{"HOME": home}}
	fx.write(actions)
	fx.sys = &system{
		stdout:  &fx.stdout,
		stderr:  &fx.stderr,
		getenv:  func(k string) string { return fx.env[k] },
		environ: func() []string { return []string{"HOME=" + home, "PATH=/usr/bin:/bin", "PULT_OLD=inherited"} },
		uid:     os.Getuid(),
		exec: func(c command) error {
			fx.execs = append(fx.execs, c)
			return nil
		},
		syntax: shellSyntax,
	}
	return fx
}

func (fx *fixture) write(actions string) {
	fx.t.Helper()
	if err := os.WriteFile(fx.file, []byte(actions), 0o600); err != nil {
		fx.t.Fatal(err)
	}
}

func (fx *fixture) pult(args ...string) int {
	fx.stdout.Reset()
	fx.stderr.Reset()
	fx.execs = nil
	return pult(args, fx.sys)
}

func (fx *fixture) wantExit(code, want int) {
	fx.t.Helper()
	if code != want {
		fx.t.Fatalf("exit %d, want %d\nstdout: %s\nstderr: %s", code, want, fx.stdout.String(), fx.stderr.String())
	}
}

func (fx *fixture) wantStderr(parts ...string) {
	fx.t.Helper()
	for _, p := range parts {
		if !strings.Contains(fx.stderr.String(), p) {
			fx.t.Errorf("stderr does not hold %q:\n%s", p, fx.stderr.String())
		}
	}
}

func TestUsage(t *testing.T) {
	fx := newFixture(t, testFile)
	fx.wantExit(fx.pult(), exitUsage)
	fx.wantExit(fx.pult("frobnicate"), exitUsage)
	fx.wantStderr(`unknown command "frobnicate"`)
	fx.wantExit(fx.pult("help"), exitOK)
	if !strings.Contains(fx.stdout.String(), "pult run <id>") {
		t.Errorf("help: %s", fx.stdout.String())
	}
}

func TestActionFileLocation(t *testing.T) {
	fx := newFixture(t, testFile)
	got, err := actionFile(fx.sys)
	if err != nil || got != fx.file {
		t.Errorf("actionFile = %q, %v; want %q", got, err, fx.file)
	}
	fx.env["XDG_CONFIG_HOME"] = "/etc/xdg-test"
	if got, _ := actionFile(fx.sys); got != "/etc/xdg-test/pult/actions.yaml" {
		t.Errorf("with XDG_CONFIG_HOME: %q", got)
	}
	fx.env["XDG_CONFIG_HOME"] = "relative/dir"
	if got, _ := actionFile(fx.sys); got != fx.file {
		t.Errorf("a relative XDG_CONFIG_HOME was used: %q", got)
	}
	fx.env = map[string]string{"HOME": "relative"}
	if _, err := actionFile(fx.sys); err == nil {
		t.Error("a relative HOME was accepted")
	}
}
