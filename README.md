# Nimbus Backup - Proxmox Backup Server Client

Modern Windows backup client for Proxmox Backup Server with GUI and CLI interfaces.

## 🌐 Links

- **RDEM Systems**: https://www.rdem-systems.com/
- **Nimbus Backup - Fully managed Proxmox Backup Servers Datastore**: https://nimbus.rdem-systems.com/

## 📦 Two Modes Available

### 🖥️ GUI Mode (Recommended)
Modern Wails v2 interface with real-time progress, VSS support, and snapshot management.

👉 **[Download Latest Release](https://github.com/tizbac/proxmoxbackupclient_go/releases)**

Features:
- User-friendly configuration interface
- Real-time backup progress with ETA
- VSS (Volume Shadow Copy) support
- Snapshot browsing and restore
- Multi-folder backup support
- Automatic hostname detection

### ⌨️ CLI Mode (Advanced)
Command-line interface for automation and scripting.

See [CLI Documentation](#cli-usage) below.

## ⚠️ Disclaimer

This software is provided as-is. While we strive for reliability, we take no responsibility for any data loss or damage.
Always test your backups and verify restoration before relying on them in production.

## 🔮 Planned Features

### High Priority
- **Client-side encryption** - Generate/import encryption keys, encrypt before upload to PBS (PBS native support)
  - ⚠️ **Important**: Users must backup encryption keys in a safe location!
- **Code signing** - Sign Windows binaries with Authenticode certificate to prevent false positives
- **Auto-update system** - Check for latest version from official releases and prompt user to update
- **System tray icon** - Run in background with tray menu for quick access
- **Scheduled backups** - Configure automatic backups (daily, weekly, custom cron)
- **Windows service mode** - Run as a Windows service for always-on backups

### Future
- **Full disk backup** - Physical drive backup (PhysicalDrive0, etc.) - requires code signing
- **Bandwidth limiting** - Control upload speed to avoid network saturation
- **Multi-core compression** - Parallel compression for faster backups
- **Windows symlinks support** - Handle Windows symlinks properly
- **Notification system** - Windows toast notifications for backup events

## 🤝 Contributions Welcome

Priority areas:
1. ✅ ~~GUI with progress display~~ (Implemented in v0.1.0)
2. 🔐 Client-side encryption support
3. 🔄 Auto-update mechanism
4. 📅 Scheduled backups
5. 🔒 Security hardening and code signing

---

## 🔨 Building from Source

### Prerequisites
- Go 1.22 or later
- Node.js 20 or later (for GUI)
- Make
- Wails CLI (for GUI): `go install github.com/wailsapp/wails/v2/cmd/wails@latest`

### Quick Build
```bash
# Build everything (CLI + GUI)
make all

# Build only CLI tools
make cli

# Build only GUI
make gui

# Build for all platforms
make cross-compile

# Run tests with security checks
make test security-check lint
```

### Build Output
All binaries are placed in the `dist/` directory:
- `dist/proxmoxbackup-directory` - Directory backup CLI
- `dist/proxmoxbackup-machine` - Machine backup CLI
- `dist/proxmoxbackup-nbd` - NBD server CLI
- `dist/NimbusBackup.exe` - GUI application (Windows)

### Development
```bash
# Setup development environment
make dev-setup

# Run GUI in dev mode (hot reload)
make gui-dev

# Run tests with coverage
make test test-coverage
```

---

## CLI Usage

### Directory Backup

A typical command would look like:

```shell
proxmoxbackupgo.exe -baseurl "https://yourpbshost:8007" -certfingerprint pbsfingerprint -authid "user@realm!apiid" -secret "apisecret" -backupdir "C:\path\to\backup" -datastore "datastorename"
```

```
proxmoxbackupgo.exe
  -authid string
        Authentication ID (PBS Api token)
  -secret string
        Secret for authentication
  -backupdir string
        Backup source directory, must not be symlink
  -baseurl string
        Base URL for the proxmox backup server, example: https://192.168.1.10:8007
  -certfingerprint string
        Certificate fingerprint for SSL connection, example: ea:7d:06:f9...
  -datastore string
        Datastore name
  -namespace string
        Namespace (optional)
  -backup-id string
        Backup ID (optional - if not specified, the hostname is used as the default for host-type backups)
  -pxarout string
        Output PXAR archive for debug purposes (optional)
  -backupstream string  ***NEW***
        Filename for stream backup
  -mail-host string
        mail notification system: mail server host(optional)
  -mail-port string
        mail notification system: mail server port(optional)
  -mail-username string
        mail notification system: mail server username(optional)
  -mail-password string
        mail notification system: mail server password(optional)
  -mail-insecure bool
        mail notification system: allow insecure communications(optional)
  -mail-from string
        mail notification system: sender mail(optional)
  -mail-to string
        mail notification system: receiver mail(optional)

  -mail-subject-template string
        mail notification system: mail subject template(optional)
  -mail-body-template string
        mail notification system: mail body template(optional)

  -config string
        Path to JSON config file. If this flag is provided all the others will override the loaded config file
```

For JSON configuration a JSON example is provided, fill in only the needed fields.

Note on mail templating:
[Go's templating engine](https://pkg.go.dev/text/template) is used for mail subjects and bodies, please refer to the documentation for the syntax.
The following variables are available for templating:

- `.NewChunks`: number of new chunks created
- `.ReusedChunks`: number of chunks reused
- `.Datastore`: datastore name
- `.Error`: error message if any
- `.Hostname`: hostname of the machine
- `.StartTime`: time the backup started
- `.EndTime`: time the backup ended
- `.Duration`: duration of the backup
- `.FromattedDuration`: formatted duration of the backup
- `.Success`: a boolean telling whether the backup was successful 
- `.Status`: string representation of the backup status [SUCCESS, FAILURE]

Stream Backup
=============

This allows backing up a stream instead of a PXAR, allows endless possibilities for example you can invoke 

```
mysqldump yourdatabase | ./proxmoxbackupgo -backupstream yourdatabase.sql [other options]
```

This allows leveraging buzhash for dedup even when using tar for example, or the sql dump itself, and if someone wants to attempt it should be possible with some hack to pipe DISM command to generate WIM image to this and have full host backup

Known Issues
============

Windows defender antimalware being active will slow backup down up to 25% of attainable speed 

~~There's as of now no mechanism to prevent two instances being launched at same time which will screw up VSS and backup~~
If you using windows planning utility it should theoretically prevent two instances starting at same time when originating from same job

# NEW! - Full machine live backup

New funcionality has been added that now allows backing up a complete Windows 10/11 system and their respective server versions without any downtime.

The command syntax is mostly the same , except `-backupdir string`

In the case of machine backup executable there's in place of backupdir, `-backupdev`
For example an invocation could be

`machinebackup.exe -authid 'yourapikey' -backupdev \\\\.\PhysicalDisk0 -baseurl https://yourpbs:8007 -certfingerprint "xx:xx:xx..." -datastore zfs -secret 'L4m3r' -backup-id "testfull1"`

The above command will look at Disk 0 , detect all mounted partition, take VSS snapshot of these, and then create a bootable backup image of whole disk as FIDX.

Next backup will be incremental, hashing has been paralleled so speeds of 1 gbyte/sec can be easily reached.

### File restore - NEW!

File restore is possible by using nbd tool

In order to use nbd please first do `modprobe nbd max_part=0`.

For unknown reasons, using `max_part != 0` causes infinite partition probe loop.

NBD tool will connect any fixed disk backup, regardless of it being VM or host ( that being said it works also for PVE backups).

To use it use a command line similiar to this
`./pbsnbd -authid 'apikey' -baseurl https://yourpbs:8007 -secret 'yoursecret' -certfingerprint 'aa:...:xx' -datastore test -namespace test1 -pat  
h "vm/107/2025-08-02T23:13:01Z/drive-virtio0.img.fidx"`
If you omit `-path` , a terminal Ui will show up allowing to select fidx file

Beware to not use this on a machine running important stuff ( corrupt filesystem can crash the OS potentially, that why Proxmox VE uses a QEMU instance for this ).

Also be very sure to have umounted anything on the nbd disk before stopping pbsnbd, if not you likely will end with busy unmountable partition, if someone has indiciation of how to recover from that please tell me.

If you get a `Device or resource busy` error, you have to force disconnect by running `nbd-client -d /dev/nbd0` or simply reboot.

### Restore to physical machine

A live cd / PXE boot system will be released that will allow logging in to a PBS server, selecting the backup, and launching clonezilla.
For now best way is spinning up a clonezilla live and copying to it nbd server executable, before proceeding with clonezilla, on another tty, you launch `pbsnbd`.

I suggest also copying over command line parameters such as authid, baseurl, fingerprint etc, they are a pain in the... to hand type!.

Once pbsnbd is up and running, you can use clonezilla disk to local disk option. 

# Tech Support in Italy

Being said that the project and **ALL** it's contribution will remain forever public and licensed under GPLv3 license, main sponsor of this project and also latest machine backup features, E.T.I. Srl ( https://etitech.net ) can provide to who needs it support for Proxmox deployments in general and specifically Windows backup tools.

There will never be a "community" & "enterprise" different edition, solely tech support will be an independent service.
Any customization that you are going to ask, even if development may be paid it will be released as GPLv3 like whole project is.
