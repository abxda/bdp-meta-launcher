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

// DefaultManifestURL es el ÚNICO dato cableado en el binario: dónde vive el
// manifiesto. TODO lo demás (qué archivos, sus URLs, hashes, versiones) se
// define DENTRO del manifiesto, que es un archivo de texto editable en HF.
// Así, para cambiar cualquier URL de descarga NO hay que recompilar: editas
// el manifiesto. Y si algún día hay que mover hasta el manifiesto mismo, se
// puede apuntar con la variable de entorno BDP_MANIFEST_URL sin recompilar.
const DefaultManifestURL = "https://huggingface.co/datasets/abxda/bdp-lab/resolve/main/manifest.txt"

// ManifestURL devuelve la URL efectiva del manifiesto: el override de entorno
// BDP_MANIFEST_URL si está definido, o la URL por defecto.
func ManifestURL() string {
	if v := os.Getenv("BDP_MANIFEST_URL"); v != "" {
		return v
	}
	return DefaultManifestURL
}

// manifestBase devuelve la carpeta del manifiesto (todo hasta el último '/'),
// usada para resolver rutas relativas de archivos que no traigan URL absoluta.
func manifestBase() string {
	u := ManifestURL()
	if i := strings.LastIndexByte(u, '/'); i >= 0 {
		return u[:i]
	}
	return u
}

// Manifest mapea clave -> valor del manifest.txt.
type Manifest map[string]string

// FetchManifest baja y parsea el manifiesto (líneas "clave=valor"; # = comentario).
func FetchManifest() (Manifest, error) {
	body, err := httpGet(ManifestURL())
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
// "test"), los datos de una distribución desde el manifiesto.
type Entry struct {
	Base     string
	File     string // nombre de archivo local destino
	URL      string // URL absoluta de descarga (opcional; si vacía, se deriva)
	SHA256   string
	LaunchAt string // ruta relativa a ejecutar/abrir tras extraer (opcional)
}

// Resolve extrae la Entry desde un Manifest. Claves reconocidas por base:
//
//	<base>.file    nombre del archivo (obligatorio)
//	<base>.sha256  hash esperado (obligatorio)
//	<base>.url     URL absoluta de descarga (opcional) — permite alojar el
//	               archivo en CUALQUIER lugar (HF, otro CDN, tu servidor) sin
//	               recompilar; solo se edita el manifiesto. Si falta, la URL
//	               se deriva de la carpeta del manifiesto + .file.
//	<base>.launch  ruta a abrir/ejecutar tras extraer (opcional)
func (m Manifest) Resolve(base string) (Entry, error) {
	e := Entry{Base: base}
	e.File = m[base+".file"]
	e.URL = m[base+".url"]
	e.SHA256 = m[base+".sha256"]
	e.LaunchAt = m[base+".launch"]
	if e.File == "" || e.SHA256 == "" {
		return e, fmt.Errorf("el manifiesto no tiene .file/.sha256 para %q", base)
	}
	return e, nil
}

// downloadURL devuelve la URL efectiva: la absoluta del manifiesto si existe,
// o la derivada de la carpeta del manifiesto + el nombre de archivo.
func (e Entry) downloadURL() string {
	if e.URL != "" {
		return e.URL
	}
	return manifestBase() + "/" + e.File
}

// ProgressFn recibe (bytesDescargados, bytesTotales). total=0 si es desconocido.
type ProgressFn func(done, total int64)

// Download trae e.File a destPath mostrando progreso. Verifica el SHA-256 al
// terminar y borra el archivo si no coincide.
func Download(e Entry, destPath string, prog ProgressFn) error {
	url := e.downloadURL()
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
	return ExtractTarGzProgress(archivePath, destDir, nil)
}

// ExtractTarGzProgress es como ExtractTarGz pero, si onProgress != nil, lo
// invoca tras cada entrada con (archivos extraídos, bytes escritos) para que la
// UI muestre avance (la distro portable pesa ~GB y descomprimir tarda minutos).
//
// Maneja regular files, directorios y SYMLINKS (y hardlinks). Los symlinks son
// imprescindibles en las distros de macOS (bundles .jdk/.app, wrappers de
// python): si se omiten, el kit queda roto. En Windows crear un symlink puede
// requerir privilegios; si falla se omite ese enlace SIN abortar (los tars de
// Windows no usan symlinks, así que es seguro).
func ExtractTarGzProgress(archivePath, destDir string, onProgress func(files int, bytes int64)) error {
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
	var files int
	var bytes int64
	tick := func() {
		if onProgress != nil {
			onProgress(files, bytes)
		}
	}
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
			n, err := io.Copy(w, tr)
			w.Close()
			if err != nil {
				return err
			}
			bytes += n
			files++
			tick()
		case tar.TypeSymlink:
			if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
				return err
			}
			os.Remove(target) // por si quedó de una extracción previa
			if err := os.Symlink(hd.Linkname, target); err != nil {
				continue // Windows sin privilegios: omitir sin abortar
			}
			files++
			tick()
		case tar.TypeLink:
			os.Remove(target)
			if err := os.Link(filepath.Join(destDir, hd.Linkname), target); err != nil {
				continue
			}
			files++
			tick()
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
