# BDP Meta-Launcher

Punto **único de entrada** para el laboratorio de Big Data del Dr. Abel Coronado.
Un solo archivo (`meta-launcher.com`) que, gracias a
[Cosmopolitan](https://github.com/jart/cosmopolitan), corre nativo en
**Windows, Linux, macOS Intel y macOS Apple Silicon** sin recompilar.

El Meta-Launcher **diagnostica** el sistema operativo y la arquitectura, y según
eso decide qué solución del laboratorio aplica, la **descarga** (desde Hugging
Face), la descomprime y **lanza** la app correspondiente.

## Matriz de decisión

| Sistema | Arquitectura | Solución(es) |
|---------|--------------|--------------|
| Windows | x86-64 | **Portable** o **Vagrant** (el alumno elige) |
| Linux | x86-64 | **Vagrant** |
| macOS | Intel x86-64 | **Vagrant** |
| macOS | Apple Silicon (ARM64) | **Portable** |

- **Portable**: distribución autocontenida; se copian los binarios y se ejecuta
  el launcher. Sin virtualización. (Repos: `bdpv6-launcher`.)
- **Vagrant**: VM con VirtualBox + Vagrant y todo el stack preinstalado.
  (Repo: `vagrant-onboarding-panel`; caja `abxda/big-data-lab` en HCP Vagrant
  Registry.)

## Por qué Cosmopolitan

Un Meta-Launcher escrito en C y compilado con `cosmocc` produce un **Actually
Portable Executable (APE)**: un binario polyglot que el cargador de cada SO
reconoce como propio (empieza con `MZ` para Windows, es ELF para Linux, Mach-O
para macOS, e incluye una mitad ARM64 para Apple Silicon). Resultado: **un solo
build, un solo archivo, todos los sistemas**. No hace falta compilar por
plataforma ni mantener varios binarios del dispatcher.

## Compilar

```bash
# Toolchain (una vez):
mkdir -p ~/cosmocc && cd ~/cosmocc
curl -fsSLO https://cosmo.zip/pub/cosmocc/cosmocc.zip && unzip -o cosmocc.zip
export PATH="$HOME/cosmocc/bin:$PATH"

# Compilar:
cd bdp-meta-launcher
./build.sh          # -> meta-launcher.com
./meta-launcher.com # pruébalo
```

## Estado (checkpoints)

- [x] **CP1** — Diagnóstico de SO + arquitectura y matriz de decisión.
      Compila a APE single-file; probado en Windows x86-64 (muestra Portable +
      Vagrant). El binario incluye el slice ARM64 para Apple Silicon.
- [ ] **CP2** — Descarga remota desde Hugging Face con barra de progreso +
      verificación de hash + descompresión.
- [ ] **CP3** — Lanzamiento de la app correspondiente (portable o panel
      Vagrant) tras la descarga; persistencia para no re-descargar.
- [ ] **CP4** — Selección interactiva cuando hay 2 opciones (Windows) y modo
      no-interactivo (`--solution=portable|vagrant`).

## Validación pendiente por plataforma

El mismo `meta-launcher.com` debe ejecutarse en:
- [x] Windows x86-64
- [ ] Linux x86-64 (validar con un agente)
- [ ] macOS Intel (validar con un agente)
- [ ] macOS Apple Silicon (validar con un agente)

> Nota: el binario es universal, así que no se recompila por plataforma; solo
> se **ejecuta y verifica** la salida del diagnóstico en cada una.

Autoría: **Dr. Abel Coronado**.
