package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestCheckAcceptsAGoodFile(t *testing.T) {
	fx := newFixture(t, testFile)
	fx.wantExit(fx.pult("check"), exitOK)
	if want := fx.file + ": 2 Actions, no faults, 0 warnings\n"; fx.stdout.String() != want {
		t.Errorf("stdout = %q, want %q", fx.stdout.String(), want)
	}
}

func TestCheckReportsFaultsWithTheirPosition(t *testing.T) {
	fx := newFixture(t, `version: 1
actions:
  - id: a
    title: A
    params:
      - {name: unused, title: U, type: bool}
    run: |
      if true; then
        echo "$PULT_TYPO"
`)
	fx.wantExit(fx.pult("check"), exitFaults)
	fx.wantStderr(
		fx.file+`:7:10: run has a syntax fault: a: 3: Syntax error: end of file unexpected (expecting "fi")`,
		fx.file+":7:10: run uses $PULT_TYPO, but no Parameter of a declares it",
		fx.file+":6:16: warning: run never names $PULT_UNUSED, the variable of Parameter unused",
	)
}

func TestCheckReportsFormatFaults(t *testing.T) {
	fx := newFixture(t, "version: 1\nactions:\n  - id: a\n    run: x\n    bogus: 1\n")
	fx.wantExit(fx.pult("check"), exitFaults)
	fx.wantStderr(fx.file + `:5:5: unknown field "bogus"`)
}

func TestCheckWarningsAlonePass(t *testing.T) {
	fx := newFixture(t, "version: 1\nactions:\n  - id: a\n    title: A\n    params:\n      - {name: v, title: V, type: bool}\n    run: ./script\n")
	fx.wantExit(fx.pult("check"), exitOK)
	fx.wantStderr("warning: run never names $PULT_V")
	if !strings.Contains(fx.stdout.String(), "1 Actions, no faults, 1 warnings") {
		t.Errorf("stdout = %q", fx.stdout.String())
	}
}

func TestCheckTakesAFile(t *testing.T) {
	fx := newFixture(t, testFile)
	private := filepath.Join(t.TempDir(), "private")
	if err := os.Mkdir(private, 0o700); err != nil {
		t.Fatal(err)
	}
	other := filepath.Join(private, "draft.yaml")
	if err := os.WriteFile(other, []byte("version: 1\nactions: [{id: x}]\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	fx.wantExit(fx.pult("check", "--file", other), exitFaults)
	fx.wantStderr(other+":2:11: title is required", other+":2:11: run is required")
	fx.wantExit(fx.pult("check", "extra"), exitUsage)
	fx.wantExit(fx.pult("check", "--bogus"), exitUsage)
}

func TestCheckRefusesAFileOthersCanWrite(t *testing.T) {
	fx := newFixture(t, testFile)
	if err := os.Chmod(fx.file, 0o620); err != nil {
		t.Fatal(err)
	}
	fx.wantExit(fx.pult("check"), exitFaults)
	fx.wantStderr(fx.file+": group or others can write it (mode 0620)", "chmod go-w "+fx.file)

	if err := os.Chmod(fx.file, 0o600); err != nil {
		t.Fatal(err)
	}
	dir := filepath.Dir(fx.file)
	if err := os.Chmod(dir, 0o702); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chmod(dir, 0o700) })
	fx.wantExit(fx.pult("check"), exitFaults)
	fx.wantStderr(dir + ": group or others can write it (mode 0702)")
}

func TestCheckFollowsASymbolicLink(t *testing.T) {
	fx := newFixture(t, testFile)
	dotfiles := filepath.Join(t.TempDir(), "dotfiles")
	if err := os.Mkdir(dotfiles, 0o700); err != nil {
		t.Fatal(err)
	}
	target := filepath.Join(dotfiles, "actions.yaml")
	if err := os.Rename(fx.file, target); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(target, fx.file); err != nil {
		t.Fatal(err)
	}
	fx.wantExit(fx.pult("check"), exitOK)

	if err := os.Chmod(dotfiles, 0o770); err != nil {
		t.Fatal(err)
	}
	fx.wantExit(fx.pult("check"), exitFaults)
	fx.wantStderr(dotfiles + ": group or others can write it (mode 0770)")
}

func TestCheckRefusesAFileOfAnotherUser(t *testing.T) {
	if os.Getuid() == 0 {
		t.Skip("root owns the test files, and root is always accepted")
	}
	fx := newFixture(t, testFile)
	err := checkOwnership(fx.file, os.Getuid()+1)
	if err == nil || !strings.Contains(err.Error(), "owns it, not you") {
		t.Errorf("checkOwnership = %v", err)
	}
}

// The example in the repository is the template for each Machine's own file,
// so pult accepts it with no fault and no warning. The checkout's mode can let
// the group write it, so the test checks a private copy.
func TestTheExampleFilePassesCheck(t *testing.T) {
	src, err := os.ReadFile(filepath.Join("..", "..", "examples", "actions.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	fx := newFixture(t, string(src))
	fx.wantExit(fx.pult("check"), exitOK)
	if !strings.HasSuffix(fx.stdout.String(), " Actions, no faults, 0 warnings\n") || fx.stderr.Len() > 0 {
		t.Errorf("stdout = %q\nstderr = %q", fx.stdout.String(), fx.stderr.String())
	}
}

func TestCheckWithoutAFile(t *testing.T) {
	fx := newFixture(t, testFile)
	if err := os.Remove(fx.file); err != nil {
		t.Fatal(err)
	}
	fx.wantExit(fx.pult("check"), exitFaults)
	fx.wantStderr(fx.file + ": there is no Action file; write one there")
}
