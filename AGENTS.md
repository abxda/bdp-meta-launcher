# AGENTS.md — Compilar y publicar el Meta-Launcher

Guía para agentes (Linux, macOS) que compilan y publican el **BDP
Meta-Launcher** de la suite *Big Data Lab* del Dr. Abel Coronado.

## Qué es

Punto único de entrada que diagnostica el sistema, descarga la solución que
aplique (desde Hugging Face) y la lanza. Dos implementaciones equivalentes:

- **Go** (`main.go`, `internal/`) — binario nativo por plataforma. **Úsalo en
  Windows** (el .com de Cosmopolitan dispara falsos positivos de EDR ahí).
- **Cosmopolitan** (`cosmo/`) — un solo `.com` para **Linux y macOS**.

## URLs externalizadas (NO recompilar por cambios de URL)

El binario solo cablea `DEFAULT_MANIFEST_URL` (dónde vive el manifiesto). Todo
lo demás está en el manifiesto (texto en HF). Para cambiar destinos:
- edita el manifiesto en `huggingface.co/datasets/abxda/bdp-lab`, **no** el código;
- o reapunta el manifiesto con `BDP_MANIFEST_URL=<url>` en el entorno.

Solo recompila si cambias **lógica**, no URLs.

## Compilar — Go (Linux nativo, macOS nativo)

Requiere Go 1.23+. Sin CGO (usa solo stdlib).

```bash
# Linux x86-64
GOOS=linux  GOARCH=amd64 CGO_ENABLED=0 go build -o meta-launcher-linux-amd64 .

# macOS Intel
GOOS=darwin GOARCH=amd64 CGO_ENABLED=0 go build -o meta-launcher-darwin-amd64 .

# macOS Apple Silicon
GOOS=darwin GOARCH=arm64 CGO_ENABLED=0 go build -o meta-launcher-darwin-arm64 .

# Verificar:
./meta-launcher-<tu-plataforma> --self-test    # debe decir "Auto-prueba CP2 EXITOSA"
```

## Compilar — Cosmopolitan (un solo .com para Linux + macOS)

```bash
# Toolchain (una vez):
mkdir -p ~/cosmocc && cd ~/cosmocc
curl -fsSLO https://cosmo.zip/pub/cosmocc/cosmocc.zip && unzip -o cosmocc.zip
export PATH="$HOME/cosmocc/bin:$PATH"

# Compilar:
cd cosmo && ./build.sh            # -> meta-launcher.com
./meta-launcher.com --self-test   # en Linux/macOS resuelve curl/tar por PATH
```

> Nota: en **Windows** el shell embebido de Cosmopolitan no resuelve
> `curl.exe`/`tar.exe` por PATH y los EDR marcan el .com; por eso en Windows
> usamos el binario Go. En Linux/macOS el .com funciona bien.

## Verificación por plataforma (qué reportar)

Ejecuta y pega la salida de:

```bash
./meta-launcher-<plataforma>            # diagnóstico: debe detectar tu SO+arch
./meta-launcher-<plataforma> --self-test # descarga el fixture y verifica hash
```

Esperado:
- **Linux x86-64** → recomienda *Vagrant*.
- **macOS Intel** → recomienda *Vagrant*.
- **macOS Apple Silicon** → recomienda *Portable*.
- self-test: "Auto-prueba CP2 EXITOSA … sin alertas de antivirus".

## Publicar binarios

1. Sube los binarios a **GitHub Releases** de este repo (`abxda/bdp-meta-launcher`).
2. (Opcional) Sube copia al dataset HF `abxda/bdp-lab` para que el alumno baje
   todo de un lugar. Necesitas el token HF (ver más abajo).

## Cómo recibir el token de Hugging Face (sin exponerlo en git)

El token **NUNCA** va en el repo. Opciones, en orden de preferencia:

1. **Variable de entorno** que el Dr. Coronado te pasa por fuera (chat privado,
   gestor de secretos): `export HF_TOKEN=hf_xxx` antes de subir.
2. **Archivo local fuera del repo**: `~/.bdp/hf_token` (chmod 600), que tú lees
   en el momento de publicar y borras después. Está en `.gitignore`.
3. Para **solo compilar y verificar** NO necesitas token: la descarga del
   fixture/distros es pública. El token solo hace falta para SUBIR a HF.

Tras usarlo, el Dr. Coronado **rota el token** en huggingface.co/settings/tokens.

## Reportar de vuelta

Abre un issue o PR en este repo con:
- Plataforma (uname -a) y arquitectura.
- Salida de `--self-test`.
- Qué solución recomendó el diagnóstico.
- Binarios subidos (enlaces de Release).

Autoría: **Dr. Abel Coronado**.
