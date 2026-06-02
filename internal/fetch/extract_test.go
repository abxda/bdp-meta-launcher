package fetch

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

// TestExtractTarGzProgress verifica los tres puntos que importan para el kit
// portable de macOS: (1) los bits de ejecución se conservan, (2) los SYMLINKS
// se recrean (en plataformas que los soportan) — sin esto el .app/JDK quedan
// rotos —, y (3) el callback de progreso se invoca.
func TestExtractTarGzProgress(t *testing.T) {
	// Construir un tar.gz en memoria: dir + archivo ejecutable + symlink.
	var buf bytes.Buffer
	gz := gzip.NewWriter(&buf)
	tw := tar.NewWriter(gz)
	must := func(err error) {
		if err != nil {
			t.Fatal(err)
		}
	}
	must(tw.WriteHeader(&tar.Header{Name: "bin/", Typeflag: tar.TypeDir, Mode: 0o755}))
	script := []byte("#!/bin/sh\necho hola\n")
	must(tw.WriteHeader(&tar.Header{Name: "bin/run.sh", Typeflag: tar.TypeReg, Mode: 0o755, Size: int64(len(script))}))
	_, err := tw.Write(script)
	must(err)
	must(tw.WriteHeader(&tar.Header{Name: "bin/current", Typeflag: tar.TypeSymlink, Linkname: "run.sh", Mode: 0o777}))
	must(tw.Close())
	must(gz.Close())

	dst := t.TempDir()
	arch := filepath.Join(t.TempDir(), "k.tar.gz")
	must(os.WriteFile(arch, buf.Bytes(), 0o644))

	var calls int
	if err := ExtractTarGzProgress(arch, dst, func(files int, bytes int64) { calls = files }); err != nil {
		t.Fatalf("extract: %v", err)
	}
	if calls == 0 {
		t.Fatal("el callback de progreso nunca se invocó")
	}

	// (1) exec bit del archivo regular (en Windows el modo no aplica igual).
	fi, err := os.Stat(filepath.Join(dst, "bin", "run.sh"))
	must(err)
	if runtime.GOOS != "windows" && fi.Mode().Perm()&0o111 == 0 {
		t.Fatalf("se perdió el bit de ejecución: %v", fi.Mode())
	}

	// (2) symlink recreado (en SO que lo soportan sin privilegios).
	if runtime.GOOS != "windows" {
		li, err := os.Lstat(filepath.Join(dst, "bin", "current"))
		if err != nil {
			t.Fatalf("symlink no recreado: %v", err)
		}
		if li.Mode()&os.ModeSymlink == 0 {
			t.Fatal("la entrada 'current' no quedó como symlink")
		}
	}
}
