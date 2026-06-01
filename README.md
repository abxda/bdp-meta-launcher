# BDP Meta-Launcher

Punto **único de entrada** de la suite **Big Data Lab** del Dr. Abel Coronado.
Diagnostica el sistema (SO + arquitectura), decide qué solución del laboratorio
aplica, la **descarga** desde Hugging Face, verifica su integridad (SHA-256), la
descomprime y la lanza.

## Matriz de decisión

| Sistema | Arquitectura | Solución(es) |
|---------|--------------|--------------|
| Windows | x86-64 | **Portable** o **Vagrant** (el alumno elige) |
| Linux | x86-64 | **Vagrant** |
| macOS | Intel x86-64 | **Vagrant** |
| macOS | Apple Silicon (ARM64) | **Portable** |

- **Portable**: distribución autocontenida; binarios + launcher, sin
  virtualización. (Repo `bdpv6-launcher`.)
- **Vagrant**: VM con VirtualBox + Vagrant y todo el stack preinstalado.
  (Repo `vagrant-onboarding-panel`; caja `abxda/big-data-lab`.)

## Arquitectura: Go nativo + Cosmopolitan (híbrido)

Hay **dos implementaciones equivalentes** del mismo Meta-Launcher:

| Implementación | Plataformas | Por qué |
|----------------|-------------|---------|
| **Go** (`main.go`, `internal/`) | **Windows** (y también Linux/Mac) | Binario nativo PE/ELF/Mach-O, **sin disparar antivirus**. Usa solo la stdlib (net/http, crypto/sha256, archive/tar): no necesita `curl`/`tar` externos. |
| **Cosmopolitan** (`cosmo/`) | **Linux y macOS** (single-file `.com`) | Un solo archivo universal donde no hay EDR de por medio. |

**¿Por qué no Cosmopolitan en Windows?** El `.com` de Cosmopolitan, para emular
`fork`/`exec` de POSIX en Windows, reasigna el proceso padre — un patrón que los
EDR corporativos (Cortex XDR, Defender ATP, CrowdStrike) marcan como
`parent_process_spoofing` / "comportamiento malicioso". Es un **falso positivo**,
pero en equipos administrados —justo el público objetivo— puede acabar en
cuarentena o reportarse a TI a nombre del alumno. La versión Go nativa no tiene
ese problema. En Linux/macOS no aplica, así que ahí el `.com` single-file es
ideal.

> Ambas comparten la **misma marca y look & feel** (paquete `internal/brand`
> en Go; constantes equivalentes en C), homologados con los launchers Wails de
> la suite: misma paleta, mismo tono en español, misma autoría.

## Compilar

### Go (recomendado para Windows)

```bash
go build -o meta-launcher.exe .      # Windows
GOOS=linux  go build -o meta-launcher-linux .
GOOS=darwin GOARCH=amd64 go build -o meta-launcher-mac-intel .
GOOS=darwin GOARCH=arm64 go build -o meta-launcher-mac-arm .
```

### Cosmopolitan (single-file para Linux/Mac)

```bash
# toolchain (una vez):
mkdir -p ~/cosmocc && cd ~/cosmocc
curl -fsSLO https://cosmo.zip/pub/cosmocc/cosmocc.zip && unzip -o cosmocc.zip
export PATH="$HOME/cosmocc/bin:$PATH"
# compilar:
cd cosmo && ./build.sh               # -> meta-launcher.com
```

## Probar la descarga

```bash
meta-launcher --self-test
```

Descarga el fixture `test` del dataset Hugging Face
[`abxda/bdp-lab`](https://huggingface.co/datasets/abxda/bdp-lab), verifica su
SHA-256 y lo descomprime en `~/.bigdatalab/`.

## Hospedaje de distribuciones

- **Caja Vagrant** (4.43 GB): HCP Vagrant Registry (`abxda/big-data-lab`).
- **Distros portables** y **manifiesto**: dataset público de Hugging Face
  [`abxda/bdp-lab`](https://huggingface.co/datasets/abxda/bdp-lab) — sin token
  para descargar (es público); el token solo se usa para subir.
- **Binarios del Meta-Launcher**: GitHub Releases (canónico) + copia en el
  dataset HF para que el alumno baje todo de un solo lugar.

El `manifest.txt` mapea cada combinación a su archivo y hash:

```
windows-amd64-portable.file=...
windows-amd64-portable.sha256=...
windows-amd64-portable.launch=...
```

## Estado (checkpoints)

- [x] **CP1** — Diagnóstico SO+arquitectura y matriz de decisión.
- [x] **CP2** — Descarga desde Hugging Face + verificación SHA-256 +
      descompresión. Validado en Windows (Go, sin dependencias externas, sin
      alerta de antivirus). Fixture `test` en el dataset HF.
- [ ] **CP3** — Lanzamiento de la app tras descargar; persistencia para no
      re-descargar; selección interactiva cuando hay 2 opciones.
- [ ] **Distros reales** — subir las distribuciones portables a HF y poblar el
      manifiesto con las entradas `<os>-<arch>-<solución>`.

## Validación por plataforma

| Plataforma | Go nativo | Cosmo `.com` |
|------------|-----------|--------------|
| Windows x86-64 | ✅ probado | ⚠️ EDR (no usar) |
| Linux x86-64 | ⏳ agente | ⏳ agente |
| macOS Intel | ⏳ agente | ⏳ agente |
| macOS Apple Silicon | ⏳ agente | ⏳ agente |

Autoría: **Dr. Abel Coronado**.
