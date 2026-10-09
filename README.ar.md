# Nimbus Backup — عميل Windows لـ Proxmox Backup Server

[🇬🇧 English](README.md) · [🇫🇷 Français](README.fr.md) · [🇮🇹 Italiano](README.it.md) · [🇩🇪 Deutsch](README.de.md) · [🇪🇸 Español](README.es.md) · [🇷🇺 Русский](README.ru.md) · [🇨🇳 中文](README.zh.md) · [🇯🇵 日本語](README.ja.md) · [🇬🇷 Ελληνικά](README.el.md) · [🇷🇴 Română](README.ro.md) · [🇸🇪 Svenska](README.sv.md) · 🇸🇦 العربية · [🇮🇷 فارسی](README.fa.md)

> 🤖 أُنتجت هذه الترجمة بواسطة الذكاء الاصطناعي انطلاقًا من ملف README الإنجليزي؛ وفي حال وجود أي اختلاف، تكون [النسخة الإنجليزية](README.md) هي المرجع.
> *AI-generated translation of the English README; the [English version](README.md) is authoritative.*

<div dir="rtl">

[![License](https://img.shields.io/badge/license-GPLv3-blue.svg)](LICENSE)
[![Release](https://img.shields.io/github/v/release/rdemsystems/NimbusBackupClient)](https://github.com/rdemsystems/NimbusBackupClient/releases)
[![Documentation](https://img.shields.io/badge/docs-nimbus.rdem--systems.com-orange)](https://nimbus.rdem-systems.com/en/?utm_source=github)

**Nimbus Backup هو عميل نسخ احتياطي مفتوح المصدر (GPL-3.0) لنظام Windows، مخصص لـ Proxmox Backup Server (PBS).**
واجهة رسومية حديثة لنسخ خوادم Windows ومحطات العمل احتياطيًا إلى PBS — لقطات متسقة عبر VSS، ومهام مجدولة، ووضعا الملفات والقرص، وتصفح اللقطات والاستعادة، ودعم عدة خوادم PBS، وخدمة Windows — إضافةً إلى أدوات سطر أوامر لنسخ المجلدات احتياطيًا ونسخ الجهاز بالكامل. هل تبحث عن تخزين PBS **خارج الموقع وغير قابل للتغيير** دون استضافته بنفسك؟ اطّلع على [الخدمة المُدارة](#️-pbs-مدار-خارج-الموقع-وغير-قابل-للتغيير) أدناه.

Nimbus Backup هو إصدار RDEM Systems من [Proxmox Backup Client GO](https://github.com/tizbac/proxmoxbackupclient_go): فقد دُمجت الواجهة الرسومية التي طوّرناها في المشروع الأصلي (upstream)، وأصبح المشروعان يتشاركان الآن قاعدة الشيفرة نفسها (راجع [العلاقة مع المشروع الأصلي](#-العلاقة-مع-المشروع-الأصلي-upstream)).

📖 **التوثيق الكامل ودليل التثبيت واستضافة PBS:** [nimbus.rdem-systems.com](https://nimbus.rdem-systems.com/en/blog/backup-windows-proxmox-backup-server/?utm_source=github&utm_medium=readme&utm_campaign=nbc-readme-top)

## 📦 التنزيل

👉 **[تنزيل أحدث إصدار](https://github.com/rdemsystems/NimbusBackupClient/releases)**

يتضمن كل إصدار:
- `NimbusBackup.msi` — برنامج التثبيت (الواجهة الرسومية + خدمة Windows)، **موصى به لبيئات الإنتاج**
- `NimbusBackup.exe` — واجهة رسومية مستقلة
- `nimbus-backup-cli-<version>-windows.zip` / `-linux.tar.gz` / `-macos.tar.gz` — أدوات سطر الأوامر
- `SHA256SUMS.txt` — المجاميع الاختبارية

> ⚠️ **هل يُظهر Windows رسالة "تم اكتشاف فيروس" (مثل `Trojan:Win32/Sabsik.FL.A!ml`) أو تحذيرًا من SmartScreen؟**
> هذه **إيجابية كاذبة** معروفة لتطبيقات Go/Wails — وهي *ليست* فيروسًا. تعني اللاحقة `!ml` أن التنبيه صادر عن نموذج تعلّم آلي يُعلِّم الملفات التنفيذية *قليلة الانتشار* (وحتى الإصدار 0.4.0، غير الموقّعة أيضًا).
> اقرأ [سبب حدوث ذلك وكيفية التحقق من الملف الذي نزّلته](https://nimbus.rdem-systems.com/en/antivirus-false-positive/?utm_source=github).

### 🔎 التحقق من أي ملف تم تنزيله

يأتي كل إصدار مع مجاميع SHA-256 الاختبارية و**شهادة مصدر البناء** (build-provenance attestation) موقّعة (دليل تشفيري على أن الملف التنفيذي أُنتج بواسطة CI الخاص بهذا المستودع، انطلاقًا من commit محدد):

</div>

```powershell
Get-FileHash .\NimbusBackup.msi -Algorithm SHA256   # compare against SHA256SUMS.txt
gh attestation verify .\NimbusBackup.msi --repo rdemsystems/NimbusBackupClient
```

<div dir="rtl">

**VirusTotal.** تتضمن ملاحظات كل إصدار رابطًا إلى تقرير VirusTotal لبرنامج التثبيت الخاص بذلك البناء (يُنشر فقط عندما يكون الفحص نظيفًا). تقارير سابقة، 0 اكتشافات:
[0.2.108](https://www.virustotal.com/gui/file/6e8fb7ce9af740d470e947addb8daba4331c0b88e8bfdec9e0697ea8f7f29e9e/detection) ·
[0.2.107](https://www.virustotal.com/gui/file/6fd6c6fa77e0305c129ef882a3745100aa6033187a6d52a4af94149ab6b666d2/detection) ·
[0.2.106](https://www.virustotal.com/gui/file/ad6e56700ed9df8e088906e38cee2e2882fc7045f4e39269de0e379a01784ad7/detection)

> 🔏 **توقيع الشيفرة:** منذ الإصدار 0.4.1، أصبحت `NimbusBackup.exe` وخدمته و`NimbusBackup.msi` **موقّعة بتوقيع Authenticode من RDEM SYSTEMS** (Azure Artifact Signing)؛ وتعرض *الخصائص ← التوقيعات الرقمية* اسم الناشر. قد يستمر SmartScreen في التحذير عند صدور إصدار جديد إلى أن تترسّخ سمعته: تحقّق من أن الناشر هو RDEM SYSTEMS، ثم اختر *مزيد من المعلومات ← التشغيل على أي حال*. أدوات سطر الأوامر ليست موقّعة بعد؛ وتغطي شهادة مصدر البناء والمجاميع الاختبارية المذكورة أعلاه جميع الملفات.

### 🐧 على Linux؟ استخدم العميل الرسمي

الواجهة الرسومية لـ Nimbus Backup متاحة لنظام Windows فقط (أما أدوات سطر الأوامر فتُبنى أيضًا لـ Linux وmacOS). للنسخ الاحتياطي على مستوى الملفات في Linux، استخدم `proxmox-backup-client` الخاص بـ Proxmox نفسه — ونحن نحزّمه للتوزيعات التي لا تغطيها Proxmox:

👉 **[rdemsystems/unofficial-proxmox-backup-client](https://github.com/rdemsystems/unofficial-proxmox-backup-client)** — مستودعات حزم موقّعة لـ Debian/Ubuntu وFedora/RHEL/Rocky/AlmaLinux وArch وAlpine (amd64 وarm64). الملف التنفيذي الثابت الرسمي من Proxmox، مُعاد تحزيمه دون أي تغيير — لا تعديل عليه ولا إعادة ترجمة.

## ☁️ PBS مدار (خارج الموقع وغير قابل للتغيير)

لا ترغب في استضافة Proxmox Backup Server بنفسك؟ استخدم مخازن بيانات PBS المُدارة بالكامل لدينا، **خارج الموقع وغير القابلة للتغيير**:
👉 **[اضبط نسختك الاحتياطية واطّلع على الأسعار](https://nimbus.rdem-systems.com/en/choose-backup/?utm_source=github)**

- ✅ ابتداءً من 12 €/تيرابايت/شهريًا
- ✅ تجربة مجانية بسعة 1 تيرابايت
- ✅ [وجهة خارج الموقع لـ PBS الخاص بك](https://nimbus.rdem-systems.com/en/offsite-proxmox-backup/?utm_source=github) جاهزة للاستخدام — مخازن بيانات معزولة (air-gapped) وغير قابلة للتغيير
- ✅ [NimbusBackup — استضافة PBS مُدارة في فرنسا](https://nimbus.rdem-systems.com/en/?utm_source=github)

## 📚 التوثيق

- **الدليل الكامل لـ Proxmox Backup** — أفضل ممارسات نشر PBS ([🇬🇧 EN](https://nimbus.rdem-systems.com/en/blog/complete-proxmox-backup-guide/?utm_source=github))
- **نسخ Windows احتياطيًا باستخدام Proxmox Backup Server** — دليل نشر خاص بـ Windows ([🇬🇧 EN](https://nimbus.rdem-systems.com/en/blog/backup-windows-proxmox-backup-server/?utm_source=github))
- **PBS مقابل Veeam** — مقارنة مع Proxmox Backup Server ([🇬🇧 EN](https://nimbus.rdem-systems.com/en/blog/pbs-vs-veeam-proxmox-backup-comparison/?utm_source=github))
- في هذا المستودع: [دليل المستخدم لعدة خوادم PBS](MULTI_PBS_USER_GUIDE.md) · [أمثلة النشر غير التفاعلي](examples/automation/) · [الاستعادة الكاملة للنظام (bare-metal) باستخدام Clonezilla](PATCH-CLONEZILLA.md) · [سجل التغييرات](CHANGELOG.md)

## ✨ الميزات

### الواجهة الرسومية (موصى بها)
- **🌍 متعددة اللغات** — واجهة بالإنجليزية والفرنسية والإيطالية والألمانية والبولندية
- إعداد سهل الاستخدام مع اختبار الاتصال (رمز API أو اسم المستخدم/كلمة المرور)
- عرض تقدّم النسخ الاحتياطي في الوقت الفعلي مع السرعة والوقت المتبقي المقدّر، وإمكانية الإلغاء في أي وقت، وتبويب للمهام قيد التشغيل
- VSS (Volume Shadow Copy) لنسخ احتياطية متسقة: لقطة واحدة لنسخة احتياطية كاملة متعددة المجلدات، ونسخة ظل واحدة لكل وحدة تخزين
- نسخ احتياطي لعدة مجلدات، ووضعا الملفات والقرص (الجهاز بالكامل)؛ ويمكن معالجة المجلدات بالتوازي (الموصى به: عدد المعالجات / 4)
- تُستعاد النسخ الاحتياطية للأقراص في Proxmox VE كآلة افتراضية مطابقة للجهاز (المعالجات، وذاكرة RAM، والبرنامج الثابت، وبطاقات الشبكة بعناوين MAC الخاصة بها، ومعرّف VM مخصّص)
- تصفح اللقطات، والبحث عن الملفات (بأحرف البدل) والاستعادة، بما في ذلك قوائم NTFS ACL
- دعم عدة خوادم PBS مع خادم PBS لكل مهمة، وتثبيت بصمة الشهادة (TOFU)
- **🔒 تشفير من جانب العميل** (AES-256-GCM)، بملفات مفاتيح متوافقة مع `proxmox-backup-client` وProxmox VE
- وضع خدمة Windows + نسخ احتياطية مجدولة، وسجل للنسخ الاحتياطية مع إعادة التشغيل بنقرة واحدة
- سجلات تصحيح (debug) لاستكشاف الأخطاء وإصلاحها

### أدوات سطر الأوامر
- `proxmoxbackup-directory` — نسخ احتياطي للمجلدات (PXAR) مع إزالة التكرار، ونسخ احتياطي للتدفقات (`-backupstream`، مثل أنبوب `mysqldump`)، والاستبعادات (`-exclude "*.tmp"`، قابل للتكرار، أو `-exclude-from file`؛ `"exclude"` في ملف إعداد JSON)، وعدة مجلدات في آن واحد (`-parallel N`، الموصى به: عدد المعالجات / 4)، وإشعارات البريد الإلكتروني، وملف إعداد JSON
- `proxmoxbackup-machine` — نسخ احتياطي كامل للجهاز أثناء التشغيل في صورة قرص قابلة للإقلاع (FIDX): VSS على Windows، تزايدي، مع حساب التجزئة بالتوازي
- `proxmoxbackup-nbd` — خادم NBD لتركيب نسخة احتياطية لقرص على Linux (استعادة على مستوى الملفات، واستعادة كاملة للنظام من [صورة Clonezilla live ISO معدّلة](PATCH-CLONEZILLA.md))

### 📸 لقطات الشاشة

![Server configuration](docs/screenshots/nimbus-gui-liste-servers.png)
*إدارة عدة خوادم PBS مع مؤشرات الحالة*

![Add server form](docs/screenshots/nimbus-gui-add-server-form.png)
*إعداد سهل للخادم مع اختبار الاتصال*

![One-shot backup](docs/screenshots/nimbus-gui-one-shot-backup.png)
*تقدّم النسخ الاحتياطي في الوقت الفعلي مع الوقت المتبقي المقدّر والسرعة*

### الاستبعادات الذكية للنظام (وضع الملفات)
عند نسخ محرك أقراص كامل احتياطيًا (مثل `D:\`)، يستبعد Nimbus Backup تلقائيًا:

**مجلدات النظام:** `System Volume Information` (تخزين VSS، وقد يتجاوز 100 غيغابايت)، و`$RECYCLE.BIN`، و`Recovery`.
**ملفات النظام:** `pagefile.sys`، و`hiberfil.sys`، و`swapfile.sys`.

**لماذا يهم ذلك:** قد يُظهر محرك الأقراص 1.03 تيرابايت مستخدمة بينما لا يتجاوز حجم الملفات الفعلية ~141 غيغابايت. بدون الاستبعادات، ستتضمن النسخة الاحتياطية لقطات VSS (هدر للمساحة والوقت)؛ ومعها يطابق حجم النسخة الاحتياطية حجم البيانات الفعلية.

**التوصية:** استخدم **وضع الملفات** (الافتراضي) مع الاستبعادات التلقائية للنسخ الاحتياطي على مستوى الملفات؛ واستخدم **وضع القرص** في مهمة منفصلة للاستعادة الكاملة للنظام (bare-metal) (يتضمن كل شيء).

### الأمان والجودة
- التحقق من صحة المدخلات وتنقية بيانات الاعتماد (حجب الأسرار من السجلات)
- منع اجتياز المسارات (path traversal)
- منطق إعادة المحاولة مع تراجع أُسّي (exponential backoff)
- بوابات CI في كل بناء: الاختبارات، و`golangci-lint`، و`gosec`، و`go mod tidy`، ومجموعة اختبارات شاملة (end-to-end) على خادم PBS حقيقي (عمليات استعادة باستخدام `proxmox-backup-client` الرسمي، والتحقق PBS verify)

### 🔒 التشفير من جانب العميل
يمكن تشفير النسخ الاحتياطية **على جهاز العميل** قبل مغادرتها الجهاز، باستخدام
المخطط نفسه الذي يعتمده `proxmox-backup-client` الرسمي (AES-256-GCM، وملخّصات
مقاطع مرتبطة بالمفتاح). لا يخزّن خادم PBS سوى بيانات معتمة ولا يرى
المفتاح أبدًا — وهذا مفيد على خادم PBS مشترك أو مُدار.

طوّر Tiziano Bacocco التشفير في المشروع الأصلي ([tizbac/proxmoxbackupclient_go](https://github.com/tizbac/proxmoxbackupclient_go))؛ أما المفتاح الورقي وتصدير رمز QR واستيراد المفتاح فهي إضافات من Nimbus Backup.

- **الواجهة الرسومية**: *Servers → Edit → Encryption key* — أنشئ ملف مفتاح أو اختر
  ملفًا موجودًا (من `proxmox-backup-client key create --kdf none` أو من تخزين
  PVE)؛ وتُعرض بصمته. ثم احتفظ بنسخة منه خارج الجهاز:
  يحفظ **Print (paper key)** صفحة قابلة للطباعة تحتوي على المفتاح ورمز QR الخاص به (بتنسيق
  `proxmox-backup-client key paperkey`)، مع إمكانية حمايتها بعبارة مرور.
  يحوّل **Import a key from text or a QR code** رمز QR ممسوحًا ضوئيًا أو مفتاحًا ورقيًا
  إلى ملف مفتاح من جديد. لا تستخدم الواجهة الرسومية إلا ملفات المفاتيح غير المحمية: فاستيراد
  مفتاح محمي بعبارة مرور يفك قفله ويحفظ ملف المفتاح الجديد **بدون** عبارة
  مرور — احفظ ذلك الملف بالعناية نفسها التي تحفظ بها المفتاح ذاته.
- **سطر الأوامر**: `-keyfile path/to/key.json` (أو `"keyfile"` في ملف إعداد JSON)؛
  ويتطلب المفتاح المحمي بعبارة مرور الخيار `-keyfile-passphrase`، وإلا فسيُطلب منك إدخالها.
- **الاستعادة الكاملة للنظام (bare-metal)**: تستعيد صورة Clonezilla ISO المعدّلة نسخ
  الأقراص الاحتياطية المشفّرة أيضًا (المفتاح من ذاكرة USB، مع عبارة مروره إن وُجدت) — راجع
  [PATCH-CLONEZILLA.md](PATCH-CLONEZILLA.md).
- **متوافق مع `proxmox-backup-client`، ويُتحقَّق من ذلك في CI عند كل بناء**
  مقابل خادم PBS حقيقي: مجلد شفّره Nimbus Backup بمفتاح أُنشئ بواسطة
  `proxmox-backup-client` يستعيده العميل الرسمي بالمفتاح نفسه
  (ويرفض استعادته بدونه)، ونسخة احتياطية مشفّرة لقرص أنشأها العميل
  الرسمي يقرأها Nimbus Backup. ويستخدم Proxmox VE ملفات المفاتيح نفسها.

> ⚠️ **بدون المفتاح، لا يمكن استرداد النسخ الاحتياطية المشفّرة.** أول
> نسخة احتياطية مشفّرة ترفع كل شيء من جديد (لا إزالة للتكرار مع النسخ الاحتياطية
> غير المشفّرة). تبقى معرّفات النسخ الاحتياطية وأسماء الأرشيفات وأحجامها مرئية للخادم؛ أما محتويات
> الملفات وأسماؤها والفهرس فمشفّرة.

## 🤖 النشر غير التفاعلي (Ansible وIaC)

يُثبَّت Nimbus Backup ويُضبط بالكامل انطلاقًا من ملفات — دون أي إعداد تفاعلي.
هناك مساران، وكلاهما مغطى بأمثلة جاهزة للاستخدام في
[`examples/automation/`](examples/automation/):

- **سطر الأوامر** — ملف JSON واحد لكل مضيف يحمل إعدادات النسخ الاحتياطي كاملة
  (`proxmoxbackup-directory.exe --config file.json`)؛ والجدولة عبر Task Scheduler
  في Windows. رموز خروج موثوقة (`0` نجاح، `1` خطأ فادح، `2` مقفل، `3` جزئي)
  بحيث تكتشف أداة التنسيق لديك حالات الفشل. الخيار الأفضل للبنية التحتية ككود (IaC) بشكل خالص.
- **الواجهة الرسومية/الخدمة (MSI)** — ملف **`config.json` واحد** يحمل اتصال PBS،
  وإعدادات النسخ الاحتياطي **والجدولة** (`scheduled_jobs`). انشر الملف، وأعد تشغيل
  خدمة `NimbusBackup`: تتم مواءمة المهام بشكل متساوي القوة (idempotent) ويُحسب `nextRun`
  نيابةً عنك. لا حاجة لحسابات الطوابع الزمنية في قالب Jinja2 الخاص بك.

تقرأ الخدمة إعداداتها من `C:\ProgramData\ProxmoxBackupClient\`
(منذ الإصدار 0.4.0؛ راجع [الترقية](#️-الترقية-من-الإصدار--030)).

📖 الشرح الكامل خطوة بخطوة: [Automate Windows backup to PBS with Ansible](https://nimbus.rdem-systems.com/en/blog/unattended-windows-backup-ansible/?utm_source=github).

## 🚀 البدء السريع

1. نزّل `NimbusBackup.msi` (أو `NimbusBackup.exe` المستقل) من صفحة الإصدارات
2. ثبّته / شغّله بصلاحيات المسؤول (مطلوبة لـ VSS)
3. اضبط اتصال PBS واختبره
4. اختر المجلدات المراد نسخها احتياطيًا
5. ابدأ النسخ الاحتياطي — أو قم بجدولته

### 🔑 مستخدم PBS والصلاحيات

لا يحتاج العميل إلا إلى الدور **`DatastoreBackup`** على مخزن البيانات المستهدف — دون حساب مسؤول:

1. في واجهة PBS، أنشئ مستخدمًا (مثل `nimbus@pbs`) ورمز API له (مثل `nimbus@pbs!laptop01`).
2. في **Datastore → Permissions** (أو **Configuration → Access Control → Permissions**)، امنح `DatastoreBackup` على `/datastore/<name>` — أو على `/datastore/<name>/<namespace>` إذا كنت تنسخ احتياطيًا داخل namespace.
3. **رمز مع فصل الصلاحيات** (خيار "Privilege Separation" مُفعَّل، وهو الافتراضي): امنح الدور **للرمز** نفسه (`nimbus@pbs!laptop01`)، وليس للمستخدم فقط — فالصلاحيات الفعلية هي تقاطع الاثنين. وهذا هو السبب الأكثر شيوعًا لخطأ "permission denied".

يتيح `DatastoreBackup` للعميل إنشاء النسخ الاحتياطية وسرد مجموعات النسخ الاحتياطية الخاصة به واستعادتها. أما حذف اللقطات أو تقليمها (prune) فيتطلب `DatastorePowerUser`.

## ⬆️ الترقية من الإصدار ≤ 0.3.0

تثبيت الإصدار 0.4.0 أو أحدث فوق تثبيت موجود يرقّيه في مكانه (هوية MSI نفسها، وخدمة `NimbusBackup` نفسها). ينتقل مجلد البيانات من `C:\ProgramData\NimbusBackup` إلى المجلد المشترك `C:\ProgramData\ProxmoxBackupClient`: عند أول تشغيل، تُنسخ إعداداتك ومهامك المجدولة وسجلك ورمز API مرة واحدة (ولا يُستبدل أي ملف موجود أبدًا). يُحتفظ بالمجلد القديم، مع وضع العلامة `COPIED-TO-ProxmoxBackupClient.txt` عليه، بحيث يظل الرجوع إلى إصدار أقدم ممكنًا. ولا يزال بالإمكان استعادة اللقطات التي أُنشئت بالإصدارات الأقدم إلى موقعها الأصلي.

## 📋 المتطلبات

- Windows 10/11 أو Windows Server (64 بت)
- صلاحيات المسؤول (للقطات VSS)
- وصول شبكي إلى Proxmox Backup Server

## 🔨 البناء من المصدر

### المتطلبات المسبقة
- Go 1.25 أو أحدث
- Node.js 20 أو أحدث
- Wails CLI: `go install github.com/wailsapp/wails/v2/cmd/wails@v2.13.0` (الإصدار الذي يستخدمه CI)

### البناء

</div>

```bash
cd gui
npm install --prefix frontend
wails build      # or: wails dev  (hot reload)
```

<div dir="rtl">

أو ابنِ كل شيء (سطر الأوامر + الواجهة الرسومية + الخدمة) باستخدام Makefile: `make install-deps && make` (راجع `make help`). أدوات البناء على Windows والبناء المتقاطع عبر Docker: [BUILD.md](BUILD.md).

تُحدَّد العلامة التجارية من اسم الملف التنفيذي: يعمل `NimbusBackup.exe` باسم Nimbus Backup، وأي اسم آخر يعمل باسم "Proxmox Backup Client" المحايد ([`gui/brand.go`](gui/brand.go)).

## 🔗 العلاقة مع المشروع الأصلي (upstream)

بدأ Nimbus Backup كتفرّع (fork) من [tizbac/proxmoxbackupclient_go](https://github.com/tizbac/proxmoxbackupclient_go) (عميل Proxmox Backup بلغة Go، من تطوير Tiziano Bacocco، بترخيص GPLv3)، وأضفنا إليه الواجهة الرسومية لـ Windows، والخدمة، والجدولة، ودعم عدة خوادم PBS، والاستعادة. في سبتمبر 2026، دمج المشروع الأصلي تلك الواجهة الرسومية وجعلها محايدة من حيث العلامة التجارية ("Proxmox Backup Client GUI"). وفي أكتوبر 2026 (0.4.1)، أُعيد بناء Nimbus Backup على الشيفرة الحالية للمشروع الأصلي، مع اعتماد التشفير من جانب العميل الخاص بالمشروع الأصلي: **يتشارك المشروعان قاعدة الشيفرة نفسها.**

يعتمد Nimbus Backup 0.4.1 على الفرع `master` للمشروع الأصلي، عند الإيداع `3c1b989` (9 أكتوبر 2026). لا يوجد بعد أي إصدار من المشروع الأصلي يحتوي على هذه الشيفرة: آخر إصداراته، v1.1.3 (مايو 2026)، سابق لدمج الواجهة الرسومية.

ما يضيفه هذا المستودع فوق المشروع الأصلي هو سلسلة موثّقة من التصحيحات ([`patches/`](patches/README.md))، عُرض معظمها على المشروع الأصلي:

- **الميزات**: المفتاح الورقي (paper key) واستيراد المفتاح (رمز QR)، والنسخ الاحتياطي المتوازي للمجلدات، ولقطة VSS واحدة لكل نسخة احتياطية متعددة المجلدات، والاستثناءات في سطر الأوامر، وإعداد آلة افتراضية لـ Proxmox VE مُولَّد من الجهاز الفعلي.
- **إصلاحات لم تُدمج بعد في المشروع الأصلي** (استعادة النسخ الاحتياطية المضغوطة والمشفّرة الخاصة بـ `proxmox-backup-client`، وخادم PBS المختار لنسخة احتياطية عبر الخدمة، وأسباب رفض PBS، والاستعادة مع الخوادم التي تستخدم اسم المستخدم/كلمة المرور، وتكبير النافذة…) — تُرسل إلى المشروع الأصلي تباعًا مع دمجها.
- **هوية Nimbus Backup** — `NimbusBackup.exe`/`.msi`، وخدمة `NimbusBackup`، ورمز ترقية MSI للتثبيتات الموجودة، بحيث تستمر ترقيتها في مكانها.
- **مسار الترقية** من Nimbus Backup ≤ 0.3.0 (ترحيل مجلد البيانات، وبيانات تعريف اللقطات القديمة).
- **خط إنتاج الإصدارات** — شهادة مصدر البناء، والمجاميع الاختبارية، وتقارير VirusTotal.

نعيد دمج المشروع الأصلي بانتظام؛ وتُنشر إصدارات المشروع الأصلي نفسه على [tizbac/proxmoxbackupclient_go](https://github.com/tizbac/proxmoxbackupclient_go/releases).

## ⚠️ إخلاء المسؤولية

يُقدَّم هذا البرنامج كما هو. ورغم سعينا إلى الموثوقية، فإننا لا نتحمل أي مسؤولية عن أي فقدان للبيانات أو ضرر. اختبر نسخك الاحتياطية دائمًا وتحقق من الاستعادة قبل الاعتماد عليها في بيئة الإنتاج.

هذا المشروع **غير تابع** لشركة **Proxmox Server Solutions GmbH**. إن "Proxmox" والأسماء المرتبطة بها ملك لأصحابها المعنيين، وتُستخدم هنا فقط لبيان التوافق.

## 📄 الترخيص

GPLv3 — راجع ملف [LICENSE](LICENSE).

## حول RDEM Systems

يُطوَّر NimbusBackupClient ويُصان بواسطة [RDEM Systems](https://www.rdem-systems.com/en/?utm_source=github)، وهي شركة فرنسية لتوفير البنية التحتية متخصصة في الخدمات المُدارة لـ Proxmox VE/PBS والبنية التحتية لـ NTP/NTS. نشغّل [16 خادم NTS عامًا](https://ntp.rdem-systems.com/en/nts.php?utm_source=github) ([الحالة المباشرة](https://ntp.rdem-systems.com/en/status.php?utm_source=github)؛ 11 منها مدرجة في [المرجع المجتمعي](https://github.com/jauderho/nts-servers))، ونوفّر [استضافة PBS مُدارة بالكامل](https://nimbus.rdem-systems.com/en/?utm_source=github) للمستخدمين الذين لا يرغبون في الاستضافة الذاتية. كما نصون [unofficial-proxmox-backup-client](https://github.com/rdemsystems/unofficial-proxmox-backup-client)، وهي حزم `proxmox-backup-client` موقّعة لتوزيعات Linux التي لا تدعمها Proxmox رسميًا.

---

**© 2024-2026 RDEM Systems and Proxmox Backup Client GO contributors.**

</div>
