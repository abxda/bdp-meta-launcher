// Package platform detecta el sistema operativo y la arquitectura, y resuelve
// la matriz de decisión: qué solución(es) del laboratorio aplican.
package platform

import "runtime"

// Solution identifica una de las soluciones del laboratorio.
type Solution string

const (
	Portable  Solution = "portable"
	Vagrant   Solution = "vagrant"
	Container Solution = "container" // laboratorio en contenedor (Podman)
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
// La matriz lista TODAS las soluciones posibles por equipo; el meta-launcher
// luego solo OFRECE las que tienen un artefacto lanzable publicado en el
// manifest (ver diagnose en main.go). Container (Podman) aplica donde corre
// Podman = todos.
//
//	Windows amd64        -> Portable, Vagrant, Container
//	Linux   amd64        -> Portable, Vagrant, Container
//	macOS   amd64 (Intel)-> Vagrant, Container
//	macOS   arm64 (Apple)-> Portable, Container
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
			in.Solutions = []Solution{Portable, Vagrant, Container}
		}
	case "linux":
		if runtime.GOARCH == "amd64" {
			in.Solutions = []Solution{Portable, Vagrant, Container}
		}
	case "darwin":
		if runtime.GOARCH == "arm64" {
			in.Solutions = []Solution{Portable, Container}
		} else if runtime.GOARCH == "amd64" {
			in.Solutions = []Solution{Vagrant, Container}
		}
	}
	return in
}

// SolutionLabel y SolutionDesc dan los textos en español, homologados con el
// resto de la suite.
func SolutionLabel(s Solution) string {
	switch s {
	case Portable:
		return "Portable"
	case Container:
		return "Container (Podman)"
	default:
		return "Vagrant"
	}
}

func SolutionDesc(s Solution) string {
	switch s {
	case Portable:
		return "Distribución autocontenida: copia los binarios y ejecuta el launcher. Sin virtualización."
	case Container:
		return "Laboratorio en contenedor (Podman): imagen precompilada, sin VM. El launcher instala Podman y descarga la imagen."
	default:
		return "Máquina virtual (VirtualBox + Vagrant) con todo el stack preinstalado."
	}
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
