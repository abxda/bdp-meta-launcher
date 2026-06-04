// BDP Meta-Launcher (implementación Go, nativa por plataforma).
//
// Punto único de entrada de la suite "Big Data Lab" del Dr. Abel Coronado.
// Diagnostica el sistema (SO + arquitectura), decide qué solución del
// laboratorio aplica (Portable / Vagrant), la descarga desde Hugging Face,
// verifica su integridad (SHA-256) y la descomprime.
//
// Por qué Go nativo en Windows en vez de Cosmopolitan: el .com de Cosmopolitan
// dispara falsos positivos de EDR (parent_process_spoofing) en equipos
// administrados — justo el público objetivo. Esta versión es un PE/ELF/Mach-O
// normal, sin trucos de proceso, y no depende de curl/tar externos (usa la
// stdlib), así que no asusta al antivirus del alumno. El .com de Cosmopolitan
// se conserva en cosmo/ para Linux/macOS, donde el problema no existe.
package main

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/abxda/bdp-meta-launcher/internal/brand"
	"github.com/abxda/bdp-meta-launcher/internal/fetch"
	"github.com/abxda/bdp-meta-launcher/internal/install"
	"github.com/abxda/bdp-meta-launcher/internal/platform"
	"github.com/abxda/bdp-meta-launcher/internal/ui"
)

const version = "0.2.0"

func main() {
	brand.Init()
	code := run()
	// Pausa antes de cerrar SOLO si se abrió por doble-click (para que el
	// alumno alcance a leer el resultado antes de que la ventana desaparezca).
	ui.PauseIfLaunchedByClick()
	os.Exit(code)
}

// run despacha el subcomando y devuelve el código de salida, sin llamar a
// os.Exit (para que la pausa de main siempre tenga oportunidad de correr).
func run() int {
	args := os.Args[1:]
	switch {
	case len(args) > 0 && args[0] == "--self-test":
		return selfTest()
	case len(args) > 0 && (args[0] == "--help" || args[0] == "-h"):
		help()
		return 0
	case len(args) > 0 && args[0] == "--version":
		fmt.Printf("%s %s v%s\n", brand.Suite, brand.Tool, version)
		return 0
	default:
		return diagnose()
	}
}

func banner() {
	hr := strings.Repeat("=", 64)
	fmt.Printf("%s%s%s\n", brand.Dim, hr, brand.Reset)
	fmt.Printf("  %s%s%s%s  %s- %s%s\n", brand.Bold, brand.Cyan, brand.Suite, brand.Reset, brand.Dim, brand.Sub, brand.Reset)
	fmt.Printf("  %s%s  ·  %s%s\n", brand.Dim, brand.Tool, brand.Author, brand.Reset)
	fmt.Printf("%s%s%s\n", brand.Dim, hr, brand.Reset)
}

func step(msg string) { fmt.Printf("  %s>%s %s\n", brand.Cyan, brand.Reset, msg) }
func ok(msg string)   { fmt.Printf("  %s✓%s %s\n", brand.Green, brand.Reset, msg) }
func bad(msg string)  { fmt.Printf("  %s[!]%s %s\n", brand.Red, brand.Reset, msg) }

