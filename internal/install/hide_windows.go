//go:build windows
// +build windows

package install

import (
	"os/exec"
	"syscall"
)

// hideConsole evita que VBoxManage abra una ventana de consola al consultarse.
func hideConsole(cmd *exec.Cmd) {
	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true, CreationFlags: 0x08000000}
}
