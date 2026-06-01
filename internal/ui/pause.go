// Package ui tiene ayudas de presentación de consola compartidas, como la
// pausa antes de cerrar cuando el programa se abrió por doble-click (para que
// el alumno alcance a leer el resultado o el error antes de que la ventana
// desaparezca).
package ui

import (
	"bufio"
	"fmt"
	"os"
)

// PauseIfLaunchedByClick espera un Enter SOLO si el programa parece haberse
// abierto con doble-click (no desde una terminal). Así, ejecutado desde una
// shell no estorba, pero por doble-click el alumno puede leer la pantalla.
//
// Llamar al final del programa (defer en main o antes de cada return).
func PauseIfLaunchedByClick() {
	if !launchedByDoubleClick() {
		return
	}
	fmt.Print("\n  Pulsa Enter para cerrar esta ventana… ")
	bufio.NewReader(os.Stdin).ReadString('\n')
}
