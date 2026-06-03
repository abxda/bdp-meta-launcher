# Arranque en Linux — permisos de ejecución, escritorio e hipervisor

Guía para alumnos y profesores: cómo abrir el **meta-launcher** en Linux
x86-64 y resolver los dos tropiezos típicos (permiso de ejecución y conflicto
de hipervisor con KVM). En Linux la edición es **Vagrant** (VirtualBox +
Vagrant), no la Portable.

---

## ¿Por qué Linux no "bloquea" como macOS?

Linux **no tiene Gatekeeper** ni el atributo `com.apple.quarantine`, así que no
verás el aviso de "software malicioso" del Mac. A cambio hay dos fricciones
propias:

1. **Permiso de ejecución.** Al descargar por navegador, el binario llega **sin
   el bit de ejecución** (`-x`). Hay que dárselo con `chmod +x` una sola vez.
2. **El panel es una app de escritorio** (WebKitGTK). Necesita una **sesión
   gráfica X11/Wayland**; no arranca en un servidor headless por SSH.

Y una tercera, ya con la VM: en equipos **AMD/Intel con KVM cargado**,
VirtualBox no puede tomar la virtualización por hardware y la VM no enciende.
Es el equivalente Linux del conflicto con Hyper-V en Windows (ver más abajo).

> 🔑 **Clave:** en Linux el "bloqueo" no es de seguridad del SO, es solo el bit
> `-x`. Si descargas con `curl` y haces `chmod +x`, listo.

---

## Alternativas de inicio (de mejor a peor)

### Opción 1 — Descargar con `curl` + `chmod +x` ✅ RECOMENDADA para alumnos

```bash
mkdir -p ~/BDP && cd ~/BDP        # usa un disco con espacio holgado (~15 GB libres)
curl -L -o meta-launcher-linux-amd64 \
  "https://github.com/abxda/bdp-meta-launcher/releases/download/v0.2.0/meta-launcher-linux-amd64"
chmod +x meta-launcher-linux-amd64
./meta-launcher-linux-amd64
```

### Opción 2 — Si ya lo bajaste por navegador (solo falta el permiso)

```bash
cd ~/Descargas                    # o donde haya quedado
chmod +x meta-launcher-linux-amd64
./meta-launcher-linux-amd64
```

### Opción 3 — Gestor de archivos (sin terminal)

1. Clic derecho sobre `meta-launcher-linux-amd64` → **Propiedades** → pestaña
   **Permisos** → marca **"Permitir ejecutar el archivo como un programa"**.
2. Doble clic → **Ejecutar** (en algunos escritorios: **Ejecutar en una
   terminal** para ver el progreso).

> Sugerencia: la Opción 1 es la más robusta porque deja el log visible en la
> terminal (útil para dar soporte remoto).

---

## Verificar integridad del binario (opcional pero recomendado)

```bash
sha256sum meta-launcher-linux-amd64
# debe imprimir:
# 2dd9dae2c00befc425550a084ee2a362582792b9a38a50a3f959a42aadfc6783
```

Comprobar la versión:

```bash
./meta-launcher-linux-amd64 --version
# Big Data Lab Meta-Launcher v0.2.0
```

---

## Requisitos previos en Linux: escritorio + VirtualBox + Vagrant

La edición Linux es la **vía Vagrant**, así que instalar **VirtualBox** y
**Vagrant** es parte del flujo (a diferencia de la Portable de macOS):

```bash
# Debian/Ubuntu (ejemplo)
sudo apt update
sudo apt install -y virtualbox vagrant
```

- Necesitas una **sesión de escritorio** (X11/Wayland) para que abra la GUI del
  panel.
- Espacio: la **caja** `abxda/big-data-lab` pesa ~4.4 GB y la VM crece a unos
  **~15 GB** en total. Ejecuta el launcher desde un volumen con espacio.

---

## Qué hace el meta-launcher (y qué abre)

Al ejecutarlo:

1. Detecta tu plataforma (`linux-amd64`) y lee el `manifest.txt` del dataset de
   Hugging Face **`abxda/bdp-lab`**.
2. Descarga **`bdp-vagrant-linux-amd64.tar.gz`**, **verifica su `sha256`** y lo
   descomprime.
3. Lanza el panel **`vagrant-onboarding-panel`** (la GUI guiada).

Desde el panel, el alumno sigue: **Diagnóstico → (VirtualBox / Vagrant ya
instalados) → Añadir la caja → Levantar VM → "Levantar todos los servicios" →
Mi laboratorio**. Al cerrar el panel, la VM se apaga limpiamente.

---

## El "Gatekeeper" de Linux: conflicto de hipervisor (KVM ↔ VirtualBox)

En equipos con **KVM** cargado (módulo `kvm_amd` en AMD, `kvm_intel` en Intel),
VirtualBox **no puede tomar la virtualización por hardware** y la VM aborta al
encender. En el registro del panel verás algo como:

```
VBoxManage: error: VirtualBox can't enable the AMD-V extension.
Please disable the KVM kernel extension... (VERR_SVM_IN_USE)
```

El panel **detecta este caso y te lo explica** (no da un falso "VM levantada").
Solución: libera la virtualización descargando el módulo de KVM **una vez**:

```bash
# AMD:
sudo modprobe -r kvm_amd kvm
# Intel:
sudo modprobe -r kvm_intel kvm
```

Luego vuelve al panel y pulsa **"Levantar VM"**. Para recuperar tus VMs de
KVM/libvirt cuando termines:

```bash
sudo modprobe kvm_amd     # (o kvm_intel)
```

> Cierra cualquier VM de **KVM/QEMU/libvirt** antes de descargar el módulo.
> VirtualBox funciona con o sin KVM cargado; el problema es que **ambos** no
> pueden usar VT-x/AMD-V a la vez.

---

## vboxdrv y Secure Boot

Si VirtualBox se queja de que **`vboxdrv` no está cargado**:

```bash
sudo /sbin/vboxconfig      # compila y carga el módulo
# o:
sudo modprobe vboxdrv
```

Con **Secure Boot activo**, el kernel puede rechazar el módulo sin firmar. Dos
caminos: **firmar** el módulo (MOK) o **desactivar Secure Boot** en la UEFI/BIOS.
El panel también reporta este caso con la indicación concreta.

---

## Espacio en disco (importante)

La caja (~4.4 GB) + el disco de la VM hacen un total de **~15 GB**. Ejecuta el
meta-launcher desde un volumen con espacio holgado. Si el disco se llena,
Elasticsearch puede cruzar el umbral del 95% y marcar índices **read-only**
(aunque el cluster figure `green`).

---

## La solución definitiva (futuro)

Para alumnos remotos, lo ideal es publicar el meta-launcher como **paquete
nativo** (`.deb` / `.rpm`) o **AppImage**, que ya traen permisos correctos y
metadatos de escritorio. Mientras tanto, la Opción 1 (`curl` + `chmod +x`)
arranca sin fricción y sin costo.