func diagnose() int {
	banner()
	in := platform.Detect()
	fmt.Printf("\n  %sTu equipo%s: %s%s · %s%s\n",
		brand.Bold, brand.Reset, brand.Green, in.OSLabel, in.ArchLabel, brand.Reset)
	fmt.Printf("  %sTu laboratorio se crea en:%s %s%s%s\n",
		brand.Bold, brand.Reset, brand.Cyan, install.Root(), brand.Reset)
	fmt.Printf("  %s(es la carpeta donde está este programa; muévelo y se moverá tu laboratorio)%s\n",
		brand.Dim, brand.Reset)

	if len(in.Solutions) == 0 {
		bad("Combinación no soportada todavía.")
		fmt.Printf("    %sEscribe al %s para soporte.%s\n\n", brand.Dim, brand.Author, brand.Reset)
		return 2
	}

	// Necesitamos el manifiesto para saber qué está instalado y descargar.
	man, err := fetch.FetchManifest()
	if err != nil {
		bad("No pude leer el catálogo de soluciones: " + err.Error())
		fmt.Printf("    %sRevisa tu conexión a internet e inténtalo de nuevo.%s\n\n", brand.Dim, brand.Reset)
		return 1
	}

	// Ofrecer SOLO las soluciones con un artefacto LANZABLE publicado en el
	// manifest (.launch presente). Así el alumno ve únicamente lo que de verdad
	// funciona para su equipo, y cada solución nueva (p.ej. Container) o cada
	// plataforma se "enciende" sola al publicar su entrada — sin recompilar ni
	// mergear. Reduce la carga cognitiva: nunca ve una opción que no existe.
	avail := in.Solutions[:0:0]
	for _, s := range in.Solutions {
		if man[in.ManifestKey(s)+".launch"] != "" {
			avail = append(avail, s)
		}
	}
	in.Solutions = avail
	if len(in.Solutions) == 0 {
		bad("Tu equipo está soportado, pero aún no hay una solución PUBLICADA para él.")
		fmt.Printf("    %sRevisa el foro: publicaré los enlaces actualizados a lo largo de la semana.%s\n\n", brand.Dim, brand.Reset)
		return 0
	}

	// Elegir solución: si solo hay una, va directa; si hay varias, menú amable.
	chosen := in.Solutions[0]
	if len(in.Solutions) > 1 {
		c, quit := chooseSolution(in, man)
		if quit {
			fmt.Printf("\n  %sHasta luego.%s\n\n", brand.Dim, brand.Reset)
			return 0
		}
		chosen = c
	} else {
		fmt.Printf("\n  %sSolución para tu equipo:%s %s%s%s\n",
			brand.Bold, brand.Reset, brand.Blue, platform.SolutionLabel(chosen), brand.Reset)
		fmt.Printf("    %s%s%s\n", brand.Dim, platform.SolutionDesc(chosen), brand.Reset)
	}

	return prepareAndLaunch(man, in, chosen)
}

// chooseSolution muestra el menú amable de elección (Windows, 2 opciones) con
// el estado de cada solución (instalada o no) y tamaño de descarga. Devuelve la
// solución elegida, o quit=true si el alumno sale.
func chooseSolution(in platform.Info, man fetch.Manifest) (platform.Solution, bool) {
	for {
		fmt.Printf("\n  %s¿Qué quieres usar hoy?%s\n\n", brand.Bold, brand.Reset)
		for i, s := range in.Solutions {
			st := install.Check(man, in, s)
			tag := brand.Dim + "(no instalado)" + brand.Reset
			if st.Installed {
				tag = brand.Green + "✓ instalado" + brand.Reset
			}
			size := manifestSize(man, in, s)
			fmt.Printf("    %s%d)%s %s%s%s  %s\n", brand.Bold, i+1, brand.Reset,
				brand.Blue, platform.SolutionLabel(s), brand.Reset, tag)
			fmt.Printf("       %s%s%s\n", brand.Dim, platform.SolutionDesc(s), brand.Reset)
			if size != "" {
				fmt.Printf("       %sDescarga: %s%s\n", brand.Dim, size, brand.Reset)
			}
		}
		fmt.Printf("    %sq)%s Salir\n", brand.Bold, brand.Reset)
		fmt.Printf("\n  %sElige una opción [1-%d, q]:%s ", brand.Bold, len(in.Solutions), brand.Reset)

		choice, gotInput := readLine()
		if !gotInput {
			fmt.Printf("\n  %sSin entrada interactiva (stdin no es una terminal). Saliendo.%s\n", brand.Dim, brand.Reset)
			return "", true
		}
		choice = strings.TrimSpace(strings.ToLower(choice))
		if choice == "q" {
			return "", true
		}
		n, err := strconv.Atoi(choice)
		if err == nil && n >= 1 && n <= len(in.Solutions) {
			return in.Solutions[n-1], false
		}
		fmt.Printf("  %sOpción no válida.%s\n", brand.Yellow, brand.Reset)
	}
}

