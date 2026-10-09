# Nimbus Backup — کلاینت Windows برای Proxmox Backup Server

[🇬🇧 English](README.md) · [🇫🇷 Français](README.fr.md) · [🇮🇹 Italiano](README.it.md) · [🇩🇪 Deutsch](README.de.md) · [🇪🇸 Español](README.es.md) · [🇷🇺 Русский](README.ru.md) · [🇨🇳 中文](README.zh.md) · [🇯🇵 日本語](README.ja.md) · [🇬🇷 Ελληνικά](README.el.md) · [🇷🇴 Română](README.ro.md) · [🇸🇪 Svenska](README.sv.md) · [🇸🇦 العربية](README.ar.md) · 🇮🇷 فارسی

> 🤖 این ترجمه توسط هوش مصنوعی از روی README انگلیسی تهیه شده است؛ در صورت هرگونه مغایرت، [نسخه انگلیسی](README.md) ملاک است.
> *AI-generated translation of the English README; the [English version](README.md) is authoritative.*

<div dir="rtl">

[![License](https://img.shields.io/badge/license-GPLv3-blue.svg)](LICENSE)
[![Release](https://img.shields.io/github/v/release/rdemsystems/NimbusBackupClient)](https://github.com/rdemsystems/NimbusBackupClient/releases)
[![Documentation](https://img.shields.io/badge/docs-nimbus.rdem--systems.com-orange)](https://nimbus.rdem-systems.com/en/?utm_source=github)

**Nimbus Backup یک کلاینت پشتیبان‌گیری متن‌باز (GPL-3.0) برای Windows است که با Proxmox Backup Server (PBS) کار می‌کند.**
یک رابط گرافیکی مدرن برای پشتیبان‌گیری از سرورها و ایستگاه‌های کاری Windows روی PBS — اسنپ‌شات‌های سازگار با VSS، کارهای زمان‌بندی‌شده، حالت‌های فایل و دیسک، مرور اسنپ‌شات‌ها و بازیابی، پشتیبانی از چند سرور PBS و یک سرویس Windows — به‌علاوه ابزارهای خط فرمان برای پشتیبان‌گیری از پوشه‌ها و کل ماشین. به دنبال فضای ذخیره‌سازی PBS **برون‌سایت و تغییرناپذیر** بدون میزبانی شخصی هستید؟ [سرویس مدیریت‌شده](#️-pbs-مدیریت-شده-برون-سایت-و-تغییرناپذیر) را در ادامه ببینید.

Nimbus Backup نسخه‌ی RDEM Systems از [Proxmox Backup Client GO](https://github.com/tizbac/proxmoxbackupclient_go) است: رابط گرافیکی‌ای که ما توسعه دادیم در پروژه‌ی اصلی (upstream) ادغام شده و اکنون هر دو پروژه از یک کد پایه‌ی مشترک استفاده می‌کنند (ببینید [رابطه با پروژه اصلی](#-رابطه-با-پروژه-اصلی-upstream)).

📖 **مستندات کامل، راهنمای نصب و میزبانی PBS:** [nimbus.rdem-systems.com](https://nimbus.rdem-systems.com/en/blog/backup-windows-proxmox-backup-server/?utm_source=github&utm_medium=readme&utm_campaign=nbc-readme-top)

## 📦 دانلود

👉 **[دانلود آخرین نسخه](https://github.com/rdemsystems/NimbusBackupClient/releases)**

هر نسخه شامل موارد زیر است:
- `NimbusBackup.msi` — نصب‌کننده (رابط گرافیکی + سرویس Windows)، **پیشنهادشده برای محیط عملیاتی**
- `NimbusBackup.exe` — رابط گرافیکی مستقل
- `nimbus-backup-cli-<version>-windows.zip` / `-linux.tar.gz` / `-macos.tar.gz` — ابزارهای خط فرمان
- `SHA256SUMS.txt` — چک‌سام‌ها

> ⚠️ **Windows پیام «ویروس شناسایی شد» (مثلاً `Trojan:Win32/Sabsik.FL.A!ml`) یا هشدار SmartScreen نشان می‌دهد؟**
> این یک **مثبت کاذب** شناخته‌شده برای برنامه‌های Go/Wails است — و ویروس *نیست*. پسوند `!ml` یعنی هشدار از یک مدل یادگیری ماشین می‌آید که فایل‌های اجرایی *امضانشده و کم‌رواج* را علامت‌گذاری می‌کند.
> بخوانید [چرا این اتفاق می‌افتد و چگونه فایل دانلودشده را بررسی کنید](https://nimbus.rdem-systems.com/en/antivirus-false-positive/?utm_source=github).

### 🔎 بررسی صحت هر فایل دانلودشده

هر نسخه همراه با چک‌سام‌های SHA-256 و یک **گواهی منشأ ساخت** (build-provenance attestation) امضاشده منتشر می‌شود (اثبات رمزنگاری‌شده‌ای که نشان می‌دهد فایل اجرایی توسط CI همین مخزن و از یک commit مشخص ساخته شده است):

</div>

```powershell
Get-FileHash .\NimbusBackup.msi -Algorithm SHA256   # compare against SHA256SUMS.txt
gh attestation verify .\NimbusBackup.msi --repo rdemsystems/NimbusBackupClient
```

<div dir="rtl">

**VirusTotal.** یادداشت‌های هر نسخه به گزارش VirusTotal نصب‌کننده‌ی همان build پیوند می‌دهند (فقط زمانی منتشر می‌شود که اسکن پاک باشد). گزارش‌های قبلی، 0 مورد شناسایی:
[0.2.108](https://www.virustotal.com/gui/file/6e8fb7ce9af740d470e947addb8daba4331c0b88e8bfdec9e0697ea8f7f29e9e/detection) ·
[0.2.107](https://www.virustotal.com/gui/file/6fd6c6fa77e0305c129ef882a3745100aa6033187a6d52a4af94149ab6b666d2/detection) ·
[0.2.106](https://www.virustotal.com/gui/file/ad6e56700ed9df8e088906e38cee2e2882fc7045f4e39269de0e379a01784ad7/detection)

> ℹ️ **امضای کد:** فایل‌های اجرایی Windows **هنوز با Authenticode امضا نشده‌اند** و همین موضوع باعث هشدارهای SmartScreen / `!ml` بالا می‌شود. درخواست ما برای یک گواهی رایگان متن‌باز از [SignPath Foundation](https://signpath.org) بی‌پاسخ ماند؛ امضا از طریق Azure Artifact Signing در حال راه‌اندازی است و هدف آن نسخه‌ی **0.4.1** است. تا آن زمان، منشأ فایل‌ها از طریق گواهی منشأ ساخت و چک‌سام‌های بالا اثبات می‌شود.

### 🐧 روی Linux؟ از کلاینت رسمی استفاده کنید

رابط گرافیکی Nimbus Backup فقط برای Windows است (ابزارهای خط فرمان برای Linux و macOS هم ساخته می‌شوند). برای پشتیبان‌گیری در سطح فایل روی Linux، از `proxmox-backup-client` خود Proxmox استفاده کنید — ما آن را برای توزیع‌هایی که Proxmox پوشش نمی‌دهد بسته‌بندی می‌کنیم:

👉 **[rdemsystems/unofficial-proxmox-backup-client](https://github.com/rdemsystems/unofficial-proxmox-backup-client)** — مخازن بسته‌ی امضاشده برای Debian/Ubuntu، Fedora/RHEL/Rocky/AlmaLinux، Arch و Alpine (amd64 و arm64). فایل اجرایی ایستای رسمی Proxmox، بدون هیچ تغییری بازبسته‌بندی شده — نه وصله شده و نه دوباره کامپایل شده.

## ☁️ PBS مدیریت شده (برون سایت و تغییرناپذیر)

نمی‌خواهید Proxmox Backup Server را خودتان میزبانی کنید؟ از دیتااستورهای PBS کاملاً مدیریت‌شده‌ی ما، **برون‌سایت و تغییرناپذیر**، استفاده کنید:
👉 **[پشتیبان‌گیری خود را پیکربندی کنید و قیمت‌ها را ببینید](https://nimbus.rdem-systems.com/en/choose-backup/?utm_source=github)**

- ✅ از 12 یورو به ازای هر ترابایت در ماه
- ✅ 1 ترابایت آزمایش رایگان
- ✅ یک [مقصد برون‌سایت آماده برای PBS شما](https://nimbus.rdem-systems.com/en/offsite-proxmox-backup/?utm_source=github) — دیتااستورهای ایزوله (air-gapped) و تغییرناپذیر
- ✅ [NimbusBackup — میزبانی مدیریت‌شده‌ی PBS در فرانسه](https://nimbus.rdem-systems.com/en/?utm_source=github)

## 📚 مستندات

- **راهنمای کامل Proxmox Backup** — بهترین روش‌های استقرار PBS ([🇬🇧 EN](https://nimbus.rdem-systems.com/en/blog/complete-proxmox-backup-guide/?utm_source=github))
- **پشتیبان‌گیری از Windows با Proxmox Backup Server** — راهنمای استقرار مخصوص Windows ([🇬🇧 EN](https://nimbus.rdem-systems.com/en/blog/backup-windows-proxmox-backup-server/?utm_source=github))
- **PBS در برابر Veeam** — مقایسه با Proxmox Backup Server ([🇬🇧 EN](https://nimbus.rdem-systems.com/en/blog/pbs-vs-veeam-proxmox-backup-comparison/?utm_source=github))
- در همین مخزن: [راهنمای کاربر چند سرور PBS](MULTI_PBS_USER_GUIDE.md) · [نمونه‌های استقرار خودکار](examples/automation/) · [بازیابی کامل سیستم (bare-metal) با Clonezilla](PATCH-CLONEZILLA.md) · [تاریخچه تغییرات](CHANGELOG.md)

## ✨ قابلیت‌ها

### رابط گرافیکی (پیشنهادی)
- **🌍 چندزبانه** — رابط کاربری به زبان‌های انگلیسی، فرانسوی، ایتالیایی، آلمانی و لهستانی
- پیکربندی کاربرپسند همراه با آزمایش اتصال (توکن API یا نام کاربری/گذرواژه)
- نمایش لحظه‌ای پیشرفت پشتیبان‌گیری با سرعت و زمان باقی‌مانده، با امکان لغو در هر لحظه
- پشتیبانی از VSS (Volume Shadow Copy) برای پشتیبان‌های سازگار
- پشتیبان‌گیری از چند پوشه، حالت‌های فایل و دیسک (کل ماشین)؛ پوشه‌ها می‌توانند به‌صورت موازی پردازش شوند (پیشنهاد: تعداد CPUها / 4)
- مرور اسنپ‌شات‌ها، جستجوی فایل (با کاراکترهای عام) و بازیابی
- پشتیبانی از چند سرور PBS، پین کردن اثر انگشت گواهی (TOFU)
- **🔒 رمزنگاری سمت کلاینت** (AES-256-GCM)، با فایل‌های کلید سازگار با `proxmox-backup-client` و Proxmox VE
- حالت سرویس Windows + پشتیبان‌گیری زمان‌بندی‌شده، تاریخچه‌ی پشتیبان‌گیری با اجرای مجدد با یک کلیک
- لاگ‌های اشکال‌زدایی (debug) برای عیب‌یابی

### ابزارهای خط فرمان
- `proxmoxbackup-directory` — پشتیبان‌گیری از پوشه‌ها (PXAR) با حذف داده‌های تکراری، پشتیبان‌گیری از جریان (`-backupstream`، مثلاً یک pipe از `mysqldump`)، استثناها (`-exclude "*.tmp"`، قابل تکرار، یا `-exclude-from file`؛ `"exclude"` در فایل پیکربندی JSON)، چند پوشه به‌طور هم‌زمان (`-parallel N`، پیشنهاد: تعداد CPUها / 4)، اعلان‌های ایمیلی، فایل پیکربندی JSON
- `proxmoxbackup-machine` — پشتیبان‌گیری کامل از ماشین در حال اجرا به‌صورت ایمیج دیسک قابل بوت (FIDX): VSS روی Windows، افزایشی، هش‌گیری موازی
- `proxmoxbackup-nbd` — سرور NBD برای mount کردن پشتیبان دیسک روی Linux (بازیابی در سطح فایل، بازیابی کامل سیستم از یک [ایمیج Clonezilla live ISO وصله‌شده](PATCH-CLONEZILLA.md))

### 📸 تصاویر

![Server configuration](docs/screenshots/nimbus-gui-liste-servers.png)
*مدیریت چند سرور PBS با نشانگرهای وضعیت*

![Add server form](docs/screenshots/nimbus-gui-add-server-form.png)
*پیکربندی آسان سرور همراه با آزمایش اتصال*

![One-shot backup](docs/screenshots/nimbus-gui-one-shot-backup.png)
*پیشرفت لحظه‌ای پشتیبان‌گیری با زمان باقی‌مانده و سرعت*

### استثناهای هوشمند سیستمی (حالت فایل)
هنگام پشتیبان‌گیری از کل یک درایو (مثلاً `D:\`)، Nimbus Backup به‌طور خودکار موارد زیر را مستثنا می‌کند:

**پوشه‌های سیستمی:** `System Volume Information` (فضای ذخیره‌سازی VSS، که می‌تواند بیش از 100 گیگابایت باشد)، `$RECYCLE.BIN`، `Recovery`.
**فایل‌های سیستمی:** `pagefile.sys`، `hiberfil.sys`، `swapfile.sys`.

**چرا مهم است:** ممکن است یک درایو 1.03 ترابایت فضای اشغال‌شده نشان دهد در حالی که حجم واقعی فایل‌ها حدود 141 گیگابایت است. بدون استثناها، پشتیبان شامل اسنپ‌شات‌های VSS می‌شود (هدر رفتن فضا و زمان)؛ با آن‌ها، حجم پشتیبان با داده‌های واقعی برابر است.

**توصیه:** برای پشتیبان‌گیری در سطح فایل از **حالت فایل** (پیش‌فرض) همراه با استثناهای خودکار استفاده کنید؛ برای بازیابی کامل سیستم (bare-metal) از **حالت دیسک** در یک کار جداگانه استفاده کنید (همه‌چیز را شامل می‌شود).

### امنیت و کیفیت
- اعتبارسنجی ورودی‌ها و پاک‌سازی اطلاعات احراز هویت (حذف اسرار از لاگ‌ها)
- جلوگیری از پیمایش مسیر (path traversal)
- منطق تلاش مجدد با عقب‌نشینی نمایی (exponential backoff)
- دروازه‌های CI در هر build: تست‌ها، `golangci-lint`، `gosec`، `go mod tidy`

### 🔒 رمزنگاری سمت کلاینت
پشتیبان‌ها را می‌توان **روی کلاینت** و پیش از خروج از ماشین رمزنگاری کرد، با
همان طرحی که `proxmox-backup-client` رسمی به کار می‌برد (AES-256-GCM، و چکیده‌ی
قطعه‌ها وابسته به کلید). سرور PBS فقط داده‌های غیرقابل‌خواندن را ذخیره می‌کند و هرگز
کلید را نمی‌بیند — که روی یک PBS اشتراکی یا مدیریت‌شده مفید است.

رمزگذاری را Tiziano Bacocco در پروژهٔ اصلی توسعه داده است ([tizbac/proxmoxbackupclient_go](https://github.com/tizbac/proxmoxbackupclient_go))؛ کلید کاغذی، خروجی کد QR و وارد کردن کلید افزوده‌های Nimbus Backup هستند.

- **رابط گرافیکی**: *Servers → Edit → Encryption key* — یک فایل کلید بسازید یا
  یک فایل موجود را انتخاب کنید (ساخته‌شده با `proxmox-backup-client key create --kdf none` یا از یک
  فضای ذخیره‌سازی PVE)؛ اثر انگشت آن نمایش داده می‌شود. سپس یک نسخه از آن را بیرون از ماشین نگه دارید:
  **Print (paper key)** یک صفحه‌ی قابل چاپ حاوی کلید و کد QR آن ذخیره می‌کند (با
  قالب `proxmox-backup-client key paperkey`)، که در صورت تمایل با عبارت عبور محافظت می‌شود.
  **Import a key from text or a QR code** یک کد QR اسکن‌شده یا یک کلید کاغذی را
  دوباره به فایل کلید تبدیل می‌کند. رابط گرافیکی فقط از فایل‌های کلید بدون محافظت استفاده می‌کند: وارد کردن
  کلیدی که با عبارت عبور محافظت شده، قفل آن را باز می‌کند و فایل کلید جدید را **بدون**
  عبارت عبور ذخیره می‌کند — از آن فایل به همان اندازه‌ی خود کلید محافظت کنید.
- **خط فرمان**: `-keyfile path/to/key.json` (یا `"keyfile"` در فایل پیکربندی JSON)؛ برای
  کلید محافظت‌شده با عبارت عبور از `-keyfile-passphrase` استفاده کنید، وگرنه عبارت عبور پرسیده می‌شود.
- **بازیابی کامل سیستم (bare-metal)**: ایمیج Clonezilla ISO وصله‌شده پشتیبان‌های
  رمزنگاری‌شده‌ی دیسک را هم بازیابی می‌کند (کلید از روی حافظه‌ی USB، به‌همراه عبارت عبور آن در صورت وجود) — ببینید
  [PATCH-CLONEZILLA.md](PATCH-CLONEZILLA.md).
- **سازگار با `proxmox-backup-client`، که در هر build در CI**
  روی یک PBS واقعی بررسی می‌شود: پوشه‌ای که Nimbus Backup با کلیدی ساخته‌شده توسط
  `proxmox-backup-client` رمزنگاری کرده، توسط کلاینت رسمی با همان کلید بازیابی می‌شود
  (و بدون آن رد می‌شود)، و پشتیبان رمزنگاری‌شده‌ی دیسکی که کلاینت رسمی ساخته،
  توسط Nimbus Backup خوانده می‌شود. Proxmox VE نیز از همین فایل‌های کلید استفاده می‌کند.

> ⚠️ **بدون کلید، پشتیبان‌های رمزنگاری‌شده قابل بازیابی نیستند.** نخستین
> پشتیبان رمزنگاری‌شده همه‌چیز را دوباره بارگذاری می‌کند (بدون حذف داده‌های تکراری با پشتیبان‌های
> رمزنگاری‌نشده). شناسه‌های پشتیبان، نام آرشیوها و حجم‌ها برای سرور قابل مشاهده می‌مانند؛ محتوای
> فایل‌ها، نام فایل‌ها و کاتالوگ رمزنگاری می‌شوند.

## 🤖 استقرار خودکار (Ansible و IaC)

Nimbus Backup به‌طور کامل از طریق فایل‌ها نصب و پیکربندی می‌شود — بدون هیچ راه‌اندازی تعاملی.
دو مسیر وجود دارد که هر دو با نمونه‌های آماده‌ی استفاده در
[`examples/automation/`](examples/automation/) پوشش داده شده‌اند:

- **خط فرمان** — یک فایل JSON برای هر میزبان، کل تنظیمات پشتیبان‌گیری را در بر دارد
  (`proxmoxbackup-directory.exe --config file.json`)؛ زمان‌بندی از طریق Task Scheduler
  در Windows. کدهای خروج قابل اعتماد (`0` موفق، `1` خطای مهلک، `2` قفل‌شده، `3` ناقص)
  تا ابزار ارکستراسیون شما شکست‌ها را تشخیص دهد. بهترین گزینه برای زیرساخت به‌عنوان کد (IaC) خالص.
- **رابط گرافیکی/سرویس (MSI)** — یک **فایل `config.json` واحد** اتصال PBS،
  تنظیمات پشتیبان‌گیری **و** زمان‌بندی (`scheduled_jobs`) را در بر دارد. فایل را منتشر کنید، سرویس
  `NimbusBackup` را راه‌اندازی مجدد کنید: کارها به‌صورت idempotent همگام‌سازی می‌شوند و `nextRun`
  برای شما محاسبه می‌شود. نیازی به محاسبه‌ی timestamp در قالب Jinja2 شما نیست.

سرویس پیکربندی خود را از `C:\ProgramData\ProxmoxBackupClient\` می‌خواند
(از نسخه‌ی 0.4.0؛ ببینید [ارتقا](#️-ارتقا-از-نسخه--030)).

📖 راهنمای کامل گام‌به‌گام: [Automate Windows backup to PBS with Ansible](https://nimbus.rdem-systems.com/en/blog/unattended-windows-backup-ansible/?utm_source=github).

## 🚀 شروع سریع

1. `NimbusBackup.msi` (یا `NimbusBackup.exe` مستقل) را از صفحه‌ی نسخه‌ها دانلود کنید
2. آن را با دسترسی مدیر (administrator) نصب / اجرا کنید (برای VSS لازم است)
3. اتصال PBS خود را پیکربندی و آزمایش کنید
4. پوشه‌هایی را که می‌خواهید پشتیبان‌گیری شوند انتخاب کنید
5. پشتیبان‌گیری را شروع کنید — یا آن را زمان‌بندی کنید

### 🔑 کاربر PBS و مجوزها

کلاینت فقط به نقش **`DatastoreBackup`** روی دیتااستور مقصد نیاز دارد — بدون حساب مدیر:

1. در رابط کاربری PBS، یک کاربر (مثلاً `nimbus@pbs`) و یک توکن API برای آن (مثلاً `nimbus@pbs!laptop01`) بسازید.
2. در **Datastore → Permissions** (یا **Configuration → Access Control → Permissions**)، نقش `DatastoreBackup` را روی `/datastore/<name>` اعطا کنید — یا روی `/datastore/<name>/<namespace>` اگر در یک namespace پشتیبان‌گیری می‌کنید.
3. **توکن با تفکیک امتیازات** (گزینه‌ی "Privilege Separation" فعال، که پیش‌فرض است): نقش را به **خود توکن** (`nimbus@pbs!laptop01`) اعطا کنید، نه فقط به کاربر — حقوق مؤثر، اشتراک هر دو است. این رایج‌ترین علت خطای "permission denied" است.

`DatastoreBackup` به کلاینت اجازه می‌دهد پشتیبان بسازد و گروه‌های پشتیبان خودش را فهرست و بازیابی کند. حذف یا هرس (prune) اسنپ‌شات‌ها به `DatastorePowerUser` نیاز دارد.

## ⬆️ ارتقا از نسخه ≤ 0.3.0

نصب نسخه‌ی 0.4.0 یا جدیدتر روی یک نصب موجود، آن را درجا ارتقا می‌دهد (همان هویت MSI، همان سرویس `NimbusBackup`). پوشه‌ی داده از `C:\ProgramData\NimbusBackup` به پوشه‌ی مشترک `C:\ProgramData\ProxmoxBackupClient` منتقل می‌شود: در نخستین اجرا، پیکربندی، کارهای زمان‌بندی‌شده، تاریخچه و توکن API شما یک بار کپی می‌شوند (فایل‌های موجود هرگز بازنویسی نمی‌شوند). پوشه‌ی قدیمی با نشانه‌ی `COPIED-TO-ProxmoxBackupClient.txt` حفظ می‌شود تا بازگشت به نسخه‌ی قبلی همچنان ممکن باشد. اسنپ‌شات‌هایی که با نسخه‌های قدیمی‌تر گرفته شده‌اند همچنان قابل بازیابی در محل اصلی خود هستند.

## 📋 پیش‌نیازها

- Windows 10/11 یا Windows Server (64 بیتی)
- دسترسی مدیر (برای اسنپ‌شات‌های VSS)
- دسترسی شبکه به یک Proxmox Backup Server

## 🔨 ساخت از سورس

### پیش‌نیازها
- Go 1.25 یا جدیدتر
- Node.js 20 یا جدیدتر
- Wails CLI: `go install github.com/wailsapp/wails/v2/cmd/wails@v2.13.0` (نسخه‌ای که CI استفاده می‌کند)

### ساخت

</div>

```bash
cd gui
npm install --prefix frontend
wails build      # or: wails dev  (hot reload)
```

<div dir="rtl">

یا همه‌چیز (خط فرمان + رابط گرافیکی + سرویس) را با Makefile بسازید: `make install-deps && make` (ببینید `make help`). ابزارهای ساخت در Windows و ساخت متقاطع با Docker: [BUILD.md](BUILD.md).

برند بر اساس نام فایل اجرایی انتخاب می‌شود: `NimbusBackup.exe` به‌عنوان Nimbus Backup اجرا می‌شود و هر نام دیگری به‌عنوان "Proxmox Backup Client" خنثی ([`gui/brand.go`](gui/brand.go)).

## 🔗 رابطه با پروژه اصلی (upstream)

Nimbus Backup به‌عنوان یک fork از [tizbac/proxmoxbackupclient_go](https://github.com/tizbac/proxmoxbackupclient_go) (کلاینت Proxmox Backup به زبان Go، نوشته‌ی Tiziano Bacocco، با مجوز GPLv3) آغاز شد و ما رابط گرافیکی Windows، سرویس، زمان‌بندی، پشتیبانی از چند سرور PBS و بازیابی را به آن افزودیم. در سپتامبر 2026، پروژه‌ی اصلی آن رابط گرافیکی را ادغام کرد و آن را از نظر برند خنثی کرد ("Proxmox Backup Client GUI"). از نسخه‌ی 0.4.0، Nimbus Backup از همان کد پایه ساخته می‌شود: **اکنون دو پروژه از نظر عملکرد تقریباً یکسان هستند.**

آنچه این مخزن به پروژه‌ی اصلی اضافه می‌کند، مجموعه‌ای کوچک و مستند از وصله‌هاست ([`patches/`](patches/README.md)):

- **اصلاحاتی که هنوز در پروژه‌ی اصلی ادغام نشده‌اند** (ساخت سرویس، بازیابی با سرورهای دارای نام کاربری/گذرواژه، کدهای خروج، حذف اطلاعات حساس از لاگ‌ها، قابلیت اطمینان پشتیبان‌گیری از ماشین…) — که به‌تدریج با ادغامشان به پروژه‌ی اصلی ارسال می‌شوند.
- **هویت Nimbus Backup** — `NimbusBackup.exe`/`.msi`، سرویس `NimbusBackup` و کد ارتقای MSI نصب‌های موجود، تا همچنان درجا ارتقا یابند.
- **مسیر ارتقا** از Nimbus Backup ≤ 0.3.0 (مهاجرت پوشه‌ی داده، فراداده‌ی اسنپ‌شات‌های قدیمی).
- **خط انتشار نسخه‌ها** — گواهی منشأ ساخت، چک‌سام‌ها، گزارش‌های VirusTotal.

ما به‌طور منظم پروژه‌ی اصلی را دوباره ادغام می‌کنیم؛ نسخه‌های خود پروژه‌ی اصلی در [tizbac/proxmoxbackupclient_go](https://github.com/tizbac/proxmoxbackupclient_go/releases) منتشر می‌شوند.

## ⚠️ سلب مسئولیت

این نرم‌افزار «همان‌گونه که هست» ارائه می‌شود. با وجود تلاش ما برای قابلیت اطمینان، هیچ مسئولیتی در قبال از دست رفتن داده یا خسارت نمی‌پذیریم. همیشه پشتیبان‌های خود را آزمایش کنید و پیش از اتکا به آن‌ها در محیط عملیاتی، بازیابی را بررسی کنید.

این پروژه **هیچ وابستگی‌ای** به **Proxmox Server Solutions GmbH** ندارد. "Proxmox" و نام‌های مرتبط متعلق به صاحبان مربوطه‌شان هستند و در اینجا فقط برای بیان سازگاری استفاده شده‌اند.

## 📄 مجوز

GPLv3 — فایل [LICENSE](LICENSE) را ببینید.

## درباره RDEM Systems

NimbusBackupClient توسط [RDEM Systems](https://www.rdem-systems.com/en/?utm_source=github) توسعه داده و نگهداری می‌شود؛ یک ارائه‌دهنده‌ی زیرساخت فرانسوی که در سرویس‌های مدیریت‌شده‌ی Proxmox VE/PBS و زیرساخت NTP/NTS تخصص دارد. ما [16 سرور عمومی NTS](https://ntp.rdem-systems.com/en/nts.php?utm_source=github) را اداره می‌کنیم ([وضعیت زنده](https://ntp.rdem-systems.com/en/status.php?utm_source=github)؛ 11 مورد از آن‌ها در [فهرست مرجع جامعه](https://github.com/jauderho/nts-servers) آمده‌اند) و برای کاربرانی که نمی‌خواهند خودشان میزبانی کنند، [میزبانی کاملاً مدیریت‌شده‌ی PBS](https://nimbus.rdem-systems.com/en/?utm_source=github) ارائه می‌دهیم. همچنین [unofficial-proxmox-backup-client](https://github.com/rdemsystems/unofficial-proxmox-backup-client) را نگهداری می‌کنیم: بسته‌های امضاشده‌ی `proxmox-backup-client` برای توزیع‌های Linux که Proxmox به‌طور رسمی پشتیبانی نمی‌کند.

---

**© 2024-2026 RDEM Systems and Proxmox Backup Client GO contributors.**

</div>
