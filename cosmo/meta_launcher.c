// BDP Meta-Launcher — punto único de entrada para "Big Data Lab"
// del Dr. Abel Coronado.
//
// Compilado con Cosmopolitan (cosmocc) produce UN SOLO archivo (Actually
// Portable Executable) que corre nativo en Windows, Linux, macOS Intel y
// macOS Apple Silicon. Diagnostica el entorno y, según SO + arquitectura,
// decide qué solución del laboratorio aplica, la DESCARGA (desde Hugging
// Face), verifica su integridad (SHA-256), la descomprime y lanza la app.
//
//     Windows  x86-64        -> Portable  o  Vagrant   (el alumno elige)
//     Linux    x86-64        -> Vagrant
//     macOS    Intel x86-64  -> Vagrant
//     macOS    Apple Silicon -> Portable
//
// CP1: diagnóstico + matriz de decisión.
// CP2 (este archivo): descarga desde Hugging Face + verificación SHA-256 +
//     descompresión. Validable con `meta-launcher.com --self-test`.
//
// Autoría: Dr. Abel Coronado.

#include <cosmo.h>
#include <stdio.h>
#include <stdlib.h>
#include <string.h>
#include <stdint.h>
#include <unistd.h>

// ============================ MARCA (look & feel) ===========================
// Homologada con el design system de los launchers Wails:
//   brand  #2563eb -> azul    success #16a34a -> verde
//   warn   #d97706 -> amarillo danger  #dc2626 -> rojo   muted -> dim
#define BRAND_SUITE  "Big Data Lab"
#define BRAND_SUB    "Laboratorio de Big Data"
#define BRAND_AUTHOR "Dr. Abel Coronado"

#define HF_BASE "https://huggingface.co/datasets/abxda/bdp-lab/resolve/main"

static int g_color;
static const char *C(const char *code) { return g_color ? code : ""; }
#define RESET  C("\033[0m")
#define BOLD   C("\033[1m")
#define DIM    C("\033[2m")
#define RED    C("\033[31m")
#define GREEN  C("\033[32m")
#define YELLOW C("\033[33m")
#define BLUE   C("\033[34m")
#define CYAN   C("\033[36m")

// ============================ SHA-256 (dominio público) =====================
typedef struct { uint32_t s[8]; uint64_t n; uint8_t b[64]; size_t c; } sha256_t;
static uint32_t ror(uint32_t x, int k){ return (x>>k)|(x<<(32-k)); }
static void sha256_blk(sha256_t *h, const uint8_t *p){
    static const uint32_t K[64]={
    0x428a2f98,0x71374491,0xb5c0fbcf,0xe9b5dba5,0x3956c25b,0x59f111f1,0x923f82a4,0xab1c5ed5,
    0xd807aa98,0x12835b01,0x243185be,0x550c7dc3,0x72be5d74,0x80deb1fe,0x9bdc06a7,0xc19bf174,
    0xe49b69c1,0xefbe4786,0x0fc19dc6,0x240ca1cc,0x2de92c6f,0x4a7484aa,0x5cb0a9dc,0x76f988da,
    0x983e5152,0xa831c66d,0xb00327c8,0xbf597fc7,0xc6e00bf3,0xd5a79147,0x06ca6351,0x14292967,
    0x27b70a85,0x2e1b2138,0x4d2c6dfc,0x53380d13,0x650a7354,0x766a0abb,0x81c2c92e,0x92722c85,
    0xa2bfe8a1,0xa81a664b,0xc24b8b70,0xc76c51a3,0xd192e819,0xd6990624,0xf40e3585,0x106aa070,
    0x19a4c116,0x1e376c08,0x2748774c,0x34b0bcb5,0x391c0cb3,0x4ed8aa4a,0x5b9cca4f,0x682e6ff3,
    0x748f82ee,0x78a5636f,0x84c87814,0x8cc70208,0x90befffa,0xa4506ceb,0xbef9a3f7,0xc67178f2};
    uint32_t w[64],a,b,c,d,e,f,g,hh,t1,t2; int i;
    for(i=0;i<16;i++) w[i]=(p[i*4]<<24)|(p[i*4+1]<<16)|(p[i*4+2]<<8)|p[i*4+3];
    for(i=16;i<64;i++){ uint32_t s0=ror(w[i-15],7)^ror(w[i-15],18)^(w[i-15]>>3);
        uint32_t s1=ror(w[i-2],17)^ror(w[i-2],19)^(w[i-2]>>10); w[i]=w[i-16]+s0+w[i-7]+s1; }
    a=h->s[0];b=h->s[1];c=h->s[2];d=h->s[3];e=h->s[4];f=h->s[5];g=h->s[6];hh=h->s[7];
    for(i=0;i<64;i++){ uint32_t S1=ror(e,6)^ror(e,11)^ror(e,25); uint32_t ch=(e&f)^((~e)&g);
        t1=hh+S1+ch+K[i]+w[i]; uint32_t S0=ror(a,2)^ror(a,13)^ror(a,22);
        uint32_t mj=(a&b)^(a&c)^(b&c); t2=S0+mj;
        hh=g;g=f;f=e;e=d+t1;d=c;c=b;b=a;a=t1+t2; }
    h->s[0]+=a;h->s[1]+=b;h->s[2]+=c;h->s[3]+=d;h->s[4]+=e;h->s[5]+=f;h->s[6]+=g;h->s[7]+=hh;
}
static void sha256_init(sha256_t *h){ h->s[0]=0x6a09e667;h->s[1]=0xbb67ae85;h->s[2]=0x3c6ef372;
    h->s[3]=0xa54ff53a;h->s[4]=0x510e527f;h->s[5]=0x9b05688c;h->s[6]=0x1f83d9ab;h->s[7]=0x5be0cd19;
    h->n=0;h->c=0; }