// prepareAndLaunch instala (si hace falta) la solución elegida y la lanza.
func prepareAndLaunch(man fetch.Manifest, in platform.Info, s platform.Solution) int {
	// Conflicto de puertos: Portable y Vagrant usan los mismos puertos del
	// stack (9870/9200/8888), así que no pueden correr a la vez. Si ya hay un
	// laboratorio en marcha, lo resolvemos antes de lanzar.
	if install.LabRunning() {
		fmt.Printf("\n  %s[!] Ya hay un laboratorio en marcha.%s\n", brand.Yellow, brand.Reset)
		fmt.Printf("    %sLas dos soluciones usan los mismos puertos (9870, 9200, 8888),\n    así que no pueden estar encendidas a la vez.%s\n", brand.Dim, brand.Reset)

		// Si lo que corre es la VM de Vagrant, podemos apagarla por nombre
		// (cerrado elegante) sin saber dónde se levantó.
		if install.VagrantVMRunning() {
			fmt.Printf("\n  %sDetecté la máquina virtual de Vagrant encendida.%s\n", brand.Bold, brand.Reset)
			fmt.Printf("  %s¿Apagarla limpiamente para continuar? (tu trabajo se conserva) [S/n]:%s ", brand.Bold, brand.Reset)
			line, gotInput := readLine()
			if !gotInput {
				fmt.Printf("\n  %sSin entrada interactiva: no apago nada. Vuelve a ejecutarme en una terminal.%s\n\n", brand.Dim, brand.Reset)
				return 0
			}
			ans := strings.TrimSpace(strings.ToLower(line))
			if ans == "" || ans == "s" || ans == "si" || ans == "sí" {
				step("Apagando la máquina virtual de Vagrant (puede tardar ~30s)…")
				if install.ShutdownVagrantVM(90 * time.Second) {
					ok("Máquina virtual apagada. Puertos liberados.")
				} else {
					bad("No confirmé el apagado a tiempo. Apaga la VM manualmente y reintenta.")
					return 1
				}
			} else {
				fmt.Printf("\n  %sBien. Apaga la VM cuando quieras y vuelve a ejecutarme.%s\n\n", brand.Dim, brand.Reset)
				return 0
			}
		} else {
			// Es el Portable (procesos locales): el meta-launcher no los mata;
			// el alumno cierra el Portable (al cerrar detiene sus servicios).
			fmt.Printf("    %sParece ser el Portable. Ciérralo (al cerrar detiene sus servicios)\n    y vuelve a ejecutarme.%s\n", brand.Dim, brand.Reset)
			fmt.Printf("\n  %s¿Lanzar %s de todos modos? Podría fallar por puertos ocupados. [s/N]:%s ",
				brand.Bold, platform.SolutionLabel(s), brand.Reset)
			line, gotInput := readLine()
			if !gotInput {
				fmt.Printf("\n  %sSin entrada interactiva: no lanzo nada para no chocar con los puertos.%s\n\n", brand.Dim, brand.Reset)
				return 0
			}
			ans := strings.TrimSpace(strings.ToLower(line))
			if ans != "s" && ans != "si" && ans != "sí" {
				fmt.Printf("\n  %sBien, hasta luego.%s\n\n", brand.Dim, brand.Reset)
				return 0
			}
		}
	}

	st := install.Check(man, in, s)
	if !st.Installed {
		fmt.Printf("\n  %sPreparando %s%s%s por primera vez…%s\n",
			brand.Bold, brand.Blue, platform.SolutionLabel(s), brand.Reset, brand.Reset)
		if s == platform.Vagrant {
			fmt.Printf("    %sSe descarga solo el panel (ligero). La máquina virtual con el stack\n    se baja después, dentro del panel.%s\n", brand.Dim, brand.Reset)
		} else {
			fmt.Printf("    %sSe descarga la distribución completa (autocontenida). Puede tardar\n    según tu conexión; solo ocurre la primera vez.%s\n", brand.Dim, brand.Reset)
		}
		onPhase := func(phase string) {
			switch phase {
			case "extrayendo":
				fmt.Println()
				step("Descomprimiendo… (puede tardar un poco con la distro completa; no cierres)")
			}
		}
		newSt, err := install.Install(man, in, s, progressBar, onPhase)
		fmt.Println()
		if err != nil {
			bad("No pude preparar la solución: " + err.Error())
			return 1
		}
		ok("Solución preparada en disco.")
		st = newSt
	} else {
		fmt.Printf("\n  %s✓ %s ya está instalada.%s\n", brand.Green, platform.SolutionLabel(s), brand.Reset)
	}

	step("Lanzando " + platform.SolutionLabel(s) + "…")
	if err := install.Launch(st); err != nil {
		bad("No pude lanzar la aplicación: " + err.Error())
		fmt.Printf("    %sLa encontrarás en: %s%s\n", brand.Dim, st.LaunchPath, brand.Reset)
		return 1
	}
	ok("¡Listo! La aplicación se está abriendo.")
	fmt.Printf("  %sPuedes cerrar esta ventana.%s\n\n", brand.Dim, brand.Reset)
	return 0
}

