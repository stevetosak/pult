package action

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
)

// The SSH hop (ADR 0002, amendment 2026-11-27): SSH carries one command string,
// and the login shell on the Machine reads it before pult starts. Every
// argument that the Core sends therefore holds only A-Z, a-z, 0-9, _ and -,
// in which no login shell finds syntax. The values of a Run cross as the
// base64url form, with no padding, of a JSON object of strings.

// maxEncodedValues is the longest --params argument that DecodeValues reads:
// 8 Parameters of 1024 characters with room for JSON escapes, below the
// kernel's limit of 128 KiB for one argument.
const maxEncodedValues = 96 << 10

const (
	flagExpect = "--expect"
	flagParams = "--params"
)

// RunArgs returns the arguments, after the program name, of the pult run
// command that starts a valid Action with the values the Operator gave:
// run, the id, --expect and the Action's pin, and --params with the encoded
// values when there are any. It checks the values first, so a value that pult
// run would refuse is refused on the phone.
func RunArgs(a Action, given map[string]string) ([]string, error) {
	if err := Validate(a); err != nil {
		return nil, err
	}
	if _, err := a.Bind(given); err != nil {
		return nil, err
	}
	args := []string{"run", a.ID, flagExpect, Pin(a)}
	if len(given) > 0 {
		encoded, err := encodeValues(given)
		if err != nil {
			return nil, err
		}
		args = append(args, flagParams, encoded)
	}
	for _, arg := range args {
		if !safeArg(arg) {
			return nil, fmt.Errorf("action: argument %q holds a character outside A-Z a-z 0-9 _ -", arg)
		}
	}
	return args, nil
}

func encodeValues(values map[string]string) (string, error) {
	// A map of strings encodes with its keys sorted; it cannot fail.
	raw, err := json.Marshal(values)
	if err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(raw), nil
}

// DecodeValues reads the argument of pult run --params: base64url with no
// padding of a JSON object whose values are strings. It checks only the form;
// Action.Bind checks the values.
func DecodeValues(arg string) (map[string]string, error) {
	if len(arg) > maxEncodedValues {
		return nil, fmt.Errorf("%s is %d bytes, more than %d", flagParams, len(arg), maxEncodedValues)
	}
	if !safeArg(arg) {
		return nil, errors.New(flagParams + " holds a character outside A-Z a-z 0-9 _ -")
	}
	raw, err := base64.RawURLEncoding.Strict().DecodeString(arg)
	if err != nil {
		return nil, fmt.Errorf("%s is not base64url: %w", flagParams, err)
	}
	var values map[string]string
	if err := json.Unmarshal(raw, &values); err != nil {
		return nil, fmt.Errorf("%s is not a JSON object of strings: %w", flagParams, err)
	}
	if values == nil {
		return nil, errors.New(flagParams + " is not a JSON object of strings")
	}
	return values, nil
}

// safeArg reports an argument that holds at least one character and only A-Z,
// a-z, 0-9, _ and -.
func safeArg(s string) bool {
	if s == "" {
		return false
	}
	for i := 0; i < len(s); i++ {
		c := s[i]
		if !('a' <= c && c <= 'z' || 'A' <= c && c <= 'Z' || '0' <= c && c <= '9' || c == '_' || c == '-') {
			return false
		}
	}
	return true
}