static void sha256_upd(sha256_t *h,const uint8_t *p,size_t n){ h->n+=n;
    while(n){ size_t k=64-h->c; if(k>n)k=n; memcpy(h->b+h->c,p,k); h->c+=k;p+=k;n-=k;
        if(h->c==64){ sha256_blk(h,h->b); h->c=0; } } }
static void sha256_fin(sha256_t *h,char *hex){ uint64_t bits=h->n*8; uint8_t pad=0x80;
    sha256_upd(h,&pad,1); pad=0; while(h->c!=56) sha256_upd(h,&pad,1);
    uint8_t L[8]; for(int i=0;i<8;i++)L[i]=(bits>>(56-i*8))&0xff; sha256_upd(h,L,8);
    static const char *hx="0123456789abcdef";
    for(int i=0;i<8;i++){ for(int j=0;j<4;j++){ uint8_t by=(h->s[i]>>(24-j*8))&0xff;
        *hex++=hx[by>>4]; *hex++=hx[by&0xf]; } } *hex=0; }
static int sha256_file(const char *path, char *hex64){
    FILE *f=fopen(path,"rb"); if(!f) return -1; sha256_t h; sha256_init(&h);
    uint8_t buf[65536]; size_t r; while((r=fread(buf,1,sizeof buf,f))>0) sha256_upd(&h,buf,r);
    fclose(f); sha256_fin(&h,hex64); return 0;
}

// ============================ utilidades ====================================
static void hr(void){ printf("%s",DIM); for(int i=0;i<64;i++) putchar('='); printf("%s\n",RESET); }
static void banner(void){
    hr();
    printf("  %s%s%s%s  %s- %s%s\n", BOLD,CYAN,BRAND_SUITE,RESET, DIM,BRAND_SUB,RESET);
    printf("  %sMeta-Launcher  ·  %s%s\n", DIM,BRAND_AUTHOR,RESET);
    hr();
}
static void step(const char *msg){ printf("  %s>%s %s\n", CYAN,RESET,msg); }
static void okmsg(const char *msg){ printf("  %s\xE2\x9C\x93%s %s\n", GREEN,RESET,msg); }
static void errmsg(const char *msg){ printf("  %s[!]%s %s\n", RED,RESET,msg); }

// run ejecuta un comando del sistema y devuelve su exit code.
static int run(const char *cmd){ int rc=system(cmd); return rc; }

// have_tool comprueba si una herramienta existe (curl/tar) de forma silenciosa.
// Cosmopolitan ejecuta system() con su PROPIO shell POSIX embebido en TODAS
// las plataformas (incluido Windows), así que usamos sintaxis POSIX siempre.
static int have_tool(const char *tool){
    char cmd[256];
    snprintf(cmd,sizeof cmd,"command -v %s >/dev/null 2>&1",tool);
    return run(cmd)==0;
}

// ============================ plataforma ====================================
typedef enum { OS_WINDOWS,OS_LINUX,OS_MACOS,OS_BSD,OS_UNKNOWN } os_t;
typedef enum { ARCH_X86_64,ARCH_ARM64,ARCH_UNKNOWN } arch_t;

static os_t detect_os(void){
    if(IsWindows()) return OS_WINDOWS;
    if(IsXnu())     return OS_MACOS;
    if(IsLinux())   return OS_LINUX;
    if(IsFreebsd()||IsOpenbsd()||IsNetbsd()) return OS_BSD;
    return OS_UNKNOWN;
}
static arch_t detect_arch(void){
#if defined(__aarch64__)
    return ARCH_ARM64;
#elif defined(__x86_64__)
    return ARCH_X86_64;
#else
    return ARCH_UNKNOWN;
#endif
}
static const char *os_name(os_t o){ switch(o){case OS_WINDOWS:return "Windows";case OS_LINUX:return "Linux";
    case OS_MACOS:return "macOS";case OS_BSD:return "BSD";default:return "desconocido";} }
