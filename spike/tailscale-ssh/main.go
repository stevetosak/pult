// Command spike logs in to a Tailscale SSH server with no key (the client's
// "none" method) and reports what the Pult Core needs to know: every banner,
// the host keys, a command's output and exit code, and a PTY's size before and
// after a window change.
//
// It is the one-time spike of Pult Plan step "Spike: golang.org/x/crypto/ssh
// logs in to Tailscale SSH with no key". It lives on a branch and is not
// merged; the Core's SSH code replaces it.
package main

import (
	"bufio"
	"bytes"
	"errors"
	"flag"
	"fmt"
	"io"
	"net"
	"os"
	"strings"
	"time"

	"golang.org/x/crypto/ssh"
)

const (
	dialTimeout = 15 * time.Second
	lineTimeout = 10 * time.Second

	ptyRows, ptyCols       = 40, 100
	ptyNewRows, ptyNewCols = 50, 120
)

// The commands the spike runs on the Machine. The wrong pin must make pult run
// refuse with 125 before vault-status starts; vault-status only reads the
// vault state, so a missing refusal changes nothing.
const (
	listCommand     = "pult list --json"
	wrongPinCommand = "pult run vault-status --param vault=personal --expect wrong-pin"
	ptyCommand      = "trap 'echo WINCH' WINCH; stty size; read x; stty size"
)

// errProbeDone stops a handshake once the host key is read.
var errProbeDone = errors.New("host key read")

func main() {
	addr := flag.String("addr", "", "host:port of the Machine; a tailnet IP, because the phone has no resolver")
	user := flag.String("user", "", "the user on the Machine")
	flag.Parse()
	if *addr == "" || *user == "" {
		fmt.Fprintln(os.Stderr, "usage: spike -addr <ip>:22 -user <name>")
		os.Exit(2)
	}
	if err := run(*addr, *user); err != nil {
		fmt.Fprintln(os.Stderr, "FAIL:", err)
		os.Exit(1)
	}
	fmt.Println("== done")
}

func run(addr, user string) error {
	probeHostKeys(addr, user)

	client, err := login(addr, user)
	if err != nil {
		return err
	}
	defer client.Close()

	for _, cmd := range []string{listCommand, wrongPinCommand} {
		if err := runCommand(client, cmd); err != nil {
			return err
		}
	}
	return ptyWindowChange(client)
}

// probeHostKeys reads the key the server shows for each host key algorithm,
// so that every key can be compared with the Machine's own list.
func probeHostKeys(addr, user string) {
	fmt.Println("== host keys offered")
	algorithms := []string{ssh.KeyAlgoED25519, ssh.KeyAlgoECDSA256, ssh.KeyAlgoRSASHA256}
	for _, algorithm := range algorithms {
		var key ssh.PublicKey
		cfg := &ssh.ClientConfig{
			User:              user,
			HostKeyAlgorithms: []string{algorithm},
			HostKeyCallback: func(_ string, _ net.Addr, k ssh.PublicKey) error {
				key = k
				return errProbeDone
			},
			Timeout: dialTimeout,
		}
		_, err := ssh.Dial("tcp", addr, cfg)
		if key == nil {
			fmt.Printf("%-22s none: %v\n", algorithm, err)
			continue
		}
		fmt.Printf("%-22s %s %s\n", algorithm, key.Type(), ssh.FingerprintSHA256(key))
	}
}

// login connects with no AuthMethod: x/crypto/ssh always tries "none" first,
// so the login succeeds only when the server accepts "none".
func login(addr, user string) (*ssh.Client, error) {
	var hostKey ssh.PublicKey
	started := time.Now()
	elapsed := func() time.Duration { return time.Since(started).Round(time.Millisecond) }
	cfg := &ssh.ClientConfig{
		User: user,
		HostKeyCallback: func(_ string, _ net.Addr, k ssh.PublicKey) error {
			hostKey = k // reported, not checked: the Core pins it
			return nil
		},
		BannerCallback: func(message string) error {
			fmt.Printf("== banner (%d bytes) after %s\n%s\n", len(message), elapsed(), message)
			return nil
		},
		Timeout: dialTimeout,
	}
	client, err := ssh.Dial("tcp", addr, cfg)
	fmt.Println("== login")
	if hostKey != nil {
		fmt.Printf("negotiated host key: %s %s\n", hostKey.Type(), ssh.FingerprintSHA256(hostKey))
	}
	if err != nil {
		return nil, fmt.Errorf("login after %s: %w", elapsed(), err)
	}
	fmt.Printf("method none accepted after %s; server %q\n", elapsed(), client.ServerVersion())
	return client, nil
}

