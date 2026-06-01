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
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/abxda/bdp-meta-launcher/internal/brand"
	"github.com/abxda/bdp-meta-launcher/internal/fetch"
	"github.com/abxda/bdp-meta-launcher/internal/platform"
)

const version = "0.2.0"

func main() {
	brand.Init()
	args := os.Args[1:]
	switch {
	case len(args) > 0 && args[0] == "--self-test":
		os.Exit(selfTest())
	case len(args) > 0 && (args[0] == "--help" || args[0] == "-h"):
		help()
	case len(args) > 0 && args[0] == "--version":
		fmt.Printf("%s %s v%s\n", brand.Suite, brand.Tool, version)
	default:
		os.Exit(diagnose())
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
	fmt.Printf("\n  %sDiagnóstico del entorno%s\n", brand.Bold, brand.Reset)
	fmt.Printf("    Sistema operativo : %s%s%s\n", brand.Green, in.OSLabel, brand.Reset)
	fmt.Printf("    Arquitectura      : %s%s%s\n", brand.Green, in.ArchLabel, brand.Reset)

	fmt.Printf("\n  %sSolución(es) recomendada(s) para tu equipo%s\n", brand.Bold, brand.Reset)
	if len(in.Solutions) == 0 {
		bad("Combinación no soportada todavía.")
		fmt.Printf("    %sEscribe al %s para soporte.%s\n\n", brand.Dim, brand.Author, brand.Reset)
		return 2
	}
	for i, s := range in.Solutions {
		fmt.Printf("    %s%d) %s%s%s\n", brand.Bold, i+1, brand.Blue, platform.SolutionLabel(s), brand.Reset)
		fmt.Printf("       %s%s%s\n", brand.Dim, platform.SolutionDesc(s), brand.Reset)
	}
	if len(in.Solutions) > 1 {
		fmt.Printf("\n  %sTu equipo admite ambas:%s\n", brand.Bold, brand.Reset)
		fmt.Printf("    %s- Portable%s  máxima velocidad, sin instalar VirtualBox.\n", brand.Blue, brand.Reset)
		fmt.Printf("    %s- Vagrant%s   entorno aislado e idéntico para todos.\n", brand.Blue, brand.Reset)
	}
	fmt.Printf("\n  %sSiguiente:%s descarga + lanzamiento (CP3). Prueba la descarga con:\n", brand.Yellow, brand.Reset)
	fmt.Printf("    %smeta-launcher --self-test%s\n\n", brand.Dim, brand.Reset)
	return 0
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
	if err := fetch.ExtractTarGz(dest, wd); err != nil {
		bad("Falló la descompresión: " + err.Error())
		return 1
	}
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
