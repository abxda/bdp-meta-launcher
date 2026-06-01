#!/bin/bash
# Compila el BDP Meta-Launcher a un Actually Portable Executable (APE) con
# Cosmopolitan. El resultado, meta-launcher.com, corre nativo en Windows,
# Linux, macOS Intel y macOS Apple Silicon — un solo archivo.
#
# Requiere el toolchain cosmocc. Si no lo tienes:
#   mkdir -p ~/cosmocc && cd ~/cosmocc
#   curl -fsSLO https://cosmo.zip/pub/cosmocc/cosmocc.zip && unzip -o cosmocc.zip
#   export PATH="$HOME/cosmocc/bin:$PATH"
#
# Uso:  ./build.sh   (con cosmocc en el PATH o en $COSMOCC/bin)
set -euo pipefail

CC="cosmocc"
if ! command -v "$CC" >/dev/null 2>&1; then
    for cand in "$HOME/cosmocc/bin/cosmocc" "/c/cosmocc/bin/cosmocc" "/opt/cosmocc/bin/cosmocc"; do
        [ -x "$cand" ] && CC="$cand" && break
    done
fi
if ! command -v "$CC" >/dev/null 2>&1 && [ ! -x "$CC" ]; then
    echo "ERROR: no encontré cosmocc. Instálalo (ver comentarios arriba)." >&2
    exit 1
fi

echo "Compilando con: $CC"
"$CC" -O2 -o meta-launcher.com meta_launcher.c
echo "OK -> meta-launcher.com ($(wc -c < meta-launcher.com) bytes)"
echo "Pruébalo:  ./meta-launcher.com"