// manifestSize devuelve la etiqueta de tamaño de descarga (<base>.size) si el
// manifiesto la trae, p.ej. "~2 GB". Solo informativo.
func manifestSize(man fetch.Manifest, in platform.Info, s platform.Solution) string {
	return man[in.OS+"-"+in.Arch+"-"+string(s)+".size"]
}

// stdinReader es un único lector compartido: recrear bufio.NewReader en cada
// llamada puede descartar bytes ya bufferizados entre lecturas.
var stdinReader = bufio.NewReader(os.Stdin)

// readLine lee una línea de stdin (la elección del menú). ok=false cuando se
// alcanza EOF sin datos: stdin no es una terminal (redirigido, cerrado o sin
// TTY). El llamador DEBE tratarlo como "salir" para no caer en un bucle
// infinito que reimprime el menú y satura la salida.
func readLine() (string, bool) {
	line, err := stdinReader.ReadString('\n')
	if line == "" && err != nil {
		return "", false
	}
	return line, true
}

func selfTest() int {
	banner()
	fmt.Printf("\n  %sAuto-prueba CP2%s — descarga + verificación + descompresión\n", brand.Bold, brand.Reset)
	fmt.Printf("  %sUsa el fixture 'test' del repo Hugging Face.%s\n\n", brand.Dim, brand.Reset)

	step("Leyendo el manifiesto de distribuciones desde Hugging Face…")
	man, err := fetch.FetchManifest()
	if err != nil {
		bad("No pude leer el manifiesto: " + err.Error())
		return 1
	}
	entry, err := man.Resolve("test")
	if err != nil {
		bad(err.Error())
		return 1
	}

	wd := workDir()
	dest := filepath.Join(wd, entry.File)
	step(fmt.Sprintf("Descargando %s%s%s …", brand.Bold, entry.File, brand.Reset))
	if err := fetch.Download(entry, dest, progressBar); err != nil {
		fmt.Println()
		bad("Falló la descarga/verificación: " + err.Error())
		return 1
	}
	fmt.Println()
	ok("Integridad verificada (SHA-256).")

	step(fmt.Sprintf("Descomprimiendo en %s%s%s …", brand.Bold, wd, brand.Reset))
	last := 0
	if err := fetch.ExtractTarGzProgress(dest, wd, func(files int, b int64) {
		if files-last >= 400 {
			last = files
			fmt.Printf("\r    descomprimiendo… %d archivos (%d MB)   ", files, b/(1<<20))
		}
	}); err != nil {
		fmt.Println()
		bad("Falló la descompresión: " + err.Error())
		return 1
	}
	fmt.Println()
	ok("Solución preparada en disco.")
	fmt.Println()
	ok("Auto-prueba CP2 EXITOSA: la cadena de descarga funciona en esta plataforma.")
	fmt.Printf("  %sSin dependencias externas (curl/tar) y sin alertas de antivirus.%s\n\n", brand.Dim, brand.Reset)
	return 0
}

// progressBar dibuja una barra ANSI in-place a partir del progreso.
func progressBar(done, total int64) {
	const width = 32
	if total <= 0 {
		fmt.Printf("\r  %s%s%s descargados…", brand.Cyan, humanBytes(done), brand.Reset)
		return
	}
	frac := float64(done) / float64(total)
	if frac > 1 {
		frac = 1
	}
	filled := int(frac * width)
	bar := strings.Repeat("█", filled) + strings.Repeat("░", width-filled)
	fmt.Printf("\r  %s%s%s %3d%%  %s / %s", brand.Cyan, bar, brand.Reset,
		int(frac*100), humanBytes(done), humanBytes(total))
}

func humanBytes(n int64) string {
	const u = 1024
	if n < u {
		return fmt.Sprintf("%d B", n)
	}
	div, exp := int64(u), 0
	for x := n / u; x >= u; x /= u {
		div *= u
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(n)/float64(div), "KMGTPE"[exp])
}

// workDir devuelve <home>/.bigdatalab, donde se descargan/extraen soluciones.
func workDir() string {
	home, err := os.UserHomeDir()
	if err != nil || home == "" {
		home = "."
	}
	return filepath.Join(home, ".bigdatalab")
}

func help() {
	fmt.Printf("%s — %s (%s)\n", brand.Suite, brand.Tool, brand.Author)
	fmt.Println("Uso:")
	fmt.Println("  meta-launcher              diagnostica y recomienda solución")
	fmt.Println("  meta-launcher --self-test  prueba la descarga (CP2)")
	fmt.Println("  meta-launcher --version    versión")
	fmt.Println("  meta-launcher --help       esta ayuda")
}
