// Package core is the Go Core of the Pult app, which the Android app binds
// with gomobile (ADR 0001). For now it holds only its version; the Core step
// adds the Machines, the SSH sessions and the Runs.
package core

// version is set at link time by `make bind` (-ldflags -X).
var version = "dev"

// Version returns the version of the Core the app was built with.
func Version() string {
	return version
}