// runCommand runs one command without a PTY and prints its exit code, stdout
// and stderr apart.
func runCommand(client *ssh.Client, cmd string) error {
	session, err := client.NewSession()
	if err != nil {
		return fmt.Errorf("session for %q: %w", cmd, err)
	}
	defer session.Close()

	var stdout, stderr bytes.Buffer
	session.Stdout, session.Stderr = &stdout, &stderr
	code, err := exitCode(session.Run(cmd))
	if err != nil {
		return fmt.Errorf("%q: %w", cmd, err)
	}
	fmt.Printf("== command %q\nexit %d\n-- stdout (%d bytes)\n%s\n-- stderr (%d bytes)\n%s\n",
		cmd, code, stdout.Len(), stdout.String(), stderr.Len(), stderr.String())
	return nil
}

// ptyWindowChange runs ptyCommand in a PTY: it reads the first size, sends a
// window change, then lets the command read its line and print the new size.
func ptyWindowChange(client *ssh.Client) error {
	session, err := client.NewSession()
	if err != nil {
		return fmt.Errorf("pty session: %w", err)
	}
	defer session.Close()

	modes := ssh.TerminalModes{ssh.ECHO: 1, ssh.TTY_OP_ISPEED: 38400, ssh.TTY_OP_OSPEED: 38400}
	if err := session.RequestPty("xterm-256color", ptyRows, ptyCols, modes); err != nil {
		return fmt.Errorf("pty request: %w", err)
	}
	stdin, err := session.StdinPipe()
	if err != nil {
		return err
	}
	stdout, err := session.StdoutPipe()
	if err != nil {
		return err
	}
	if err := session.Start(ptyCommand); err != nil {
		return fmt.Errorf("pty start: %w", err)
	}

	lines := readLines(stdout)
	var seen []string
	firstSize := fmt.Sprintf("%d %d", ptyRows, ptyCols)
	if err := waitForLine(lines, firstSize, &seen); err != nil {
		return fmt.Errorf("pty first size: %w (seen %q)", err, seen)
	}
	if err := session.WindowChange(ptyNewRows, ptyNewCols); err != nil {
		return fmt.Errorf("window change: %w", err)
	}
	time.Sleep(500 * time.Millisecond) // lets the WINCH trap print before the read ends
	if _, err := io.WriteString(stdin, "\n"); err != nil {
		return fmt.Errorf("pty input: %w", err)
	}
	for line := range lines {
		seen = append(seen, line)
	}
	code, err := exitCode(session.Wait())
	if err != nil {
		return fmt.Errorf("pty wait: %w", err)
	}
	fmt.Printf("== pty %dx%d, then window change to %dx%d\ncommand %q\nexit %d\n",
		ptyRows, ptyCols, ptyNewRows, ptyNewCols, ptyCommand, code)
	for _, line := range seen {
		fmt.Printf("| %q\n", line)
	}
	return nil
}

// readLines sends each line of r, without its CR LF, until r ends.
func readLines(r io.Reader) <-chan string {
	lines := make(chan string)
	go func() {
		defer close(lines)
		scanner := bufio.NewScanner(r)
		for scanner.Scan() {
			lines <- strings.TrimRight(scanner.Text(), "\r")
		}
	}()
	return lines
}

// waitForLine reads lines into seen until one equals want.
func waitForLine(lines <-chan string, want string, seen *[]string) error {
	deadline := time.After(lineTimeout)
	for {
		select {
		case line, ok := <-lines:
			if !ok {
				return io.ErrUnexpectedEOF
			}
			*seen = append(*seen, line)
			if line == want {
				return nil
			}
		case <-deadline:
			return fmt.Errorf("no line %q after %s", want, lineTimeout)
		}
	}
}

// exitCode turns the result of Run or Wait into the remote exit code; any
// other error (no exit status, a broken connection) is returned.
func exitCode(err error) (int, error) {
	var exitErr *ssh.ExitError
	switch {
	case err == nil:
		return 0, nil
	case errors.As(err, &exitErr):
		return exitErr.ExitStatus(), nil
	default:
		return 0, err
	}
}
