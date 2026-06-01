//go:build windows
// +build windows

package ui

import (
	"path/filepath"
	"strings"
	"unsafe"

	"golang.org/x/sys/windows"
)

// launchedByDoubleClick reporta si el proceso se abrió con doble-click en el
// Explorador. Heurística: si el proceso PADRE es explorer.exe, fue doble-click;
// si es una shell (cmd, powershell, pwsh, bash, sh, wt), se ejecutó desde una
// terminal y no debemos pausar.
func launchedByDoubleClick() bool {
	parent := parentProcessName()
	if parent == "" {
		return false // no pudimos saber; mejor no pausar para no estorbar
	}
	parent = strings.ToLower(parent)
	switch parent {
	case "explorer.exe":
		return true
	case "cmd.exe", "powershell.exe", "pwsh.exe", "bash.exe", "sh.exe",
		"windowsterminal.exe", "wt.exe", "conhost.exe", "code.exe":
		return false
	}
	// Padre desconocido (p.ej. lanzado por otra app): no pausar.
	return false
}

// parentProcessName devuelve el nombre del ejecutable del proceso padre usando
// el snapshot de Toolhelp. Cadena vacía si no se puede determinar.
func parentProcessName() string {
	snap, err := windows.CreateToolhelp32Snapshot(windows.TH32CS_SNAPPROCESS, 0)
	if err != nil {
		return ""
	}
	defer windows.CloseHandle(snap)

	me := uint32(windows.GetCurrentProcessId())
	var ppid uint32
	var entry windows.ProcessEntry32
	entry.Size = uint32(unsafe.Sizeof(entry))

	// Primer recorrido: encontrar nuestro PPID.
	if err := windows.Process32First(snap, &entry); err != nil {
		return ""
	}
	found := false
	for {
		if entry.ProcessID == me {
			ppid = entry.ParentProcessID
			found = true
			break
		}
		if err := windows.Process32Next(snap, &entry); err != nil {
			break
		}
	}
	if !found {
		return ""
	}

	// Segundo recorrido: encontrar el nombre del PPID.
	if err := windows.Process32First(snap, &entry); err != nil {
		return ""
	}
	for {
		if entry.ProcessID == ppid {
			name := windows.UTF16ToString(entry.ExeFile[:])
			return filepath.Base(name)
		}
		if err := windows.Process32Next(snap, &entry); err != nil {
			break
		}
	}
	return ""
}

