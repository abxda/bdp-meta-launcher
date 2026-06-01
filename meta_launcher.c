// BDP Meta-Launcher — punto único de entrada para el laboratorio de Big Data.
//
// Compilado con Cosmopolitan (cosmocc) produce UN SOLO archivo (.com /
// Actually Portable Executable) que corre nativo en Windows, Linux, macOS
// Intel y macOS Apple Silicon. Diagnostica el entorno y, según el sistema
// operativo y la arquitectura, decide qué solución del laboratorio aplica:
//
//     Windows  x86-64        -> Portable  o  Vagrant   (el alumno elige)
//     Linux    x86-64        -> Vagrant
//     macOS    Intel x86-64  -> Vagrant
//     macOS    Apple Silicon -> Portable
//
// CP1 (este archivo): SOLO diagnostica y muestra la matriz de decisión.
// La descarga remota (desde Hugging Face) y el lanzamiento de la app real
// llegan en checkpoints siguientes.
//
// Autoría: Dr. Abel Coronado.

#include <cosmo.h>
#include <stdio.h>
#include <string.h>
#include <unistd.h>

// ---- colores ANSI (se desactivan si la salida no es una terminal) ----------
static int g_color;
static const char *C(const char *code) { return g_color ? code : ""; }
#define RESET   C("\033[0m")
#define BOLD    C("\033[1m")
#define DIM     C("\033[2m")
#define RED     C("\033[31m")
#define GREEN   C("\033[32m")
#define YELLOW  C("\033[33m")
#define BLUE    C("\033[34m")
#define CYAN    C("\033[36m")

// ---- identidad de plataforma ----------------------------------------------

typedef enum { OS_WINDOWS, OS_LINUX, OS_MACOS, OS_BSD, OS_UNKNOWN } os_t;
typedef enum { ARCH_X86_64, ARCH_ARM64, ARCH_UNKNOWN } arch_t;

static os_t detect_os(void) {
    if (IsWindows()) return OS_WINDOWS;
    if (IsXnu())     return OS_MACOS;   // Darwin / macOS
    if (IsLinux())   return OS_LINUX;
    if (IsFreebsd() || IsOpenbsd() || IsNetbsd()) return OS_BSD;
    return OS_UNKNOWN;
}

// detect_arch usa las macros del compilador. En un APE "fat", cosmocc compila
// este archivo para x86-64 y para arm64 y empaqueta ambos; en tiempo de
// ejecución la CPU corre su mitad, donde la macro correspondiente está
// definida. Por eso esta detección en compile-time ES correcta en runtime.
static arch_t detect_arch(void) {
#if defined(__aarch64__)
    return ARCH_ARM64;
#elif defined(__x86_64__)
    return ARCH_X86_64;
#else
    return ARCH_UNKNOWN;
#endif
}

static const char *os_name(os_t os) {
    switch (os) {
        case OS_WINDOWS: return "Windows";
        case OS_LINUX:   return "Linux";
        case OS_MACOS:   return "macOS";
        case OS_BSD:     return "BSD";
        default:         return "desconocido";
    }
}

static const char *arch_name(arch_t a) {
    switch (a) {
        case ARCH_X86_64: return "x86-64 (Intel/AMD)";
        case ARCH_ARM64:  return "ARM64 (Apple Silicon)";
        default:          return "desconocida";
    }
}

// ---- soluciones del laboratorio -------------------------------------------

typedef enum { SOL_PORTABLE, SOL_VAGRANT } sol_t;

static const char *sol_name(sol_t s) {
    return s == SOL_PORTABLE ? "Portable" : "Vagrant";
}

static const char *sol_desc(sol_t s) {
    return s == SOL_PORTABLE
        ? "Distribucion autocontenida: copia los binarios y ejecuta el launcher. Sin virtualizacion."
        : "Maquina virtual (VirtualBox + Vagrant) con todo el stack preinstalado.";
}

// solutions_for llena `out` (capacidad >=2) con las soluciones aplicables a
// (os, arch) y devuelve cuantas hay. 0 = combinacion no soportada.
static int solutions_for(os_t os, arch_t arch, sol_t *out) {
    int n = 0;
    switch (os) {
        case OS_WINDOWS:
            if (arch == ARCH_X86_64) { out[n++] = SOL_PORTABLE; out[n++] = SOL_VAGRANT; }
            break;
        case OS_LINUX:
            if (arch == ARCH_X86_64) { out[n++] = SOL_VAGRANT; }
            break;
        case OS_MACOS:
            if (arch == ARCH_ARM64)  { out[n++] = SOL_PORTABLE; }   // Apple Silicon
            else if (arch == ARCH_X86_64) { out[n++] = SOL_VAGRANT; } // Mac Intel
            break;
        default:
            break;
    }
    return n;
}

// ---- presentacion ----------------------------------------------------------

static void hr(void) {
    printf("%s", DIM);
    for (int i = 0; i < 64; i++) putchar('=');
    printf("%s\n", RESET);
}

static void banner(void) {
    hr();
    printf("  %s%sBDP Meta-Launcher%s  %s- Laboratorio de Big Data%s\n",
           BOLD, CYAN, RESET, DIM, RESET);
    printf("  %sDr. Abel Coronado%s\n", DIM, RESET);
    hr();
}

int main(int argc, char *argv[]) {
    g_color = isatty(1);

    banner();

    os_t os = detect_os();
    arch_t arch = detect_arch();

    printf("\n  %sDiagnostico del entorno%s\n", BOLD, RESET);
    printf("    Sistema operativo : %s%s%s\n", GREEN, os_name(os), RESET);
    printf("    Arquitectura      : %s%s%s\n", GREEN, arch_name(arch), RESET);

    sol_t sols[2];
    int n = solutions_for(os, arch, sols);

    printf("\n  %sSolucion(es) recomendada(s) para tu equipo%s\n", BOLD, RESET);
    if (n == 0) {
        printf("    %s[!] Combinacion no soportada todavia: %s / %s%s\n",
               RED, os_name(os), arch_name(arch), RESET);
        printf("    %sEscribe al Dr. Abel Coronado para soporte.%s\n", DIM, RESET);
        printf("\n");
        return 2;
    }
    for (int i = 0; i < n; i++) {
        printf("    %s%d) %s%s%s  %s\n",
               BOLD, i + 1, BLUE, sol_name(sols[i]), RESET,
               n > 1 && i == 0 ? CYAN : "");
        printf("       %s%s%s\n", DIM, sol_desc(sols[i]), RESET);
    }

    if (n > 1) {
        printf("\n  %sTu equipo admite ambas. Recomendacion:%s\n", BOLD, RESET);
        printf("    %s- Portable%s si quieres maxima velocidad y no instalar VirtualBox.\n", BLUE, RESET);
        printf("    %s- Vagrant%s  si prefieres un entorno aislado e identico para todos.\n", BLUE, RESET);
    }

    printf("\n  %sCP1: por ahora solo diagnostico. La descarga automatica desde%s\n", YELLOW, RESET);
    printf("  %sHugging Face y el lanzamiento de la app llegan en el siguiente paso.%s\n", YELLOW, RESET);
    printf("\n");

    (void)argc; (void)argv;
    return 0;
}
