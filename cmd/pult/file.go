package main

import (
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"syscall"

	"github.com/stevetosak/pult/action"
)

// actionFile returns the path of the Action file:
// $XDG_CONFIG_HOME/pult/actions.yaml, or ~/.config/pult/actions.yaml when the
// variable is empty or not absolute (the XDG Base Directory rule).
func actionFile(sys *system) (string, error) {
	if dir := sys.getenv("XDG_CONFIG_HOME"); filepath.IsAbs(dir) {
		return filepath.Join(dir, "pult", "actions.yaml"), nil
	}
	home, err := homeDir(sys)
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".config", "pult", "actions.yaml"), nil
}

func homeDir(sys *system) (string, error) {
	home := sys.getenv("HOME")
	if !filepath.IsAbs(home) {
		return "", errors.New("HOME is not an absolute path")
	}
	return home, nil
}

// loadFile reads and parses the Action file, after checking that no other
// user can change it.
func loadFile(path string, sys *system) (*action.File, error) {
	if err := checkOwnership(path, sys.uid); err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return nil, fmt.Errorf("%s: there is no Action file; write one there", path)
		}
		return nil, err
	}
	src, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	return action.Parse(src)
}

// checkOwnership refuses an Action file that another user could change, as
// sshd refuses such an authorized_keys file. The file and its directory, and
// for a symbolic link also the target and the target's directory, must be
// owned by the user or by root and not be writable by group or others.
func checkOwnership(path string, uid int) error {
	target, err := filepath.EvalSymlinks(path)
	if err != nil {
		return err
	}
	seen := map[string]bool{}
	for _, p := range []string{target, filepath.Dir(target), filepath.Dir(path)} {
		if seen[p] {
			continue
		}
		seen[p] = true
		info, err := os.Stat(p)
		if err != nil {
			return err
		}
		if p == target && !info.Mode().IsRegular() {
			return fmt.Errorf("%s: not a regular file", p)
		}
		if perm := info.Mode().Perm(); perm&0o022 != 0 {
			return fmt.Errorf("%s: group or others can write it (mode %04o); fix it with: chmod go-w %s", p, perm, p)
		}
		if st, ok := info.Sys().(*syscall.Stat_t); ok && int(st.Uid) != uid && st.Uid != 0 {
			return fmt.Errorf("%s: uid %d owns it, not you (uid %d) or root", p, st.Uid, uid)
		}
	}
	return nil
}

// printFaults writes each fault as path:line:column: message, or as
// path: message when it has no line. Another error names its path itself.
func printFaults(w io.Writer, path string, err error, prefix string) {
	var faults action.Faults
	if !errors.As(err, &faults) {
		fmt.Fprintf(w, "%s%v\n", prefix, err)
		return
	}
	for _, f := range faults {
		if f.Line > 0 {
			fmt.Fprintf(w, "%s:%d:%d: %s%s\n", path, f.Line, f.Column, prefix, f.Message)
		} else {
			fmt.Fprintf(w, "%s: %s%s\n", path, prefix, f)
		}
	}
}
