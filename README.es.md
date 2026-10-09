# Nimbus Backup — cliente de Windows para Proxmox Backup Server

[🇬🇧 English](README.md) · [🇫🇷 Français](README.fr.md) · [🇮🇹 Italiano](README.it.md) · [🇩🇪 Deutsch](README.de.md) · 🇪🇸 Español · [🇷🇺 Русский](README.ru.md) · [🇨🇳 中文](README.zh.md) · [🇯🇵 日本語](README.ja.md) · [🇬🇷 Ελληνικά](README.el.md) · [🇷🇴 Română](README.ro.md) · [🇸🇪 Svenska](README.sv.md) · [🇸🇦 العربية](README.ar.md) · [🇮🇷 فارسی](README.fa.md)

> 🤖 Esta traducción ha sido generada por una IA a partir del README en inglés. En caso de discrepancia, prevalece la [versión en inglés](README.md).
> *AI-generated translation of the English README; the [English version](README.md) is authoritative.*

[![License](https://img.shields.io/badge/license-GPLv3-blue.svg)](LICENSE)
[![Release](https://img.shields.io/github/v/release/rdemsystems/NimbusBackupClient)](https://github.com/rdemsystems/NimbusBackupClient/releases)
[![Documentation](https://img.shields.io/badge/docs-nimbus.rdem--systems.com-orange)](https://nimbus.rdem-systems.com/en/?utm_source=github)

**Nimbus Backup es un cliente de copias de seguridad para Windows de código abierto (GPL-3.0) para Proxmox Backup Server (PBS).**
Una interfaz gráfica moderna para respaldar servidores y estaciones de trabajo Windows en PBS — snapshots coherentes mediante VSS, tareas programadas, modos archivo y disco, exploración y restauración de snapshots, soporte multi-PBS y un servicio de Windows — además de herramientas de línea de comandos para copias de directorios y de máquinas completas. ¿Busca almacenamiento PBS **externo e inmutable** sin alojarlo usted mismo? Consulte el [servicio gestionado](#️-pbs-gestionado-externo-e-inmutable) más abajo.

Nimbus Backup es la compilación de RDEM Systems de [Proxmox Backup Client GO](https://github.com/tizbac/proxmoxbackupclient_go): la interfaz gráfica que desarrollamos se ha integrado en el proyecto upstream, y ambos proyectos comparten ahora la misma base de código (véase [Relación con el proyecto upstream](#-relación-con-el-proyecto-upstream)).

📖 **Documentación completa, guía de instalación y alojamiento PBS:** [nimbus.rdem-systems.com](https://nimbus.rdem-systems.com/en/blog/backup-windows-proxmox-backup-server/?utm_source=github&utm_medium=readme&utm_campaign=nbc-readme-top)

## 📦 Descarga

👉 **[Descargar la última versión](https://github.com/rdemsystems/NimbusBackupClient/releases)**

Cada versión incluye:
- `NimbusBackup.msi` — instalador (GUI + servicio de Windows), **recomendado para producción**
- `NimbusBackup.exe` — GUI independiente
- `nimbus-backup-cli-<version>-windows.zip` / `-linux.tar.gz` / `-macos.tar.gz` — herramientas de línea de comandos
- `SHA256SUMS.txt` — sumas de verificación

> ⚠️ **¿Windows indica "virus detectado" (p. ej. `Trojan:Win32/Sabsik.FL.A!ml`) o muestra una advertencia de SmartScreen?**
> Se trata de un **falso positivo** conocido en aplicaciones Go/Wails — *no* es un virus. El sufijo `!ml` significa que proviene de un modelo de aprendizaje automático que marca los ejecutables *sin firmar y poco extendidos*.
> Lea [por qué ocurre y cómo verificar la descarga](https://nimbus.rdem-systems.com/en/antivirus-false-positive/?utm_source=github).

### 🔎 Verificar cualquier descarga

Cada versión incluye sumas de verificación SHA-256 y una **atestación de procedencia de la compilación** firmada (prueba criptográfica de que el binario fue generado por la CI de este repositorio, a partir de un commit concreto):

```powershell
Get-FileHash .\NimbusBackup.msi -Algorithm SHA256   # compare against SHA256SUMS.txt
gh attestation verify .\NimbusBackup.msi --repo rdemsystems/NimbusBackupClient
```

**VirusTotal.** Las notas de cada versión enlazan al informe de VirusTotal del instalador de esa compilación (publicado solo si el análisis está limpio). Informes anteriores, 0 detecciones:
[0.2.108](https://www.virustotal.com/gui/file/6e8fb7ce9af740d470e947addb8daba4331c0b88e8bfdec9e0697ea8f7f29e9e/detection) ·
[0.2.107](https://www.virustotal.com/gui/file/6fd6c6fa77e0305c129ef882a3745100aa6033187a6d52a4af94149ab6b666d2/detection) ·
[0.2.106](https://www.virustotal.com/gui/file/ad6e56700ed9df8e088906e38cee2e2882fc7045f4e39269de0e379a01784ad7/detection)

> ℹ️ **Firma de código:** los binarios de Windows **aún no están firmados con Authenticode**, que es lo que provoca las advertencias de SmartScreen / `!ml` mencionadas arriba. Nuestra solicitud de un certificado OSS gratuito a la [SignPath Foundation](https://signpath.org) no obtuvo respuesta; la firma mediante Azure Artifact Signing se está configurando y está prevista para la **0.4.1**. Hasta entonces, la procedencia se acredita mediante la atestación de procedencia de la compilación y las sumas de verificación indicadas arriba.

### 🐧 ¿En Linux? Use el cliente oficial

La GUI de Nimbus Backup solo está disponible para Windows (las herramientas CLI también se compilan para Linux y macOS). Para copias a nivel de archivo en Linux, use el `proxmox-backup-client` de Proxmox — lo empaquetamos para las distribuciones que Proxmox no cubre:

👉 **[rdemsystems/unofficial-proxmox-backup-client](https://github.com/rdemsystems/unofficial-proxmox-backup-client)** — repositorios de paquetes firmados para Debian/Ubuntu, Fedora/RHEL/Rocky/AlmaLinux, Arch y Alpine (amd64 y arm64). El binario estático oficial de Proxmox, reempaquetado sin cambios — ni parcheado ni recompilado.

## ☁️ PBS gestionado (externo e inmutable)

¿No quiere alojar Proxmox Backup Server usted mismo? Use nuestros datastores PBS totalmente gestionados, **externos e inmutables**:
👉 **[Configure su copia de seguridad y consulte los precios](https://nimbus.rdem-systems.com/en/choose-backup/?utm_source=github)**

- ✅ Desde 12 €/TB/mes
- ✅ Prueba gratuita de 1 TB
- ✅ Un [destino externo para su PBS](https://nimbus.rdem-systems.com/en/offsite-proxmox-backup/?utm_source=github) listo para usar — datastores aislados (air-gapped) e inmutables
- ✅ [NimbusBackup — Alojamiento PBS gestionado en Francia](https://nimbus.rdem-systems.com/en/?utm_source=github)

## 📚 Documentación

- **Guía completa de Proxmox Backup** — buenas prácticas de despliegue de PBS ([🇬🇧 EN](https://nimbus.rdem-systems.com/en/blog/complete-proxmox-backup-guide/?utm_source=github))
- **Copia de seguridad de Windows con Proxmox Backup Server** — guía de despliegue específica para Windows ([🇬🇧 EN](https://nimbus.rdem-systems.com/en/blog/backup-windows-proxmox-backup-server/?utm_source=github))
- **PBS frente a Veeam** — comparativa con Proxmox Backup Server ([🇬🇧 EN](https://nimbus.rdem-systems.com/en/blog/pbs-vs-veeam-proxmox-backup-comparison/?utm_source=github))
- En este repositorio: [guía de usuario multi-PBS](MULTI_PBS_USER_GUIDE.md) · [ejemplos de despliegue desatendido](examples/automation/) · [restauración bare-metal con Clonezilla](PATCH-CLONEZILLA.md) · [registro de cambios](CHANGELOG.md)

## ✨ Funcionalidades

### GUI (recomendada)
- **🌍 Multilingüe** — interfaz en inglés, francés, italiano, alemán y polaco
- Configuración sencilla con prueba de conexión (token de API o usuario/contraseña)
- Progreso de la copia en tiempo real con velocidad y tiempo estimado, cancelable en cualquier momento
- Soporte de VSS (Volume Shadow Copy) para copias coherentes
- Copia de varias carpetas, modos archivo y disco (máquina completa); las carpetas pueden procesarse en paralelo (recomendado: CPU / 4)
- Exploración de snapshots, búsqueda de archivos (comodines) y restauración
- Soporte de varios servidores PBS, fijación de la huella del certificado (TOFU)
- **🔒 Cifrado del lado del cliente** (AES-256-GCM), archivos de clave compatibles con `proxmox-backup-client` y Proxmox VE
- Modo servicio de Windows + copias programadas, historial de copias con reejecución en un clic
- Registro de depuración para la resolución de problemas

### Herramientas de línea de comandos
- `proxmoxbackup-directory` — copias de directorios (PXAR) con deduplicación, copias de flujos (`-backupstream`, p. ej. una tubería `mysqldump`), exclusiones (`-exclude "*.tmp"`, repetible, o `-exclude-from file`; `"exclude"` en la configuración JSON), varios directorios a la vez (`-parallel N`, recomendado: CPU / 4), notificaciones por correo electrónico, archivo de configuración JSON
- `proxmoxbackup-machine` — copias completas en caliente de la máquina como imagen de disco arrancable (FIDX): VSS en Windows, incremental, hashing en paralelo
- `proxmoxbackup-nbd` — servidor NBD para montar una copia de disco en Linux (restauración a nivel de archivo, restauración bare-metal desde una [ISO live de Clonezilla parcheada](PATCH-CLONEZILLA.md))

### 📸 Capturas de pantalla

![Server configuration](docs/screenshots/nimbus-gui-liste-servers.png)
*Gestión de varios servidores PBS con indicadores de estado*

![Add server form](docs/screenshots/nimbus-gui-add-server-form.png)
*Configuración sencilla del servidor con prueba de conexión*

![One-shot backup](docs/screenshots/nimbus-gui-one-shot-backup.png)
*Progreso de la copia en tiempo real con tiempo estimado y velocidad*

### Exclusiones de sistema inteligentes (modo archivo)
Al respaldar una unidad completa (p. ej. `D:\`), Nimbus Backup excluye automáticamente:

**Carpetas del sistema:** `System Volume Information` (almacenamiento de VSS, puede superar los 100 GB), `$RECYCLE.BIN`, `Recovery`.
**Archivos del sistema:** `pagefile.sys`, `hiberfil.sys`, `swapfile.sys`.

**Por qué importa:** una unidad puede indicar 1,03 TB ocupados mientras que los archivos reales suman ~141 GB. Sin exclusiones, la copia incluiría los snapshots de VSS (espacio y tiempo desperdiciados); con ellas, el tamaño de la copia corresponde a los datos reales.

**Recomendación:** use el **modo archivo** (predeterminado) con exclusiones automáticas para las copias a nivel de archivo; use el **modo disco** en una tarea independiente para la restauración bare-metal (lo incluye todo).

### Seguridad y calidad
- Validación de entradas y saneamiento de credenciales (secretos ocultados en los registros)
- Prevención de path traversal
- Lógica de reintentos con espera exponencial
- Controles de CI en cada compilación: pruebas, `golangci-lint`, `gosec`, `go mod tidy`

### 🔒 Cifrado del lado del cliente
Las copias pueden cifrarse **en el cliente** antes de salir de la máquina, con
el mismo esquema que el `proxmox-backup-client` oficial (AES-256-GCM, resúmenes de chunks
con clave). El servidor PBS solo almacena datos opacos y nunca ve
la clave — útil en un PBS compartido o gestionado.

El cifrado fue desarrollado en el proyecto original por Tiziano Bacocco ([tizbac/proxmoxbackupclient_go](https://github.com/tizbac/proxmoxbackupclient_go)); la clave en papel, la exportación en código QR y la importación de claves son añadidos de Nimbus Backup.

- **GUI**: *Servidores → Editar → Clave de cifrado* — cree un archivo de clave o elija uno
  existente (de `proxmox-backup-client key create --kdf none` o de un almacenamiento
  PVE); se muestra su huella. Después, guarde una copia fuera de la máquina:
  **Imprimir (clave en papel)** guarda una página imprimible con la clave y su código QR (el
  formato de `proxmox-backup-client key paperkey`), opcionalmente protegida con frase de contraseña.
  **Importar una clave desde texto o código QR** convierte un código QR escaneado o una clave en papel
  de nuevo en un archivo de clave. La GUI solo usa archivos de clave sin protección: al importar una
  clave protegida con frase de contraseña, esta se desbloquea y el nuevo archivo de clave se guarda **sin**
  frase de contraseña — guarde ese archivo con el mismo cuidado que la propia clave.
- **CLI**: `-keyfile path/to/key.json` (o `"keyfile"` en la configuración JSON); una
  clave protegida con frase de contraseña requiere `-keyfile-passphrase`, o la solicita de forma interactiva.
- **Restauración bare-metal**: la ISO de Clonezilla parcheada también restaura copias de disco
  cifradas (clave desde una memoria USB, con su frase de contraseña si la tiene) — véase
  [PATCH-CLONEZILLA.md](PATCH-CLONEZILLA.md).
- **Compatible con `proxmox-backup-client`, verificado en la CI en cada compilación**
  contra un PBS real: una carpeta cifrada por Nimbus Backup con una clave creada por
  `proxmox-backup-client` es restaurada por el cliente oficial con la misma clave
  (y rechazada sin ella), y una copia de disco cifrada creada por el cliente
  oficial es leída por Nimbus Backup. Proxmox VE usa los mismos archivos de clave.

> ⚠️ **Sin la clave, las copias cifradas son irrecuperables.** La primera
> copia cifrada vuelve a subirlo todo (sin deduplicación con las copias
> no cifradas). Los ID de copia, los nombres de archivo de las copias y los tamaños siguen siendo visibles para el servidor; el
> contenido de los archivos, sus nombres y el catálogo están cifrados.

## 🤖 Despliegue desatendido (Ansible e IaC)

Nimbus Backup se instala y se configura íntegramente mediante archivos — sin configuración interactiva.
Dos vías, ambas cubiertas por ejemplos listos para usar en
[`examples/automation/`](examples/automation/):

- **Línea de comandos** — un archivo JSON por host contiene toda la copia
  (`proxmoxbackup-directory.exe --config file.json`); programación mediante el Programador de tareas
  de Windows. Códigos de salida fiables (`0` OK, `1` error fatal, `2` bloqueado, `3` parcial)
  para que su orquestador detecte los fallos. Ideal para infraestructura como código pura.
- **GUI/servicio (MSI)** — un **único `config.json`** contiene la conexión PBS,
  los ajustes de copia **y** la programación (`scheduled_jobs`). Distribuya el archivo, reinicie
  el servicio `NimbusBackup`: las tareas se concilian de forma idempotente y `nextRun` se
  calcula automáticamente. Sin cálculos de marcas de tiempo en su plantilla Jinja2.

El servicio lee su configuración desde `C:\ProgramData\ProxmoxBackupClient\`
(desde la 0.4.0; véase [Actualización](#️-actualizar-desde--030)).

📖 Guía completa: [Automate Windows backup to PBS with Ansible](https://nimbus.rdem-systems.com/en/blog/unattended-windows-backup-ansible/?utm_source=github).

## 🚀 Inicio rápido

1. Descargue `NimbusBackup.msi` (o el `NimbusBackup.exe` independiente) desde las versiones publicadas
2. Instálelo / ejecútelo con privilegios de administrador (necesarios para VSS)
3. Configure la conexión a su PBS y pruébela
4. Seleccione los directorios que desea respaldar
5. Inicie la copia — o prográmela

### 🔑 Usuario PBS y permisos

El cliente solo necesita el rol **`DatastoreBackup`** en el datastore de destino — ninguna cuenta de administrador:

1. En la interfaz de PBS, cree un usuario (p. ej. `nimbus@pbs`) y un token de API para él (p. ej. `nimbus@pbs!laptop01`).
2. En **Datastore → Permissions** (o **Configuration → Access Control → Permissions**), conceda `DatastoreBackup` sobre `/datastore/<name>` — o sobre `/datastore/<name>/<namespace>` si respalda en un namespace.
3. **Token con separación de privilegios** ("Privilege Separation" marcado, valor predeterminado): conceda el rol al propio **token** (`nimbus@pbs!laptop01`), no solo al usuario — los derechos efectivos son la intersección de ambos. Es la causa más habitual de "permission denied".

`DatastoreBackup` permite al cliente crear copias y listar y restaurar sus propios grupos de copias. Eliminar o depurar (prune) snapshots requiere `DatastorePowerUser`.

## ⬆️ Actualizar desde ≤ 0.3.0

Instalar la 0.4.0 o posterior sobre una instalación existente la actualiza en el mismo lugar (misma identidad MSI, mismo servicio `NimbusBackup`). La carpeta de datos pasa de `C:\ProgramData\NimbusBackup` a la carpeta compartida `C:\ProgramData\ProxmoxBackupClient`: en el primer inicio, su configuración, tareas programadas, historial y token de API se copian una sola vez (los archivos existentes nunca se sobrescriben). La carpeta antigua se conserva, marcada con `COPIED-TO-ProxmoxBackupClient.txt`, para que una vuelta a una versión anterior siga funcionando. Los snapshots creados por versiones anteriores pueden seguir restaurándose en su ubicación original.

## 📋 Requisitos

- Windows 10/11 o Windows Server (64 bits)
- Derechos de administrador (para los snapshots de VSS)
- Acceso de red a un Proxmox Backup Server

## 🔨 Compilar desde el código fuente

### Requisitos previos
- Go 1.25 o posterior
- Node.js 20 o posterior
- Wails CLI: `go install github.com/wailsapp/wails/v2/cmd/wails@v2.13.0` (la versión que usa la CI)

### Compilación
```bash
cd gui
npm install --prefix frontend
wails build      # or: wails dev  (hot reload)
```

O compile todo (CLI + GUI + servicio) con el Makefile: `make install-deps && make` (véase `make help`). Toolchain de Windows y compilación cruzada con Docker: [BUILD.md](BUILD.md).

La marca se elige según el nombre del ejecutable: `NimbusBackup.exe` se ejecuta como Nimbus Backup, cualquier otro nombre como el neutro "Proxmox Backup Client" ([`gui/brand.go`](gui/brand.go)).

## 🔗 Relación con el proyecto upstream

Nimbus Backup nació como un fork de [tizbac/proxmoxbackupclient_go](https://github.com/tizbac/proxmoxbackupclient_go) (Proxmox Backup Client en Go, de Tiziano Bacocco, GPLv3), al que añadimos la GUI de Windows, el servicio, la programación, el multi-PBS y la restauración. En septiembre de 2026, el proyecto upstream integró esa GUI y la hizo neutra en cuanto a marca ("Proxmox Backup Client GUI"). Desde la 0.4.0, Nimbus Backup se compila a partir de la misma base de código: **los dos proyectos son ahora casi idénticos en cuanto a funcionalidad.**

Lo que este repositorio añade sobre el proyecto upstream es una pequeña serie de parches documentada ([`patches/`](patches/README.md)):

- **Correcciones aún no integradas upstream** (compilación del servicio, restauración con servidores de usuario/contraseña, códigos de salida, ocultación en los registros, fiabilidad de la copia de máquina…) — enviadas upstream a medida que se integran.
- **La identidad Nimbus Backup** — `NimbusBackup.exe`/`.msi`, el servicio `NimbusBackup` y el código de actualización MSI de las instalaciones existentes, para que sigan actualizándose en el mismo lugar.
- **Ruta de actualización** desde Nimbus Backup ≤ 0.3.0 (migración de la carpeta de datos, metadatos de snapshots heredados).
- **Cadena de publicación** — atestación de procedencia de la compilación, sumas de verificación, informes de VirusTotal.

Volvemos a fusionar el proyecto upstream con regularidad; las versiones propias del upstream se publican en [tizbac/proxmoxbackupclient_go](https://github.com/tizbac/proxmoxbackupclient_go/releases).

## ⚠️ Aviso legal

Este software se proporciona tal cual. Aunque nos esforzamos por que sea fiable, no asumimos ninguna responsabilidad por pérdidas o daños de datos. Pruebe siempre sus copias y verifique la restauración antes de confiar en ellas en producción.

Este proyecto **no está afiliado** a **Proxmox Server Solutions GmbH**. "Proxmox" y los nombres relacionados son propiedad de sus respectivos titulares y se utilizan aquí únicamente para indicar la compatibilidad.

## 📄 Licencia

GPLv3 — véase el archivo [LICENSE](LICENSE).

## Acerca de RDEM Systems

NimbusBackupClient es desarrollado y mantenido por [RDEM Systems](https://www.rdem-systems.com/en/?utm_source=github), un proveedor de infraestructura francés especializado en servicios gestionados de Proxmox VE/PBS y en infraestructura NTP/NTS. Operamos [16 servidores NTS públicos](https://ntp.rdem-systems.com/en/nts.php?utm_source=github) ([estado en tiempo real](https://ntp.rdem-systems.com/en/status.php?utm_source=github); 11 de ellos figuran en la [referencia de la comunidad](https://github.com/jauderho/nts-servers)) y ofrecemos [alojamiento PBS totalmente gestionado](https://nimbus.rdem-systems.com/en/?utm_source=github) para quienes no quieren alojarlo por su cuenta. También mantenemos [unofficial-proxmox-backup-client](https://github.com/rdemsystems/unofficial-proxmox-backup-client), paquetes `proxmox-backup-client` firmados para las distribuciones Linux que Proxmox no admite oficialmente.

---

**© 2024-2026 RDEM Systems and Proxmox Backup Client GO contributors.**
