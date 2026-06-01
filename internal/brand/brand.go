// Package brand centraliza la identidad visual de la suite "Big Data Lab"
// del Dr. Abel Coronado, para que el Meta-Launcher, el launcher Portable y el
// panel Vagrant compartan UN SOLO look & feel.
//
// La paleta es la misma del design system de los launchers Wails:
//
//	brand   #2563eb (azul)      success #16a34a (verde)
//	warn    #d97706 (amarillo)  danger  #dc2626 (rojo)
//	muted   atenuado/gris
package brand

import (
	"os"
	"runtime"
)

const (
	Suite  = "Big Data Lab"
	Sub    = "Laboratorio de Big Data"
	Author = "Dr. Abel Coronado"
	Tool   = "Meta-Launcher"
)

// Códigos ANSI. Se vacían si la salida no es una terminal (ver Init).
var (
	Reset, Bold, Dim                 = "\033[0m", "\033[1m", "\033[2m"
	Red, Green, Yellow, Blue, Cyan   = "\033[31m", "\033[32m", "\033[33m", "\033[34m", "\033[36m"
)

// Init desactiva los colores cuando no hay terminal (p.ej. salida redirigida a
// un archivo) o cuando el entorno pide NO_COLOR. Llamar una vez al arranque.
func Init() {
	noTTY := false
	if fi, err := os.Stdout.Stat(); err == nil {
		noTTY = (fi.Mode() & os.ModeCharDevice) == 0
	}
	// En Windows clásico la consola puede no interpretar ANSI; Windows Terminal
	// y la consola moderna sí. Mantenemos color salvo redirección o NO_COLOR.
	if noTTY || os.Getenv("NO_COLOR") != "" {
		Reset, Bold, Dim = "", "", ""
		Red, Green, Yellow, Blue, Cyan = "", "", "", "", ""
	}
	_ = runtime.GOOS
}
