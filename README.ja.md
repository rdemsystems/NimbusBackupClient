# Nimbus Backup — Proxmox Backup Server 用 Windows クライアント

[🇬🇧 English](README.md) · [🇫🇷 Français](README.fr.md) · [🇮🇹 Italiano](README.it.md) · [🇩🇪 Deutsch](README.de.md) · [🇪🇸 Español](README.es.md) · [🇷🇺 Русский](README.ru.md) · [🇨🇳 中文](README.zh.md) · 🇯🇵 日本語 · [🇬🇷 Ελληνικά](README.el.md) · [🇷🇴 Română](README.ro.md) · [🇸🇪 Svenska](README.sv.md) · [🇸🇦 العربية](README.ar.md) · [🇮🇷 فارسی](README.fa.md)

> 🤖 この翻訳は英語版 README をもとに AI によって作成されました。内容に相違がある場合は[英語版](README.md)が優先されます。
> *AI-generated translation of the English README; the [English version](README.md) is authoritative.*

[![License](https://img.shields.io/badge/license-GPLv3-blue.svg)](LICENSE)
[![Release](https://img.shields.io/github/v/release/rdemsystems/NimbusBackupClient)](https://github.com/rdemsystems/NimbusBackupClient/releases)
[![Documentation](https://img.shields.io/badge/docs-nimbus.rdem--systems.com-orange)](https://nimbus.rdem-systems.com/en/?utm_source=github)

**Nimbus Backup は、Proxmox Backup Server（PBS）向けのオープンソース（GPL-3.0）の Windows バックアップクライアントです。**
Windows サーバーやワークステーションを PBS にバックアップするためのモダンな GUI を備えています — VSS による整合性のあるスナップショット、スケジュールジョブ、ファイルモードとディスクモード、スナップショットの閲覧とリストア、複数 PBS 対応、Windows サービス — さらに、ディレクトリ単位およびマシン全体のバックアップ用のコマンドラインツールも提供します。自前でホストせずに**オフサイトかつイミュータブル**な PBS ストレージをお探しですか？ 下記の[マネージドサービス](#️-マネージド-pbsオフサイトイミュータブル)をご覧ください。

Nimbus Backup は [Proxmox Backup Client GO](https://github.com/tizbac/proxmoxbackupclient_go) の RDEM Systems 版ビルドです。私たちが開発した GUI はアップストリームにマージされ、現在では両プロジェクトが同じコードベースを共有しています（[アップストリームとの関係](#-アップストリームとの関係)を参照）。

📖 **完全なドキュメント、インストールガイド、PBS ホスティング：** [nimbus.rdem-systems.com](https://nimbus.rdem-systems.com/en/blog/backup-windows-proxmox-backup-server/?utm_source=github&utm_medium=readme&utm_campaign=nbc-readme-top)

## 📦 ダウンロード

👉 **[最新リリースをダウンロード](https://github.com/rdemsystems/NimbusBackupClient/releases)**

各リリースには以下が含まれます：
- `NimbusBackup.msi` — インストーラー（GUI + Windows サービス）、**本番環境に推奨**
- `NimbusBackup.exe` — スタンドアロン GUI
- `nimbus-backup-cli-<version>-windows.zip` / `-linux.tar.gz` / `-macos.tar.gz` — コマンドラインツール
- `SHA256SUMS.txt` — チェックサム

> ⚠️ **Windows が「ウイルスが検出されました」（例：`Trojan:Win32/Sabsik.FL.A!ml`）と表示したり、SmartScreen の警告が出たりしますか？**
> これは Go/Wails アプリケーションで既知の**誤検知**であり、ウイルスでは*ありません*。`!ml` という接尾辞は、*普及度の低い*実行ファイル（0.4.0 までは署名のないものも）を検出する機械学習モデルによる判定であることを意味します。
> [この現象が起こる理由とダウンロードの検証方法](https://nimbus.rdem-systems.com/en/antivirus-false-positive/?utm_source=github)をお読みください。

### 🔎 ダウンロードの検証

すべてのリリースには SHA-256 チェックサムと、署名付きの**ビルド来歴証明**（build-provenance attestation：バイナリがこのリポジトリの CI によって特定のコミットからビルドされたことの暗号学的証明）が付属します：

```powershell
Get-FileHash .\NimbusBackup.msi -Algorithm SHA256   # compare against SHA256SUMS.txt
gh attestation verify .\NimbusBackup.msi --repo rdemsystems/NimbusBackupClient
```

**VirusTotal。** 各リリースノートには、そのビルドのインストーラーの VirusTotal レポートへのリンクが記載されています（スキャン結果がクリーンな場合のみ公開）。過去のレポート（検出数 0）：
[0.2.108](https://www.virustotal.com/gui/file/6e8fb7ce9af740d470e947addb8daba4331c0b88e8bfdec9e0697ea8f7f29e9e/detection) ·
[0.2.107](https://www.virustotal.com/gui/file/6fd6c6fa77e0305c129ef882a3745100aa6033187a6d52a4af94149ab6b666d2/detection) ·
[0.2.106](https://www.virustotal.com/gui/file/ad6e56700ed9df8e088906e38cee2e2882fc7045f4e39269de0e379a01784ad7/detection)

> 🔏 **コード署名：** 0.4.1 以降、`NimbusBackup.exe`、そのサービス、`NimbusBackup.msi` は **RDEM SYSTEMS により Authenticode 署名されています**（Azure Artifact Signing）。*プロパティ → デジタル署名* で発行元を確認できます。新しいリリースでは、評価が確立されるまで SmartScreen が警告を表示することがあります。発行元が RDEM SYSTEMS であることを確認してから、*詳細情報 → 実行* を選択してください。コマンドラインツールはまだ署名されていません。上記のビルド来歴証明とチェックサムはすべてのファイルを対象としています。

### 🐧 Linux をお使いですか？ 公式クライアントをご利用ください

Nimbus Backup の GUI は Windows 専用です（CLI ツールは Linux と macOS 向けにもビルドできます）。Linux でファイル単位のバックアップを行うには、Proxmox 純正の `proxmox-backup-client` をご利用ください。Proxmox がサポートしていないディストリビューション向けに、私たちがパッケージを提供しています：

👉 **[rdemsystems/unofficial-proxmox-backup-client](https://github.com/rdemsystems/unofficial-proxmox-backup-client)** — Debian/Ubuntu、Fedora/RHEL/Rocky/AlmaLinux、Arch、Alpine（amd64 および arm64）向けの署名付きパッケージリポジトリ。Proxmox 公式の静的バイナリを変更なしで再パッケージしたもので、パッチ適用も再コンパイルもしていません。

## ☁️ マネージド PBS（オフサイト・イミュータブル）

Proxmox Backup Server を自前でホストしたくないですか？ 私たちのフルマネージドな**オフサイト・イミュータブル** PBS データストアをご利用ください：
👉 **[バックアップを構成して料金を確認](https://nimbus.rdem-systems.com/en/choose-backup/?utm_source=github)**

- ✅ 月額 €12/TB から
- ✅ 1 TB の無料トライアル
- ✅ すぐに使える [PBS 用オフサイトターゲット](https://nimbus.rdem-systems.com/en/offsite-proxmox-backup/?utm_source=github) — エアギャップされたイミュータブルなデータストア
- ✅ [NimbusBackup — フランスのマネージド PBS ホスティング](https://nimbus.rdem-systems.com/en/?utm_source=github)

## 📚 ドキュメント

- **Proxmox Backup 完全ガイド** — PBS 導入のベストプラクティス（[🇬🇧 EN](https://nimbus.rdem-systems.com/en/blog/complete-proxmox-backup-guide/?utm_source=github)）
- **Proxmox Backup Server で Windows をバックアップ** — Windows 向け導入ガイド（[🇬🇧 EN](https://nimbus.rdem-systems.com/en/blog/backup-windows-proxmox-backup-server/?utm_source=github)）
- **PBS 対 Veeam** — Proxmox Backup Server の比較（[🇬🇧 EN](https://nimbus.rdem-systems.com/en/blog/pbs-vs-veeam-proxmox-backup-comparison/?utm_source=github)）
- このリポジトリ内：[複数 PBS ユーザーガイド](MULTI_PBS_USER_GUIDE.md) · [無人デプロイの例](examples/automation/) · [Clonezilla によるベアメタルリストア](PATCH-CLONEZILLA.md) · [変更履歴](CHANGELOG.md)

## ✨ 機能

### GUI（推奨）
- **🌍 多言語対応** — 英語、フランス語、イタリア語、ドイツ語、ポーランド語のインターフェース
- 接続テスト付きのわかりやすい設定（API トークンまたはユーザー名/パスワード）
- 速度と残り時間を含むリアルタイムのバックアップ進捗表示、いつでもキャンセル可能、「Running」タブ
- 整合性のあるバックアップのための VSS（ボリュームシャドウコピー）：複数フォルダーのバックアップ全体で 1 つのスナップショット、ボリュームごとに 1 つのシャドウコピー
- 複数フォルダーのバックアップ、ファイルモードとディスク（マシン全体）モード。フォルダーは並列実行可能（推奨：CPU 数 / 4）
- ディスクバックアップは Proxmox VE 上で元のマシンに合わせた VM としてリストア可能（CPU、RAM、ファームウェア、MAC アドレスを引き継いだ NIC、専用の VM ID）
- スナップショットの閲覧、ファイル検索（ワイルドカード対応）、リストア（NTFS ACL を含む）
- 複数 PBS サーバー対応（ジョブごとに PBS サーバーを指定可能）、証明書フィンガープリントのピン留め（TOFU）
- **🔒 クライアント側暗号化**（AES-256-GCM）、鍵ファイルは `proxmox-backup-client` および Proxmox VE と互換
- Windows サービスモード + スケジュールバックアップ、ワンクリックで再実行できるバックアップ履歴
- トラブルシューティング用のデバッグログ

### コマンドラインツール
- `proxmoxbackup-directory` — 重複排除付きのディレクトリ（PXAR）バックアップ、ストリームバックアップ（`-backupstream`、例：`mysqldump` のパイプ）、除外指定（`-exclude "*.tmp"`、複数指定可、または `-exclude-from file`。JSON 設定では `"exclude"`）、複数ディレクトリの同時バックアップ（`-parallel N`、推奨：CPU 数 / 4）、メール通知、JSON 設定ファイル
- `proxmoxbackup-machine` — 稼働中のマシン全体をブータブルなディスクイメージ（FIDX）としてバックアップ：Windows では VSS、増分バックアップ、並列ハッシュ計算
- `proxmoxbackup-nbd` — Linux でディスクバックアップをマウントするための NBD サーバー（ファイル単位のリストア、[パッチ適用済み Clonezilla ライブ ISO](PATCH-CLONEZILLA.md) からのベアメタルリストア）

### 📸 スクリーンショット

![サーバー設定](docs/screenshots/nimbus-gui-liste-servers.png)
*ステータス表示付きの複数 PBS サーバー管理*

![サーバー追加フォーム](docs/screenshots/nimbus-gui-add-server-form.png)
*接続テスト付きの簡単なサーバー設定*

![単発バックアップ](docs/screenshots/nimbus-gui-one-shot-backup.png)
*残り時間と速度を含むリアルタイムのバックアップ進捗*

### スマートなシステム除外（ファイルモード）
ドライブ全体（例：`D:\`）をバックアップする場合、Nimbus Backup は以下を自動的に除外します：

**システムフォルダー：** `System Volume Information`（VSS の格納領域で、100 GB を超えることもあります）、`$RECYCLE.BIN`、`Recovery`。
**システムファイル：** `pagefile.sys`、`hiberfil.sys`、`swapfile.sys`。

**重要な理由：** ドライブの使用量が 1.03 TB と表示されていても、実際のファイルは約 141 GB ということがあります。除外しなければバックアップに VSS スナップショットが含まれてしまい（容量と時間の無駄）、除外すればバックアップサイズは実データと一致します。

**推奨：** ファイル単位のバックアップには自動除外付きの**ファイルモード**（デフォルト）を、ベアメタルリストアには別ジョブで**ディスクモード**（すべてを含む）を使用してください。

### セキュリティと品質
- 入力の検証と認証情報のサニタイズ（シークレットはログ上で伏せ字化）
- パストラバーサルの防止
- 指数バックオフによるリトライ処理
- すべてのビルドで CI チェック：テスト、`golangci-lint`、`gosec`、`go mod tidy`、および実際の PBS に対するエンドツーエンドテストスイート（公式 `proxmox-backup-client` によるリストア、PBS の verify）

### 🔒 クライアント側暗号化
バックアップはマシンから送出される前に**クライアント上で**暗号化できます。
方式は公式の `proxmox-backup-client` と同じです（AES-256-GCM、鍵付きのチャンク
ダイジェスト）。PBS サーバーは中身の見えないデータを保存するだけで、鍵を
目にすることはありません — 共有 PBS やマネージド PBS で役立ちます。

暗号化はアップストリームで Tiziano Bacocco 氏が開発しました（[tizbac/proxmoxbackupclient_go](https://github.com/tizbac/proxmoxbackupclient_go)）。ペーパーキー、QR コードでのエクスポート、鍵のインポートは Nimbus Backup による追加機能です。

- **GUI**：*PBS Configuration → Edit → Encryption key file* — 鍵ファイルを作成するか、
  既存のもの（`proxmox-backup-client key create --kdf none` で作成したもの、または PVE
  ストレージのもの）を選択します。そのフィンガープリントが表示されます。その後、マシン外にコピーを保管してください：
  **印刷（ペーパーキー）** は鍵とその QR コードを含む印刷用ページを保存します（
  `proxmox-backup-client key paperkey` の形式）。任意でパスフレーズによる保護も可能です。
  **テキストまたは QR コードから鍵をインポート** すると、スキャンした QR コードやペーパーキーを
  鍵ファイルに戻せます。GUI は保護されていない鍵ファイルのみを使用します：
  パスフレーズで保護された鍵をインポートするとロックが解除され、新しい鍵ファイルはパスフレーズ**なし**で
  保存されます — そのファイルは鍵そのものと同様に安全に保管してください。
- **CLI**：`-keyfile path/to/key.json`（または JSON 設定の `"keyfile"`）。
  パスフレーズで保護された鍵には `-keyfile-passphrase` を指定するか、入力を求められます。
- **ベアメタルリストア**：パッチ適用済みの Clonezilla ISO は暗号化されたディスク
  バックアップもリストアできます（鍵は USB メモリから、パスフレーズがあればそれも使用） — 
  [PATCH-CLONEZILLA.md](PATCH-CLONEZILLA.md) を参照してください。
- **`proxmox-backup-client` との互換性を、すべてのビルドで CI により**
  実際の PBS に対して検証：`proxmox-backup-client` で作成した鍵を使って
  Nimbus Backup が暗号化したフォルダーは、同じ鍵で公式クライアントによりリストアでき
  （鍵がなければ拒否されます）、公式クライアントで作成した暗号化ディスクバックアップは
  Nimbus Backup で読み戻せます。Proxmox VE も同じ鍵ファイルを使用します。

> ⚠️ **鍵がなければ、暗号化されたバックアップは復元できません。** 最初の
> 暗号化バックアップではすべてのデータが再アップロードされます（暗号化されていない
> バックアップとの重複排除は行われません）。バックアップ ID、アーカイブ名、サイズはサーバーから見えたままですが、ファイルの
> 内容、ファイル名、カタログは暗号化されます。

## 🤖 無人デプロイ（Ansible と IaC）

Nimbus Backup はファイルだけでインストールと設定が完結します — 対話的なセットアップは不要です。
方法は 2 つあり、いずれも
[`examples/automation/`](examples/automation/) にすぐ使える例があります：

- **コマンドライン** — ホストごとに 1 つの JSON ファイルでバックアップ全体を記述します
  （`proxmoxbackup-directory.exe --config file.json`）。スケジュールは Windows の
  タスクスケジューラで設定します。信頼できる終了コード（`0` 正常、`1` 致命的エラー、`2` ロック中、`3` 部分的成功）
  により、オーケストレーターが失敗を検出できます。純粋な Infrastructure as Code に最適です。
- **GUI/サービス（MSI）** — **1 つの `config.json`** に PBS 接続、
  バックアップ設定、**そして**スケジュール（`scheduled_jobs`）をまとめて記述します。ファイルを配置して
  `NimbusBackup` サービスを再起動すれば、ジョブは冪等に反映され、`nextRun` も
  自動で計算されます。Jinja2 テンプレートでタイムスタンプを計算する必要はありません。

サービスは設定を `C:\ProgramData\ProxmoxBackupClient\` から読み込みます
（0.4.0 以降。[アップグレード](#️-030-以前からのアップグレード)を参照）。

📖 詳しい手順：[Ansible で Windows の PBS へのバックアップを自動化](https://nimbus.rdem-systems.com/en/blog/unattended-windows-backup-ansible/?utm_source=github)。

## 🚀 クイックスタート

1. リリースページから `NimbusBackup.msi`（またはスタンドアロンの `NimbusBackup.exe`）をダウンロード
2. 管理者権限でインストール／実行（VSS に必要）
3. PBS への接続を設定してテスト
4. バックアップするディレクトリを選択
5. バックアップを開始 — またはスケジュールを設定

### 🔑 PBS ユーザーと権限

クライアントに必要なのは、対象データストアに対する **`DatastoreBackup`** ロールだけです — 管理者アカウントは不要です：

1. PBS の UI でユーザー（例：`nimbus@pbs`）と、そのユーザーの API トークン（例：`nimbus@pbs!laptop01`）を作成します。
2. **Datastore → Permissions**（または **Configuration → Access Control → Permissions**）で、`/datastore/<name>` に `DatastoreBackup` を付与します — 名前空間にバックアップする場合は `/datastore/<name>/<namespace>` に付与します。
3. **権限分離トークン**（「Privilege Separation」にチェック、デフォルト）：ロールはユーザーだけでなく**トークン**自体（`nimbus@pbs!laptop01`）にも付与してください — 実効権限は両者の共通部分になります。これが「permission denied」の最も一般的な原因です。

`DatastoreBackup` により、クライアントはバックアップを作成し、自身のバックアップグループを一覧表示・リストアできます。スナップショットの削除や prune には `DatastorePowerUser` が必要です。

## ⬆️ 0.3.0 以前からのアップグレード

既存のインストールに 0.4.0 以降をインストールすると、その場でアップグレードされます（MSI の識別情報も `NimbusBackup` サービスも同一）。データフォルダーは `C:\ProgramData\NimbusBackup` から共有の `C:\ProgramData\ProxmoxBackupClient` に移ります。初回起動時に、設定、スケジュールジョブ、履歴、API トークンが一度だけコピーされます（既存のファイルが上書きされることはありません）。旧フォルダーは `COPIED-TO-ProxmoxBackupClient.txt` で印を付けて残されるため、ダウングレードも引き続き可能です。旧バージョンで作成したスナップショットも、元の場所にリストアできます。

## 📋 動作要件

- Windows 10/11 または Windows Server（64 ビット）
- 管理者権限（VSS スナップショット用）
- Proxmox Backup Server へのネットワークアクセス

## 🔨 ソースからのビルド

### 前提条件
- Go 1.25 以降
- Node.js 20 以降
- Wails CLI：`go install github.com/wailsapp/wails/v2/cmd/wails@v2.13.0`（CI で使用しているバージョン）

### ビルド
```bash
cd gui
npm install --prefix frontend
wails build      # or: wails dev  (hot reload)
```

または Makefile ですべて（CLI + GUI + サービス）をビルドできます：`make install-deps && make`（`make help` を参照）。Windows のツールチェーンと Docker によるクロスビルド：[BUILD.md](BUILD.md)。

ブランドは実行ファイル名で決まります：`NimbusBackup.exe` は Nimbus Backup として、それ以外の名前では中立的な「Proxmox Backup Client」として動作します（[`gui/brand.go`](gui/brand.go)）。

## 🔗 アップストリームとの関係

Nimbus Backup は [tizbac/proxmoxbackupclient_go](https://github.com/tizbac/proxmoxbackupclient_go)（Tiziano Bacocco による Go 製の Proxmox Backup Client、GPLv3）のフォークとして始まり、私たちはそこに Windows GUI、サービス、スケジューリング、複数 PBS 対応、リストア機能を追加しました。2026 年 9 月、アップストリームはこの GUI を取り込み、ブランド中立化しました（「Proxmox Backup Client GUI」）。2026 年 10 月（0.4.1）、Nimbus Backup はアップストリームの最新コードをもとに再構築され、アップストリームのクライアント側暗号化を採用しました：**両プロジェクトは同じコードベースを共有しています。**

Nimbus Backup 0.4.1 は、アップストリームの `master` ブランチのコミット `3c1b989`（2026 年 10 月 9 日）をベースに構築されています。このコードを含むアップストリームのリリースはまだありません。最新のリリース v1.1.3（2026 年 5 月）は GUI の統合より前のものです。

このリポジトリがアップストリームに加えているのは、文書化されたパッチシリーズ（[`patches/`](patches/README.md)）で、その大部分はアップストリームに提案済みです：

- **機能**：ペーパーキーとキーのインポート（QR コード）、フォルダーの並列バックアップ、複数フォルダーのバックアップごとに 1 つの VSS スナップショット、コマンドラインでの除外指定、実マシンから生成される Proxmox VE の VM 構成。
- **アップストリームにまだマージされていない修正**（`proxmox-backup-client` の圧縮・暗号化されたバックアップのリストア、サービスによるバックアップで選択される PBS サーバー、PBS の拒否理由、ユーザー名/パスワード認証のサーバーからのリストア、ウィンドウの最大化など） — 順次アップストリームに送り、マージを待っています。
- **Nimbus Backup としての識別情報** — `NimbusBackup.exe`/`.msi`、`NimbusBackup` サービス、既存インストールの MSI アップグレードコード。これにより既存環境は引き続きその場でアップグレードされます。
- **アップグレードパス** — Nimbus Backup 0.3.0 以前からの移行（データフォルダーの移行、旧形式のスナップショットメタデータ）。
- **リリースパイプライン** — ビルド来歴証明、チェックサム、VirusTotal レポート。

私たちは定期的にアップストリームを再マージしています。アップストリーム独自のリリースは [tizbac/proxmoxbackupclient_go](https://github.com/tizbac/proxmoxbackupclient_go/releases) で公開されています。

## ⚠️ 免責事項

本ソフトウェアは現状のまま提供されます。信頼性の確保に努めていますが、データの損失や損害について一切の責任を負いません。本番環境で利用する前に、必ずバックアップをテストし、リストアを検証してください。

本プロジェクトは **Proxmox Server Solutions GmbH** とは**一切関係ありません**。「Proxmox」および関連する名称はそれぞれの所有者に帰属し、ここでは互換性を示す目的でのみ使用しています。

## 📄 ライセンス

GPLv3 — [LICENSE](LICENSE) ファイルを参照してください。

## RDEM Systems について

NimbusBackupClient は [RDEM Systems](https://www.rdem-systems.com/en/?utm_source=github) によって開発・保守されています。RDEM Systems は、Proxmox VE/PBS のマネージドサービスと NTP/NTS インフラを専門とするフランスのインフラプロバイダーです。私たちは [16 台の公開 NTS サーバー](https://ntp.rdem-systems.com/en/nts.php?utm_source=github)を運用しており（[稼働状況](https://ntp.rdem-systems.com/en/status.php?utm_source=github)。うち 11 台は[コミュニティのリファレンス](https://github.com/jauderho/nts-servers)に掲載されています）、自前でホストしたくないユーザー向けに[フルマネージドの PBS ホスティング](https://nimbus.rdem-systems.com/en/?utm_source=github)を提供しています。また、Proxmox が公式にサポートしていない Linux ディストリビューション向けの署名付き `proxmox-backup-client` パッケージである [unofficial-proxmox-backup-client](https://github.com/rdemsystems/unofficial-proxmox-backup-client) も保守しています。

---

**© 2024-2026 RDEM Systems および Proxmox Backup Client GO コントリビューター。**
