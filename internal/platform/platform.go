// Package platform detecta el sistema operativo y la arquitectura, y resuelve
// la matriz de decisión: qué solución(es) del laboratorio aplican.
package platform

import "runtime"

// Solution identifica una de las soluciones del laboratorio.
type Solution string

const (
	Portable Solution = "portable"
	Vagrant  Solution = "vagrant"
)

// Info describe el entorno actual.
type Info struct {
	OS        string     // "windows", "linux", "darwin"
	Arch      string     // "amd64", "arm64"
	OSLabel   string     // legible en español
	ArchLabel string     // legible en español
	Solutions []Solution // soluciones aplicables, en orden de recomendación
}

// Detect inspecciona la máquina actual y devuelve su Info con la matriz ya
// resuelta:
//
//	Windows amd64        -> Portable, Vagrant
//	Linux   amd64        -> Vagrant
//	macOS   amd64 (Intel)-> Vagrant
//	macOS   arm64 (Apple)-> Portable
func Detect() Info {
	in := Info{
		OS:        runtime.GOOS,
		Arch:      runtime.GOARCH,
		OSLabel:   osLabel(runtime.GOOS),
		ArchLabel: archLabel(runtime.GOARCH),
	}
	switch runtime.GOOS {
	case "windows":
		if runtime.GOARCH == "amd64" {
			in.Solutions = []Solution{Portable, Vagrant}
		}
	case "linux":
		if runtime.GOARCH == "amd64" {
			in.Solutions = []Solution{Vagrant}
		}
	case "darwin":
		if runtime.GOARCH == "arm64" {
			in.Solutions = []Solution{Portable}
		} else if runtime.GOARCH == "amd64" {
			in.Solutions = []Solution{Vagrant}
		}
	}
	return in
}

// SolutionLabel y SolutionDesc dan los textos en español, homologados con el
// resto de la suite.
func SolutionLabel(s Solution) string {
	if s == Portable {
		return "Portable"
	}
	return "Vagrant"
}

func SolutionDesc(s Solution) string {
	if s == Portable {
		return "Distribución autocontenida: copia los binarios y ejecuta el launcher. Sin virtualización."
	}
	return "Máquina virtual (VirtualBox + Vagrant) con todo el stack preinstalado."
}

// ManifestKey devuelve la clave base en el manifest de Hugging Face para esta
// combinación (os-arch-solucion), p.ej. "windows-amd64-portable".
func (in Info) ManifestKey(s Solution) string {
	return in.OS + "-" + in.Arch + "-" + string(s)
}

func osLabel(goos string) string {
	switch goos {
	case "windows":
		return "Windows"
	case "linux":
		return "Linux"
	case "darwin":
		return "macOS"
	default:
		return goos
	}
}

func archLabel(goarch string) string {
	switch goarch {
	case "amd64":
		return "x86-64 (Intel/AMD)"
	case "arm64":
		return "ARM64 (Apple Silicon)"
	default:
		return goarch
	}
}
