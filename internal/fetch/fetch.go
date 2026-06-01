// Package fetch descarga, verifica y descomprime las distribuciones del
// laboratorio desde el repo público de Hugging Face. Usa SOLO la stdlib de Go
// (net/http, crypto/sha256, archive/tar, compress/gzip): no depende de curl ni
// tar externos, así el binario es autosuficiente y no dispara antivirus al
// lanzar procesos hijos.
package fetch

import (
	"archive/tar"
	"compress/gzip"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// HFBase es la raíz pública (sin token) del dataset en Hugging Face.
const HFBase = "https://huggingface.co/datasets/abxda/bdp-lab/resolve/main"

// Manifest mapea clave -> valor del archivo manifest.txt.
type Manifest map[string]string

// FetchManifest baja y parsea manifest.txt (líneas "clave=valor"; # = comentario).
func FetchManifest() (Manifest, error) {
	body, err := httpGet(HFBase + "/manifest.txt")
	if err != nil {
		return nil, err
	}
	defer body.Close()
	data, err := io.ReadAll(body)
	if err != nil {
		return nil, err
	}
	m := Manifest{}
	for _, line := range strings.Split(string(data), "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		if i := strings.IndexByte(line, '='); i > 0 {
			m[strings.TrimSpace(line[:i])] = strings.TrimSpace(line[i+1:])
		}
	}
	return m, nil
}

// Entry resuelve, para una clave base (p.ej. "windows-amd64-portable" o
// "test"), el archivo y el sha256 esperado del manifest.
type Entry struct {
	Base     string
	File     string
	SHA256   string
	LaunchAt string // ruta relativa a ejecutar/abrir tras extraer (opcional)
}

// Resolve extrae la Entry desde un Manifest.
func (m Manifest) Resolve(base string) (Entry, error) {
	e := Entry{Base: base}
	e.File = m[base+".file"]
	e.SHA256 = m[base+".sha256"]
	e.LaunchAt = m[base+".launch"]
	if e.File == "" || e.SHA256 == "" {
		return e, fmt.Errorf("el manifiesto no tiene .file/.sha256 para %q", base)
	}
	return e, nil
}

// ProgressFn recibe (bytesDescargados, bytesTotales). total=0 si es desconocido.
type ProgressFn func(done, total int64)

// Download trae e.File a destPath mostrando progreso. Verifica el SHA-256 al
// terminar y borra el archivo si no coincide.
func Download(e Entry, destPath string, prog ProgressFn) error {
	url := HFBase + "/" + e.File
	body, total, err := httpGetLen(url)
	if err != nil {
		return err
	}
	defer body.Close()

	if err := os.MkdirAll(filepath.Dir(destPath), 0o755); err != nil {
		return err
	}
	out, err := os.Create(destPath)
	if err != nil {
		return err
	}

	h := sha256.New()
	pw := &progWriter{total: total, prog: prog, last: time.Now()}
	// Escribe simultáneamente al archivo, al hash y al contador de progreso.
	mw := io.MultiWriter(out, h, pw)
	_, copyErr := io.Copy(mw, body)
	closeErr := out.Close()
	if copyErr != nil {
		return copyErr
	}
	if closeErr != nil {
		return closeErr
	}
	if prog != nil {
		prog(pw.done, total) // asegura el 100% final
	}

	got := hex.EncodeToString(h.Sum(nil))
	if !strings.EqualFold(got, e.SHA256) {
		os.Remove(destPath)
		return fmt.Errorf("hash no coincide: esperado %s, obtenido %s", e.SHA256, got)
	}
	return nil
}

// ExtractTarGz descomprime un .tar.gz en destDir (creándolo). Protege contra
// path traversal (entradas con .. o rutas absolutas).
func ExtractTarGz(archivePath, destDir string) error {
	f, err := os.Open(archivePath)
	if err != nil {
		return err
	}
	defer f.Close()
	gz, err := gzip.NewReader(f)
	if err != nil {
		return err
	}
	defer gz.Close()
	tr := tar.NewReader(gz)

	if err := os.MkdirAll(destDir, 0o755); err != nil {
		return err
	}
	cleanDest := filepath.Clean(destDir)
	for {
		hd, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}
		target := filepath.Join(destDir, hd.Name)
		// Anti path-traversal: el target resuelto debe quedar dentro de destDir.
		if !strings.HasPrefix(filepath.Clean(target), cleanDest+string(os.PathSeparator)) &&
			filepath.Clean(target) != cleanDest {
			return fmt.Errorf("entrada insegura en el tar: %q", hd.Name)
		}
		switch hd.Typeflag {
		case tar.TypeDir:
			if err := os.MkdirAll(target, 0o755); err != nil {
				return err
			}
		case tar.TypeReg:
			if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
				return err
			}
			w, err := os.OpenFile(target, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, os.FileMode(hd.Mode)&0o777)
			if err != nil {
				return err
			}
			if _, err := io.Copy(w, tr); err != nil {
				w.Close()
				return err
			}
			w.Close()
		}
	}
	return nil
}

// --- helpers HTTP ---

func httpGet(url string) (io.ReadCloser, error) {
	rc, _, err := httpGetLen(url)
	return rc, err
}

func httpGetLen(url string) (io.ReadCloser, int64, error) {
	req, _ := http.NewRequest("GET", url, nil)
	req.Header.Set("User-Agent", "bdp-meta-launcher")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, 0, err
	}
	if resp.StatusCode != http.StatusOK {
		resp.Body.Close()
		return nil, 0, fmt.Errorf("HTTP %d al pedir %s", resp.StatusCode, url)
	}
	return resp.Body, resp.ContentLength, nil
}

// progWriter cuenta bytes y llama a prog con throttling (~10 Hz).
type progWriter struct {
	done  int64
	total int64
	prog  ProgressFn
	last  time.Time
}

func (p *progWriter) Write(b []byte) (int, error) {
	n := len(b)
	p.done += int64(n)
	if p.prog != nil && time.Since(p.last) > 100*time.Millisecond {
		p.prog(p.done, p.total)
		p.last = time.Now()
	}
	return n, nil
}
