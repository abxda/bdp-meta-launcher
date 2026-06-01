// Package install gestiona, para cada solución (Portable / Vagrant), el ciclo
// "¿está instalada? → si no, descargar+extraer → lanzar". Cada solución tiene
// su propia carpeta bajo ~/.bigdatalab/<solucion>/ y un ejecutable de entrada
// definido en el manifiesto (clave <base>.launch).
//
// Descarga diferenciada (decisión de diseño del Dr. Coronado):
//   - Portable: el paquete trae TODO el stack (~GB). Autocontenido.
//   - Vagrant:  el paquete trae solo el panel ligero (~MB); el stack pesado
//     lo baja la caja de Vagrant (vagrant up) desde el HCP Registry, no de HF.
package install

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"

	"github.com/abxda/bdp-meta-launcher/internal/fetch"
	"github.com/abxda/bdp-meta-launcher/internal/platform"
)

// Root es la RAÍZ del laboratorio: la carpeta donde el alumno ejecutó el
// meta-launcher. El .exe es la "semilla" — donde lo pongas, ahí cae todo
// (portable/, vagrant/). Mover el .exe a otra carpeta crea un laboratorio
// nuevo en ese lugar; es predecible y el alumno ve y controla su laboratorio.
//
// La caja de Vagrant NO vive aquí: Vagrant la guarda en su caché global
// (~/.vagrant.d), compartida entre proyectos, así la imagen de ~4.4 GB se baja
// una sola vez.
//
// Excepción de robustez: si el directorio del ejecutable no es escribible
// (p.ej. se ejecutó desde una carpeta protegida o un montaje de solo lectura),
// caemos a ~/.bigdatalab para no fallar. seedDirOverridable permite forzar la
// raíz con BDP_LAB_DIR si algún día hace falta.
func Root() string {
	if v := os.Getenv("BDP_LAB_DIR"); v != "" {
		return v
	}
	dir := executableDir()
	if dir != "" && isWritable(dir) {
		return dir
	}
	// Fallback: home.
	home, err := os.UserHomeDir()
	if err != nil || home == "" {
		home = "."
	}
	return filepath.Join(home, ".bigdatalab")
}

// executableDir devuelve la carpeta donde vive el meta-launcher (resolviendo
// symlinks). Esa es la raíz del laboratorio.
func executableDir() string {
	exe, err := os.Executable()
	if err != nil {
		if cwd, e := os.Getwd(); e == nil {
			return cwd
		}
		return ""
	}
	if resolved, e := filepath.EvalSymlinks(exe); e == nil {
		exe = resolved
	}
	return filepath.Dir(exe)
}

// isWritable comprueba que se puede escribir en dir creando y borrando un
// archivo temporal. Evita elegir como raíz una carpeta de solo lectura.
func isWritable(dir string) bool {
	f, err := os.CreateTemp(dir, ".bdp-write-test-*")
	if err != nil {
		return false
	}
	name := f.Name()
	f.Close()
	os.Remove(name)
	return true
}

// SolutionDir es la carpeta donde se instala una solución.
func SolutionDir(s platform.Solution) string {
	return filepath.Join(Root(), string(s))
}

// Status describe si una solución está instalada y dónde está su ejecutable.
type Status struct {
	Installed  bool
	LaunchPath string // ruta absoluta al ejecutable/app de entrada
}

// manifestKey devuelve la clave base en el manifiesto para (info, solución),
// p.ej. "windows-amd64-portable".
func manifestKey(info platform.Info, s platform.Solution) string {
	return info.OS + "-" + info.Arch + "-" + string(s)
}

// Check averigua si la solución ya está instalada: existe su carpeta y dentro
// el ejecutable de entrada que indica el manifiesto (.launch). Si el manifiesto
// no define .launch, se considera instalada con que la carpeta no esté vacía.
func Check(man fetch.Manifest, info platform.Info, s platform.Solution) Status {
	dir := SolutionDir(s)
	launch := man[manifestKey(info, s)+".launch"]
	if launch != "" {
		p := filepath.Join(dir, launch)
		if _, err := os.Stat(p); err == nil {
			return Status{Installed: true, LaunchPath: p}
		}
		return Status{Installed: false}
	}
	// Sin .launch: instalada si la carpeta existe y tiene contenido.
	entries, err := os.ReadDir(dir)
	if err == nil && len(entries) > 0 {
		return Status{Installed: true, LaunchPath: dir}
	}
	return Status{Installed: false}
}

// Install descarga (si hace falta) y extrae la solución. prog recibe el avance
// de la descarga. Idempotente: si ya está instalada, no hace nada.
func Install(man fetch.Manifest, info platform.Info, s platform.Solution, prog fetch.ProgressFn) (Status, error) {
	if st := Check(man, info, s); st.Installed {
		return st, nil
	}
	base := manifestKey(info, s)
	entry, err := man.Resolve(base)
	if err != nil {
		return Status{}, fmt.Errorf("no hay distribución en el manifiesto para %s (%s): %w", platform.SolutionLabel(s), base, err)
	}

	dir := SolutionDir(s)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return Status{}, err
	}
	archive := filepath.Join(Root(), entry.File)
	if err := fetch.Download(entry, archive, prog); err != nil {
		return Status{}, fmt.Errorf("descarga: %w", err)
	}
	if err := fetch.ExtractTarGz(archive, dir); err != nil {
		return Status{}, fmt.Errorf("descompresión: %w", err)
	}
	_ = os.Remove(archive) // el .tar.gz ya no se necesita tras extraer

	st := Check(man, info, s)
	if !st.Installed {
		return st, fmt.Errorf("se extrajo pero no encontré el ejecutable de entrada (%s)", entry.LaunchAt)
	}
	return st, nil
}

// Launch lanza el ejecutable de entrada de una solución ya instalada. En macOS,
// si es un .app, usa `open`. En el resto, ejecuta el binario directamente,
// restaurando el bit de ejecución (puede perderse al extraer en algunos FS).
func Launch(st Status) error {
	if !st.Installed {
		return fmt.Errorf("la solución no está instalada")
	}
	if runtime.GOOS == "darwin" && filepath.Ext(st.LaunchPath) == ".app" {
		return exec.Command("open", st.LaunchPath).Start()
	}
	if runtime.GOOS != "windows" {
		_ = os.Chmod(st.LaunchPath, 0o755)
	}
	cmd := exec.Command(st.LaunchPath)
	cmd.Dir = filepath.Dir(st.LaunchPath)
	return cmd.Start()
}
