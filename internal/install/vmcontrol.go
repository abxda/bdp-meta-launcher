package install

import (
	"os"
	"os/exec"
	"runtime"
	"strings"
	"time"
)

// VMName es el nombre FIJO que el Vagrantfile da a la máquina virtual del
// laboratorio. VirtualBox la gestiona globalmente por ese nombre, así que el
// meta-launcher puede apagarla sin saber desde qué carpeta-semilla se levantó.
const VMName = "BDP-BigDataLab"

// VagrantVMRunning reporta si la VM del laboratorio está corriendo (por nombre).
func VagrantVMRunning() bool {
	out, err := vboxOut("list", "runningvms")
	if err != nil {
		return false
	}
	return strings.Contains(out, `"`+VMName+`"`)
}

// ShutdownVagrantVM apaga la VM por NOMBRE con la señal ACPI (apagado limpio,
// el SO invitado cierra ordenado). No necesita el directorio del Vagrantfile.
// Espera hasta timeout a que se apague; devuelve true si lo confirmó.
func ShutdownVagrantVM(timeout time.Duration) bool {
	if !VagrantVMRunning() {
		return true
	}
	_, _ = vboxOut("controlvm", VMName, "acpipowerbutton")
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if !VagrantVMRunning() {
			return true
		}
		time.Sleep(2 * time.Second)
	}
	return !VagrantVMRunning()
}

func vboxOut(args ...string) (string, error) {
	cmd := exec.Command(vboxManageBin(), args...)
	hideConsole(cmd)
	out, err := cmd.CombinedOutput()
	return string(out), err
}

func vboxManageBin() string {
	// PATH primero.
	name := "VBoxManage"
	if runtime.GOOS == "windows" {
		name = "VBoxManage.exe"
	}
	if p, err := exec.LookPath(name); err == nil {
		return p
	}
	// Rutas conocidas por SO.
	for _, c := range vboxKnownPaths() {
		if fileExists(c) {
			return c
		}
	}
	return name
}

func vboxKnownPaths() []string {
	switch runtime.GOOS {
	case "windows":
		return []string{`C:\Program Files\Oracle\VirtualBox\VBoxManage.exe`}
	case "darwin":
		return []string{"/usr/local/bin/VBoxManage", "/opt/homebrew/bin/VBoxManage", "/Applications/VirtualBox.app/Contents/MacOS/VBoxManage"}
	default:
		return []string{"/usr/bin/VBoxManage", "/usr/local/bin/VBoxManage"}
	}
}

func fileExists(p string) bool {
	fi, err := os.Stat(p)
	return err == nil && !fi.IsDir()
}
