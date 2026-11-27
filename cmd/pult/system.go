package main

import (
	"bytes"
	"context"
	"errors"
	"io"
	"os"
	"os/exec"
	"strings"
	"time"

	"golang.org/x/sys/unix"
)

// shell runs every Action: POSIX sh (dash on Ubuntu), as ADR 0003 says.
const shell = "/bin/sh"

// system is what pult takes from the operating system, so that a test can
// replace it.
type system struct {
	stdout, stderr io.Writer
	getenv         func(string) string
	environ        func() []string
	uid            int
	// exec replaces pult with the command. It returns only on a failure.
	exec func(c command) error
	// syntax checks a command with sh -n; name becomes $0.
	syntax func(run, name string) error
}

// command is what pult run replaces itself with.
type command struct {
	path    string
	argv    []string
	env     []string
	dir     string
	noInput bool // the input is /dev/null
}

func hostSystem() *system {
	return &system{
		stdout:  os.Stdout,
		stderr:  os.Stderr,
		getenv:  os.Getenv,
		environ: os.Environ,
		uid:     os.Getuid(),
		exec:    execCommand,
		syntax:  shellSyntax,
	}
}

func execCommand(c command) error {
	if err := os.Chdir(c.dir); err != nil {
		return err
	}
	if c.noInput {
		// The opened file closes at exec (O_CLOEXEC); its copy on 0 stays.
		null, err := os.Open(os.DevNull)
		if err != nil {
			return err
		}
		if err := unix.Dup2(int(null.Fd()), 0); err != nil {
			return err
		}
	}
	return unix.Exec(c.path, c.argv, c.env)
}

// shellSyntax runs sh -n on a command: the syntax check of the shell that
// runs it. It reads the command and starts nothing.
func shellSyntax(run, name string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, shell, "-n", "-c", run, name)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		if msg := strings.TrimSpace(stderr.String()); msg != "" {
			return errors.New(msg)
		}
		return err
	}
	return nil
}
