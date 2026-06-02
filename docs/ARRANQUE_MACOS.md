# Arranque en macOS — Gatekeeper y alternativas de inicio

Guía para alumnos y profesores: cómo abrir el **meta-launcher** (y el
`BDPV6 Launcher.app`) en macOS cuando aparece el aviso de seguridad de Apple.

---

## ¿Por qué macOS bloquea el meta-launcher?

El binario `meta-launcher-macos-arm64` **no está firmado ni notarizado por
Apple** (no hay cuenta de Apple Developer). Cuando un ejecutable se descarga
**por navegador** (Safari/Chrome), macOS le pega el atributo
`com.apple.quarantine`, y **Gatekeeper** muestra:

> *"Apple no pudo verificar que «meta-launcher-macos-arm64» no contenga
> software malicioso…"*

No es malware: es la política de Apple para software sin firma. Hay que
autorizarlo **una sola vez**.

> 🔑 **Clave:** el bloqueo lo causa el atributo *quarantine*, que **solo añaden
> los navegadores**. Si descargas con `curl` desde la Terminal, **no aparece**.

---

## Alternativas de inicio (de mejor a peor)

### Opción 1 — Descargar con `curl` (sin diálogo) ✅ RECOMENDADA para alumnos
`curl` no marca quarantine, así que Gatekeeper no bloquea:

```bash
mkdir -p ~/BDP && cd ~/BDP        # o una carpeta en el USB si no hay espacio interno
curl -L -o meta-launcher-macos-arm64 \
  "https://github.com/abxda/bdp-meta-launcher/releases/download/v0.2.0/meta-launcher-macos-arm64"
chmod +x meta-launcher-macos-arm64
./meta-launcher-macos-arm64
```

### Opción 2 — Quitar quarantine en Terminal (si ya lo bajaste por navegador)
```bash
xattr -dr com.apple.quarantine /ruta/a/meta-launcher-macos-arm64
chmod +x /ruta/a/meta-launcher-macos-arm64
./meta-launcher-macos-arm64
```

### Opción 3 — Finder: clic derecho → Abrir
1. En Finder, **clic derecho** (o `Control`+clic) sobre el binario → **Abrir**.
2. En el aviso, pulsa **Abrir** otra vez.
(Para un binario de línea de comandos, la Opción 2 suele ser más cómoda.)

### Opción 4 — Ajustes del Sistema → "Abrir de todos modos"
1. Menú  (arriba a la izquierda) → **Ajustes del Sistema…**
2. Barra lateral → **Privacidad y seguridad**.
3. Baja hasta la sección **Seguridad**: aparece *"se bloqueó
   meta-launcher-macos-arm64…"* con el botón **"Abrir de todos modos"**.
4. Púlsalo, autentícate (Touch ID/contraseña) y vuelve a ejecutarlo.

> En macOS reciente el panel se llama **Ajustes del Sistema → Privacidad y
> seguridad** (antes era "Preferencias del Sistema → Seguridad y privacidad").

---

## Verificar integridad del binario (opcional pero recomendado)

```bash
shasum -a 256 meta-launcher-macos-arm64
# debe imprimir:
# b83771811f3871dc9b66c40e09fc092a417d7a28b31ca70088979447deefd07e
```

---

## El `BDPV6 Launcher.app` (lo que abre el meta-launcher)

Cuando el meta-launcher descarga y extrae el kit, la app **no lleva
quarantine** (la baja por HTTP y la extrae con `tar`, no por navegador), así
que normalmente **no dispara el diálogo de Gatekeeper**. Si la abres por tu
cuenta y se bloquea, aplica las mismas Opciones 2–4 sobre
`…/portable/BDPV6 Launcher.app`:

```bash
xattr -dr com.apple.quarantine "/ruta/a/portable/BDPV6 Launcher.app"
open "/ruta/a/portable/BDPV6 Launcher.app"
```

---

## Permiso de "volumen extraíble" (al correr desde USB)

Si ejecutas desde un USB/SSD externo, macOS puede pedir **acceso a archivos en
un volumen extraíble**. Pulsa **Permitir**: sin ese permiso, el launcher no
puede escribir en el USB y el auto-reparador (exec bits, wrappers de Python,
`libsqlite3.dylib`) no se aplica.

---

## Espacio en disco (importante)

La descarga (~2.5 GB) + extracción (~12 GB) van a la carpeta donde ejecutas el
meta-launcher. Si el **disco interno está casi lleno**, Elasticsearch cruza el
umbral del 95% y marca sus índices **read-only** (aunque el cluster figure
`green`). Ejecuta desde un volumen con espacio holgado (p. ej. el USB externo).

---

## La solución definitiva (futuro): notarización

Para eliminar por completo la fricción de Gatekeeper en alumnos remotos hay que
**firmar (Developer ID) + notarizar** el binario y el `.app` con `notarytool` y
`stapler` (requiere cuenta Apple Developer, ~99 USD/año). Mientras tanto, la
Opción 1 (`curl`) evita el diálogo sin costo.