static const char *arch_name(arch_t a){ switch(a){case ARCH_X86_64:return "x86-64 (Intel/AMD)";
    case ARCH_ARM64:return "ARM64 (Apple Silicon)";default:return "desconocida";} }

typedef enum { SOL_PORTABLE,SOL_VAGRANT } sol_t;
static const char *sol_name(sol_t s){ return s==SOL_PORTABLE?"Portable":"Vagrant"; }
static const char *sol_desc(sol_t s){ return s==SOL_PORTABLE
    ? "Distribucion autocontenida: copia los binarios y ejecuta el launcher. Sin virtualizacion."
    : "Maquina virtual (VirtualBox + Vagrant) con todo el stack preinstalado."; }
static int solutions_for(os_t os,arch_t arch,sol_t *out){ int n=0;
    switch(os){
        case OS_WINDOWS: if(arch==ARCH_X86_64){ out[n++]=SOL_PORTABLE; out[n++]=SOL_VAGRANT; } break;
        case OS_LINUX:   if(arch==ARCH_X86_64){ out[n++]=SOL_VAGRANT; } break;
        case OS_MACOS:   if(arch==ARCH_ARM64){ out[n++]=SOL_PORTABLE; }
                         else if(arch==ARCH_X86_64){ out[n++]=SOL_VAGRANT; } break;
        default: break;
    } return n;
}

// ============================ descarga / CP2 ================================
// Directorio de trabajo donde se descargan/extraen las soluciones.
static void workdir(char *out,size_t n){
    const char *home = getenv(IsWindows()?"USERPROFILE":"HOME");
    if(!home||!*home) home=".";
    snprintf(out,n,"%s/.bigdatalab",home);
}

// download trae una URL a un archivo destino mostrando barra de progreso.
static int download(const char *url,const char *dest){
    char cmd[2048];
    snprintf(cmd,sizeof cmd,"curl -fL --progress-bar -o \"%s\" \"%s\"",dest,url);
    return run(cmd);
}

// fetch_manifest_value baja el manifest y extrae el valor de una clave.
// Formato del manifest: lineas "clave=valor"; lineas con # se ignoran.
static int fetch_manifest_value(const char *workdir_path,const char *key,char *out,size_t n){
    char mpath[1024]; snprintf(mpath,sizeof mpath,"%s/manifest.txt",workdir_path);
    char url[1024];   snprintf(url,sizeof url,"%s/manifest.txt",HF_BASE);
    char cmd[2200];   snprintf(cmd,sizeof cmd,"curl -fsSL -o \"%s\" \"%s\"",mpath,url);
    if(run(cmd)!=0) return -1;
    FILE *f=fopen(mpath,"rb"); if(!f) return -1;
    char line[1024]; int found=-1; size_t klen=strlen(key);
    while(fgets(line,sizeof line,f)){
        if(line[0]=='#') continue;
        if(strncmp(line,key,klen)==0 && line[klen]=='='){
            char *v=line+klen+1; v[strcspn(v,"\r\n")]=0;
            snprintf(out,n,"%s",v); found=0; break;
        }
    }
    fclose(f); return found;
}

// extract descomprime un .tar.gz (o .tgz) en destdir usando tar.
static int extract_targz(const char *archive,const char *destdir){
    char cmd[2048];
    snprintf(cmd,sizeof cmd,"tar -xzf \"%s\" -C \"%s\"",archive,destdir);
    return run(cmd);
}

