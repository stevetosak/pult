// Package mobile is the Bridge: the package gomobile binds, through which the
// Android app reaches the Core (ADR 0001). It passes simple types and holds no
// logic of its own; the Bridge step adds its API.
package mobile

import "github.com/stevetosak/pult/core"

// Version returns the version of the Core the app was built with.
func Version() string {
	return core.Version()
}
