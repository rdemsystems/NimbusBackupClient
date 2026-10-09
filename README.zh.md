# Nimbus Backup — 适用于 Proxmox Backup Server 的 Windows 客户端

[🇬🇧 English](README.md) · [🇫🇷 Français](README.fr.md) · [🇮🇹 Italiano](README.it.md) · [🇩🇪 Deutsch](README.de.md) · [🇪🇸 Español](README.es.md) · [🇷🇺 Русский](README.ru.md) · 🇨🇳 中文 · [🇯🇵 日本語](README.ja.md) · [🇬🇷 Ελληνικά](README.el.md) · [🇷🇴 Română](README.ro.md) · [🇸🇪 Svenska](README.sv.md) · [🇸🇦 العربية](README.ar.md) · [🇮🇷 فارسی](README.fa.md)

> 🤖 本译文由 AI 根据英文 README 生成；如有出入，以[英文版](README.md)为准。
> *AI-generated translation of the English README; the [English version](README.md) is authoritative.*

[![License](https://img.shields.io/badge/license-GPLv3-blue.svg)](LICENSE)
[![Release](https://img.shields.io/github/v/release/rdemsystems/NimbusBackupClient)](https://github.com/rdemsystems/NimbusBackupClient/releases)
[![Documentation](https://img.shields.io/badge/docs-nimbus.rdem--systems.com-orange)](https://nimbus.rdem-systems.com/en/?utm_source=github)

**Nimbus Backup 是一款开源（GPL-3.0）的 Windows 备份客户端，用于 Proxmox Backup Server（PBS）。**
它提供现代化的图形界面，可将 Windows 服务器和工作站备份到 PBS——支持 VSS 一致性快照、计划任务、文件与磁盘模式、快照浏览与恢复、多 PBS 支持以及 Windows 服务——另外还提供用于目录备份和整机备份的命令行工具。想要无需自建、**异地且不可变**的 PBS 存储？请参阅下文的[托管服务](#️-托管-pbs异地不可变)。

Nimbus Backup 是 RDEM Systems 对 [Proxmox Backup Client GO](https://github.com/tizbac/proxmoxbackupclient_go) 的构建版本：我们开发的图形界面已合并到上游，两个项目现在共享同一代码库（参见[与上游项目的关系](#-与上游项目的关系)）。

📖 **完整文档、安装指南和 PBS 托管：** [nimbus.rdem-systems.com](https://nimbus.rdem-systems.com/en/blog/backup-windows-proxmox-backup-server/?utm_source=github&utm_medium=readme&utm_campaign=nbc-readme-top)

## 📦 下载

👉 **[下载最新版本](https://github.com/rdemsystems/NimbusBackupClient/releases)**

每个版本包含：
- `NimbusBackup.msi` — 安装程序（GUI + Windows 服务），**推荐用于生产环境**
- `NimbusBackup.exe` — 独立 GUI
- `nimbus-backup-cli-<version>-windows.zip` / `-linux.tar.gz` / `-macos.tar.gz` — 命令行工具
- `SHA256SUMS.txt` — 校验和

> ⚠️ **Windows 提示“检测到病毒”（例如 `Trojan:Win32/Sabsik.FL.A!ml`）或显示 SmartScreen 警告？**
> 这是 Go/Wails 应用程序已知的**误报**——它*不是*病毒。`!ml` 后缀表示该告警来自一个机器学习模型，该模型会标记*普及度低*的可执行文件（在 0.4.0 及之前，还包括未签名的文件）。
> 阅读[为什么会出现这种情况以及如何验证下载文件](https://nimbus.rdem-systems.com/en/antivirus-false-positive/?utm_source=github)。

### 🔎 验证任何下载文件

每个版本都附带 SHA-256 校验和以及经过签名的**构建来源证明**（build-provenance attestation，以密码学方式证明该二进制文件由本仓库的 CI 从特定提交构建）：

```powershell
Get-FileHash .\NimbusBackup.msi -Algorithm SHA256   # compare against SHA256SUMS.txt
gh attestation verify .\NimbusBackup.msi --repo rdemsystems/NimbusBackupClient
```

**VirusTotal。** 每个版本的发行说明都会链接该构建安装程序的 VirusTotal 报告（仅在扫描结果干净时发布）。早期报告，0 检出：
[0.2.108](https://www.virustotal.com/gui/file/6e8fb7ce9af740d470e947addb8daba4331c0b88e8bfdec9e0697ea8f7f29e9e/detection) ·
[0.2.107](https://www.virustotal.com/gui/file/6fd6c6fa77e0305c129ef882a3745100aa6033187a6d52a4af94149ab6b666d2/detection) ·
[0.2.106](https://www.virustotal.com/gui/file/ad6e56700ed9df8e088906e38cee2e2882fc7045f4e39269de0e379a01784ad7/detection)

> 🔏 **代码签名：** 自 0.4.1 起，`NimbusBackup.exe`、其服务以及 `NimbusBackup.msi` 均**由 RDEM SYSTEMS 进行 Authenticode 签名**（Azure Artifact Signing）；*属性 → 数字签名* 中会显示发布者。在新版本的信誉建立之前，SmartScreen 仍可能发出警告：请确认发布者为 RDEM SYSTEMS，然后点击 *更多信息 → 仍要运行*。命令行工具尚未签名；上述构建来源证明和校验和覆盖所有文件。

### 🐧 使用 Linux？请使用官方客户端

Nimbus Backup GUI 仅支持 Windows（CLI 工具也可为 Linux 和 macOS 构建）。在 Linux 上进行文件级备份，请使用 Proxmox 自己的 `proxmox-backup-client`——我们为 Proxmox 未覆盖的发行版提供了打包：

👉 **[rdemsystems/unofficial-proxmox-backup-client](https://github.com/rdemsystems/unofficial-proxmox-backup-client)** — 适用于 Debian/Ubuntu、Fedora/RHEL/Rocky/AlmaLinux、Arch 和 Alpine（amd64 和 arm64）的签名软件包仓库。这是 Proxmox 官方静态二进制文件的原样重新打包——未打补丁，未重新编译。

## ☁️ 托管 PBS（异地、不可变）

不想自行托管 Proxmox Backup Server？使用我们全托管的**异地不可变** PBS 数据存储：
👉 **[配置您的备份并查看价格](https://nimbus.rdem-systems.com/en/choose-backup/?utm_source=github)**

- ✅ 低至 €12/TB/月
- ✅ 1 TB 免费试用
- ✅ 现成的 [PBS 异地备份目标](https://nimbus.rdem-systems.com/en/offsite-proxmox-backup/?utm_source=github)——物理隔离（air-gapped）的不可变数据存储
- ✅ [NimbusBackup — 位于法国的托管 PBS 服务](https://nimbus.rdem-systems.com/en/?utm_source=github)

## 📚 文档

- **Proxmox Backup 完整指南** — PBS 部署最佳实践（[🇬🇧 EN](https://nimbus.rdem-systems.com/en/blog/complete-proxmox-backup-guide/?utm_source=github)）
- **使用 Proxmox Backup Server 备份 Windows** — Windows 专用部署指南（[🇬🇧 EN](https://nimbus.rdem-systems.com/en/blog/backup-windows-proxmox-backup-server/?utm_source=github)）
- **PBS 与 Veeam 对比** — Proxmox Backup Server 比较（[🇬🇧 EN](https://nimbus.rdem-systems.com/en/blog/pbs-vs-veeam-proxmox-backup-comparison/?utm_source=github)）
- 本仓库内：[多 PBS 用户指南](MULTI_PBS_USER_GUIDE.md) · [无人值守部署示例](examples/automation/) · [使用 Clonezilla 进行裸机恢复](PATCH-CLONEZILLA.md) · [更新日志](CHANGELOG.md)

## ✨ 功能

### GUI（推荐）
- **🌍 多语言** — 英语、法语、意大利语、德语和波兰语界面
- 友好的配置界面，支持连接测试（API 令牌或用户名/密码）
- 实时显示备份进度、速度和预计剩余时间，可随时取消，提供“运行中的作业”标签页
- VSS（卷影复制）实现一致性备份：整个多文件夹备份只用一个快照，每个卷一个卷影副本
- 多文件夹备份，文件模式和磁盘（整机）模式；文件夹可并行备份（推荐：CPU 数 / 4）
- 磁盘备份可在 Proxmox VE 中恢复为与原机器一致的虚拟机（CPU、内存、固件、带原 MAC 地址的网卡、专用 VM ID）
- 快照浏览、文件搜索（支持通配符）和恢复，包括 NTFS ACL
- 支持多台 PBS 服务器，每个作业可使用各自的 PBS 服务器，证书指纹固定（TOFU）
- **🔒 客户端加密**（AES-256-GCM），密钥文件与 `proxmox-backup-client` 及 Proxmox VE 兼容
- Windows 服务模式 + 计划备份，备份历史记录支持一键重新运行
- 用于故障排查的调试日志

### 命令行工具
- `proxmoxbackup-directory` — 带去重的目录（PXAR）备份，流式备份（`-backupstream`，例如 `mysqldump` 管道），排除规则（`-exclude "*.tmp"`，可重复使用，或 `-exclude-from file`；JSON 配置中的 `"exclude"`），同时备份多个目录（`-parallel N`，推荐：CPU 数 / 4），电子邮件通知，JSON 配置文件
- `proxmoxbackup-machine` — 以可启动磁盘镜像（FIDX）形式对运行中的整机进行完整备份：Windows 上使用 VSS，增量备份，并行哈希计算
- `proxmoxbackup-nbd` — NBD 服务器，用于在 Linux 上挂载磁盘备份（文件级恢复，以及通过[打过补丁的 Clonezilla live ISO](PATCH-CLONEZILLA.md) 进行裸机恢复）

### 📸 截图

![服务器配置](docs/screenshots/nimbus-gui-liste-servers.png)
*带状态指示的多 PBS 服务器管理*

![添加服务器表单](docs/screenshots/nimbus-gui-add-server-form.png)
*带连接测试的简便服务器配置*

![一次性备份](docs/screenshots/nimbus-gui-one-shot-backup.png)
*实时显示备份进度、预计剩余时间和速度*

### 智能系统排除（文件模式）
备份整个驱动器（例如 `D:\`）时，Nimbus Backup 会自动排除：

**系统文件夹：** `System Volume Information`（VSS 存储，可能超过 100 GB）、`$RECYCLE.BIN`、`Recovery`。
**系统文件：** `pagefile.sys`、`hiberfil.sys`、`swapfile.sys`。

**为什么重要：** 一个驱动器可能显示已用 1.03 TB，而实际文件只有约 141 GB。若不排除，备份将包含 VSS 快照（浪费空间和时间）；排除之后，备份大小与实际数据相符。

**建议：** 文件级备份请使用带自动排除的**文件模式**（默认）；裸机恢复请在单独的任务中使用**磁盘模式**（包含所有内容）。

### 安全与质量
- 输入验证和凭据清理（日志中的机密信息会被脱敏）
- 防止路径遍历
- 带指数退避的重试逻辑
- 每次构建都执行 CI 检查：测试、`golangci-lint`、`gosec`、`go mod tidy`，以及针对真实 PBS 的端到端测试套件（使用官方 `proxmox-backup-client` 恢复、PBS 校验）

### 🔒 客户端加密
备份可以在离开机器之前**在客户端**进行加密，
采用与官方 `proxmox-backup-client` 相同的方案（AES-256-GCM，带密钥的数据块
摘要）。PBS 服务器只存储不透明的数据，永远看不到
密钥——这在共享或托管的 PBS 上非常有用。

加密功能由 Tiziano Bacocco 在上游项目中开发（[tizbac/proxmoxbackupclient_go](https://github.com/tizbac/proxmoxbackupclient_go)）；纸质密钥、二维码导出和密钥导入是 Nimbus Backup 新增的功能。

- **GUI**：*服务器 → 编辑 → 加密密钥* — 创建密钥文件或选择
  已有的密钥文件（来自 `proxmox-backup-client key create --kdf none` 或 PVE
  存储）；界面会显示其指纹。然后请在机器之外保留一份副本：
  **打印（纸质密钥）**会保存一个可打印页面，包含密钥及其二维码（即
  `proxmox-backup-client key paperkey` 的格式），可选用口令保护。
  **从文本或二维码导入密钥**可将扫描的二维码或纸质密钥
  还原为密钥文件。GUI 只使用未受保护的密钥文件：导入
  受口令保护的密钥时会将其解锁，并将新的密钥文件保存为**无**
  口令——请像保管密钥本身一样妥善保管该文件。
- **CLI**：`-keyfile path/to/key.json`（或 JSON 配置中的 `"keyfile"`）；
  受口令保护的密钥使用 `-keyfile-passphrase`，否则会提示输入口令。
- **裸机恢复**：打过补丁的 Clonezilla ISO 也能恢复加密的磁盘
  备份（密钥从 U 盘读取，如有口令则一并提供）——参见
  [PATCH-CLONEZILLA.md](PATCH-CLONEZILLA.md)。
- **与 `proxmox-backup-client` 兼容，每次构建都在 CI 中**
  针对真实的 PBS 进行验证：由 Nimbus Backup 使用 `proxmox-backup-client`
  创建的密钥加密的文件夹，可由官方客户端用同一密钥恢复
  （没有密钥则被拒绝）；由官方客户端创建的加密磁盘备份
  也能被 Nimbus Backup 读回。Proxmox VE 使用相同的密钥文件。

> ⚠️ **没有密钥，加密备份将无法恢复。** 第一次
> 加密备份会重新上传所有数据（与未加密备份之间不进行去重）。
> 备份 ID、归档名称和大小对服务器仍然可见；文件
> 内容、文件名和目录（catalog）均已加密。

## 🤖 无人值守部署（Ansible 与 IaC）

Nimbus Backup 完全通过文件进行安装和配置——无需交互式设置。
有两种方式，均在
[`examples/automation/`](examples/automation/) 中提供了可直接使用的示例：

- **命令行** — 每台主机一个 JSON 文件即可描述整个备份
  （`proxmoxbackup-directory.exe --config file.json`）；通过 Windows
  任务计划程序进行调度。可靠的退出码（`0` 成功，`1` 致命错误，`2` 已锁定，`3` 部分完成），
  便于编排工具检测失败。最适合纯粹的基础设施即代码。
- **GUI/服务（MSI）** — **单个 `config.json`** 即包含 PBS 连接、
  备份设置**以及**计划（`scheduled_jobs`）。推送该文件，重启
  `NimbusBackup` 服务：任务会以幂等方式进行协调，`nextRun` 也会
  自动计算。您的 Jinja2 模板中无需进行任何时间戳计算。

服务从 `C:\ProgramData\ProxmoxBackupClient\` 读取配置
（自 0.4.0 起；参见[升级](#️-从--030-升级)）。

📖 完整教程：[使用 Ansible 自动将 Windows 备份到 PBS](https://nimbus.rdem-systems.com/en/blog/unattended-windows-backup-ansible/?utm_source=github)。

## 🚀 快速开始

1. 从发布页面下载 `NimbusBackup.msi`（或独立的 `NimbusBackup.exe`）
2. 以管理员权限安装/运行（VSS 需要）
3. 配置 PBS 连接并进行测试
4. 选择要备份的目录
5. 开始备份——或为其设置计划

### 🔑 PBS 用户与权限

客户端只需要目标数据存储上的 **`DatastoreBackup`** 角色——无需管理员账户：

1. 在 PBS 界面中创建一个用户（例如 `nimbus@pbs`）并为其创建 API 令牌（例如 `nimbus@pbs!laptop01`）。
2. 在 **Datastore → Permissions**（或 **Configuration → Access Control → Permissions**）中，在 `/datastore/<name>` 上授予 `DatastoreBackup`——如果备份到命名空间，则在 `/datastore/<name>/<namespace>` 上授予。
3. **权限分离令牌**（勾选了“Privilege Separation”，这是默认设置）：将角色授予**令牌**本身（`nimbus@pbs!laptop01`），而不仅仅是用户——实际权限是两者的交集。这是出现“permission denied”最常见的原因。

`DatastoreBackup` 允许客户端创建备份，并列出和恢复其自己的备份组。删除或清理（prune）快照需要 `DatastorePowerUser`。

## ⬆️ 从 ≤ 0.3.0 升级

在现有安装上安装 0.4.0 或更高版本会进行原地升级（相同的 MSI 标识，相同的 `NimbusBackup` 服务）。数据文件夹从 `C:\ProgramData\NimbusBackup` 迁移到共享的 `C:\ProgramData\ProxmoxBackupClient`：首次启动时，您的配置、计划任务、历史记录和 API 令牌会被复制一次（绝不会覆盖已有文件）。旧文件夹会被保留，并以 `COPIED-TO-ProxmoxBackupClient.txt` 标记，因此仍可降级。旧版本创建的快照仍可恢复到其原始位置。

## 📋 系统要求

- Windows 10/11 或 Windows Server（64 位）
- 管理员权限（用于 VSS 快照）
- 能够通过网络访问 Proxmox Backup Server

## 🔨 从源代码构建

### 前置条件
- Go 1.25 或更高版本
- Node.js 20 或更高版本
- Wails CLI：`go install github.com/wailsapp/wails/v2/cmd/wails@v2.13.0`（CI 使用的版本）

### 构建
```bash
cd gui
npm install --prefix frontend
wails build      # or: wails dev  (hot reload)
```

或者使用 Makefile 构建全部内容（CLI + GUI + 服务）：`make install-deps && make`（参见 `make help`）。Windows 工具链和 Docker 交叉构建：[BUILD.md](BUILD.md)。

品牌由可执行文件名决定：`NimbusBackup.exe` 以 Nimbus Backup 身份运行，其他任何名称则以中性的“Proxmox Backup Client”身份运行（[`gui/brand.go`](gui/brand.go)）。

## 🔗 与上游项目的关系

Nimbus Backup 最初是 [tizbac/proxmoxbackupclient_go](https://github.com/tizbac/proxmoxbackupclient_go)（用 Go 编写的 Proxmox Backup Client，作者 Tiziano Bacocco，GPLv3）的一个分支，我们在其基础上添加了 Windows GUI、服务、计划任务、多 PBS 和恢复功能。2026 年 9 月，上游合并了该 GUI 并使其品牌中立（“Proxmox Backup Client GUI”）。2026 年 10 月（0.4.1），Nimbus Backup 基于上游的当前代码重新构建，并采用了上游的客户端加密：**两个项目共享同一代码库。**

Nimbus Backup 0.4.1 基于上游项目 `master` 分支的提交 `3c1b989`（2026 年 10 月 9 日）构建。目前还没有包含这些代码的上游 release：上游最新的 release v1.1.3（2026 年 5 月）早于 GUI 合并。

本仓库在上游基础上增加的是一个有文档说明的补丁系列（[`patches/`](patches/README.md)），其中大部分已提交给上游：

- **功能**：纸质密钥和密钥导入（二维码）、文件夹并行备份、多文件夹备份只用一个 VSS 快照、命令行排除项、根据真实机器生成的 Proxmox VE 虚拟机配置。
- **尚未合并到上游的修复**（恢复 `proxmox-backup-client` 的压缩加密备份、服务备份所选用的 PBS 服务器、PBS 拒绝原因、使用用户名/密码的服务器的恢复、窗口最大化……）——已陆续提交给上游，等待其合并。
- **Nimbus Backup 标识** — `NimbusBackup.exe`/`.msi`、`NimbusBackup` 服务，以及现有安装的 MSI 升级代码，以便它们继续原地升级。
- **升级路径** — 从 Nimbus Backup ≤ 0.3.0 升级（数据文件夹迁移、旧版快照元数据）。
- **发布流水线** — 构建来源证明、校验和、VirusTotal 报告。

我们会定期重新合并上游；上游自己的版本发布在 [tizbac/proxmoxbackupclient_go](https://github.com/tizbac/proxmoxbackupclient_go/releases)。

## ⚠️ 免责声明

本软件按“原样”提供。尽管我们力求可靠，但对任何数据丢失或损坏概不负责。在生产环境中依赖备份之前，请务必测试您的备份并验证恢复。

本项目与 **Proxmox Server Solutions GmbH** **没有任何关联**。“Proxmox”及相关名称归其各自所有者所有，此处仅用于说明兼容性。

## 📄 许可证

GPLv3 — 参见 [LICENSE](LICENSE) 文件。

## 关于 RDEM Systems

NimbusBackupClient 由 [RDEM Systems](https://www.rdem-systems.com/en/?utm_source=github) 开发和维护，这是一家法国基础设施服务商，专注于 Proxmox VE/PBS 托管服务以及 NTP/NTS 基础设施。我们运营 [16 台公共 NTS 服务器](https://ntp.rdem-systems.com/en/nts.php?utm_source=github)（[实时状态](https://ntp.rdem-systems.com/en/status.php?utm_source=github)；其中 11 台已列入[社区参考列表](https://github.com/jauderho/nts-servers)），并为不想自行托管的用户提供[全托管 PBS 服务](https://nimbus.rdem-systems.com/en/?utm_source=github)。我们还维护 [unofficial-proxmox-backup-client](https://github.com/rdemsystems/unofficial-proxmox-backup-client)，即面向 Proxmox 未官方支持的 Linux 发行版的签名 `proxmox-backup-client` 软件包。

---

**© 2024-2026 RDEM Systems 及 Proxmox Backup Client GO 贡献者。**