// prepare_entry ejecuta la cadena completa para una entrada del manifest
// (clave base, p.ej. "test" o "windows-x86_64-portable"):
//   manifest -> file/sha256 -> descargar -> verificar -> extraer.
static int prepare_entry(const char *base){
    char wd[1024]; workdir(wd,sizeof wd);
    char mk[1024];
    // crear el workdir (shell POSIX de Cosmopolitan en todas las plataformas)
    snprintf(mk,sizeof mk,"mkdir -p \"%s\"",wd);
    run(mk);

    char keyf[256],keys[256],file[512],sha_exp[128];
    snprintf(keyf,sizeof keyf,"%s.file",base);
    snprintf(keys,sizeof keys,"%s.sha256",base);

    step("Leyendo el manifiesto de distribuciones desde Hugging Face…");
    if(fetch_manifest_value(wd,keyf,file,sizeof file)!=0){ errmsg("No encontre el archivo en el manifiesto."); return 1; }
    if(fetch_manifest_value(wd,keys,sha_exp,sizeof sha_exp)!=0){ errmsg("No encontre el SHA-256 en el manifiesto."); return 1; }

    char url[1024],dest[1024];
    snprintf(url,sizeof url,"%s/%s",HF_BASE,file);
    snprintf(dest,sizeof dest,"%s/%s",wd,file);

    char msg[1200]; snprintf(msg,sizeof msg,"Descargando %s%s%s …",BOLD,file,RESET); step(msg);
    if(download(url,dest)!=0){ errmsg("Fallo la descarga."); return 1; }

    step("Verificando integridad (SHA-256)…");
    char got[65];
    if(sha256_file(dest,got)!=0){ errmsg("No pude leer el archivo descargado."); return 1; }
    if(strcmp(got,sha_exp)!=0){
        errmsg("El hash NO coincide — el archivo podria estar corrupto. Abortando.");
        printf("    %sesperado:%s %s\n    %sobtenido:%s %s\n",DIM,RESET,sha_exp,DIM,RESET,got);
        return 1;
    }
    okmsg("Integridad verificada.");

    char emsg[1200]; snprintf(emsg,sizeof emsg,"Descomprimiendo en %s%s%s …",BOLD,wd,RESET); step(emsg);
    if(extract_targz(dest,wd)!=0){ errmsg("Fallo la descompresion."); return 1; }
    okmsg("Listo. Solucion preparada en disco.");
    return 0;
}

// ============================ self-test (CP2) ===============================
static int self_test(void){
    banner();
    printf("\n  %sAuto-prueba CP2%s — descarga + verificacion + descompresion\n",BOLD,RESET);
    printf("  %sUsa el fixture 'test' del repo Hugging Face.%s\n\n",DIM,RESET);
    if(!have_tool("curl")){ errmsg("No encontre 'curl' (necesario para descargar)."); return 3; }
    if(!have_tool("tar")){  errmsg("No encontre 'tar' (necesario para descomprimir)."); return 3; }
    int rc=prepare_entry("test");
    if(rc==0){ printf("\n"); okmsg("Auto-prueba CP2 EXITOSA: la cadena de descarga funciona en esta plataforma."); printf("\n"); }
    else     { printf("\n"); errmsg("Auto-prueba CP2 fallo. Revisa el detalle arriba."); printf("\n"); }
    return rc;
}

// ============================ diagnóstico (CP1) =============================
static int diagnose(void){
    banner();
    os_t os=detect_os(); arch_t arch=detect_arch();
    printf("\n  %sDiagnostico del entorno%s\n",BOLD,RESET);
    printf("    Sistema operativo : %s%s%s\n",GREEN,os_name(os),RESET);
    printf("    Arquitectura      : %s%s%s\n",GREEN,arch_name(arch),RESET);

    sol_t sols[2]; int n=solutions_for(os,arch,sols);
    printf("\n  %sSolucion(es) recomendada(s) para tu equipo%s\n",BOLD,RESET);
    if(n==0){
        errmsg("Combinacion no soportada todavia.");
        printf("    %sEscribe al %s para soporte.%s\n",DIM,BRAND_AUTHOR,RESET);
        printf("\n"); return 2;
    }
    for(int i=0;i<n;i++){
        printf("    %s%d) %s%s%s\n",BOLD,i+1,BLUE,sol_name(sols[i]),RESET);
        printf("       %s%s%s\n",DIM,sol_desc(sols[i]),RESET);
    }
    if(n>1){
        printf("\n  %sTu equipo admite ambas:%s\n",BOLD,RESET);
        printf("    %s- Portable%s  maxima velocidad, sin instalar VirtualBox.\n",BLUE,RESET);
        printf("    %s- Vagrant%s   entorno aislado e identico para todos.\n",BLUE,RESET);
    }
    printf("\n  %sSiguiente:%s descarga + lanzamiento (CP3). Prueba la descarga con:\n",YELLOW,RESET);
    printf("    %smeta-launcher --self-test%s\n\n",DIM,RESET);
    return 0;
}

// ============================ main ==========================================
int main(int argc,char *argv[]){
    g_color=isatty(1);
    if(argc>1 && strcmp(argv[1],"--self-test")==0) return self_test();
    if(argc>1 && (strcmp(argv[1],"--help")==0||strcmp(argv[1],"-h")==0)){
        printf("%s — Meta-Launcher (%s)\n",BRAND_SUITE,BRAND_AUTHOR);
        printf("Uso:\n  meta-launcher              diagnostica y recomienda solucion\n");
        printf("  meta-launcher --self-test  prueba la descarga (CP2)\n");
        printf("  meta-launcher --help       esta ayuda\n");
        return 0;
    }
    return diagnose();
}
