//go:build darwin || linux
// +build darwin linux

package install

import "os/exec"

// hideConsole es no-op en Unix.
func hideConsole(cmd *exec.Cmd) {}
