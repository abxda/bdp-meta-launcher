//go:build darwin || linux
// +build darwin linux

package ui

// launchedByDoubleClick: en Linux/macOS el patrón de "doble-click en consola"
// no aplica como en Windows. En macOS el .com se suele abrir desde Terminal;
// en Linux desde una shell. Nunca pausamos para no estorbar.
func launchedByDoubleClick() bool { return false }
