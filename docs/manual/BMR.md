# Bare-metal restore: user manual

This manual covers restoring a whole machine from a machine (full disk) backup. It applies to version 2.0. For file and folder restores use the GUI's Restore page ([GUI.md](GUI.md#9-restore)).

The restore copies the disk image block for block, so it works the same whatever the operating system on the disk: Windows, Linux or another. Onto a disk smaller than the one backed up, it shrinks a partition on the way when the data fits ([section 8](#8-restoring-onto-a-smaller-disk)).

## Contents

1. [Which restore do you need?](#1-which-restore-do-you-need)
2. [Before disaster strikes](#2-before-disaster-strikes)
3. [Get the ISO and make the boot media](#3-get-the-iso-and-make-the-boot-media)
4. [Automated restore, step by step](#4-automated-restore-step-by-step)
5. [The manual pbs-nbd entry](#5-the-manual-pbs-nbd-entry)
6. [Restoring a vm snapshot onto Proxmox VE](#6-restoring-a-vm-snapshot-onto-proxmox-ve)
7. [Restoring onto different hardware (P2V and replacement machines)](#7-restoring-onto-different-hardware-p2v-and-replacement-machines)
8. [Restoring onto a smaller disk](#8-restoring-onto-a-smaller-disk)
9. [Troubleshooting](#9-troubleshooting)

## 1. Which restore do you need?

| Situation | Use |
|---|---|
| Some files or folders deleted or damaged, machine still works | GUI Restore page ([GUI.md](GUI.md#9-restore)) |
| Disk dead or system unbootable, same or similar machine | Automated bare-metal restore ([section 4](#4-automated-restore-step-by-step)) |
| The new disk is smaller than the old one, but the data fits | Automated bare-metal restore, which shrinks a partition on the way ([section 8](#8-restoring-onto-a-smaller-disk)) |
| Machine gone, move it into a Proxmox VE VM | `vm` snapshot restored from Proxmox VE ([section 6](#6-restoring-a-vm-snapshot-onto-proxmox-ve)) |
| Copy a few files out of a disk image | Manual `pbs-nbd` entry, or `salvanza-cli attach-nbd` on Linux ([section 5](#5-the-manual-pbs-nbd-entry)) |
| Ransomware or corruption | Any of the above, from a snapshot taken **before** the problem |

A bare-metal restore needs a **machine** backup: a Backup Set or one-off backup with Backup type **Machine (full disk)**, or `salvanza-cli backup-disk`. Folder backups cannot be restored this way.

## 2. Before disaster strikes

* Make a machine backup of the system disk on a schedule, and keep folder backups of your data as well.
* Write down the PBS URL, a PBS **user name and password** that can read the datastore, the datastore name and any namespace. The restore ISO logs in with a user and password; it does not take API tokens. Make a dedicated PBS user (for example `restore@pbs`) with the **DatastoreReader** role on the datastore if you do not want to use an administrator account.
* If your backups are encrypted, keep a copy of the key file (and its passphrase) on a USB stick, away from the machine.
* Keep a spare USB stick with some free space, formatted as exFAT, NTFS or ext4. A restore onto a smaller disk may need it as scratch space ([section 8.4](#84-scratch-space)).
* Make the boot media now and test it once on a spare machine or VM.
* Note your machine's boot menu key (often F12, F11, F8, Esc or Del).

## 3. Get the ISO and make the boot media

The ISO is a Clonezilla Live image with this project's restore tools added.

1. Download `Salvanza-BMR-<version>.iso` from the [releases page](https://github.com/salvanza/salvanza_client/releases). In the GUI, **Tools > Download Bare Metal Restore ISO…** opens that page. Each release now has its own ISO, built with the rest of the release; an older release that did not change the ISO links to the one from an earlier release.
2. Check it against that release's `SHA256SUMS.txt`.
3. Write it to a USB stick (everything on the stick is erased):
   * Linux: `sudo dd if=Salvanza-BMR-<version>.iso of=/dev/sdX bs=4M conv=fsync status=progress`
   * Windows: a tool that writes the image as is, for example Rufus in "DD Image" mode, or balenaEtcher.

   Or burn it to a DVD, or attach it to a VM as a CD.

### A multi-purpose rescue stick with Ventoy (recommended)

Writing the ISO as above gives the whole stick to one image. [Ventoy](https://www.ventoy.net) is a better way to keep a recovery stick: it prepares the stick once, and from then on you simply **copy ISO files onto it** like ordinary files. At boot Ventoy shows a menu of every ISO on the stick, so one stick can carry this restore ISO next to a Windows installer, a Linux live system, memtest and so on. Updating to a newer restore ISO is a matter of copying the new file over.

1. Download Ventoy from [ventoy.net](https://www.ventoy.net) and install it to the stick: **Ventoy2Disk.exe** on Windows, `Ventoy2Disk.sh` or VentoyGUI on Linux. This erases the stick once.
2. Copy `Salvanza-BMR-<version>.iso` onto the stick's large data partition (it shows up as a normal drive). Do not unpack it.
3. Boot from the stick and pick the ISO in Ventoy's menu, then carry on as in [section 4](#4-automated-restore-step-by-step).

Ventoy starts on legacy BIOS and UEFI machines. With Secure Boot on, its first start asks you to enrol Ventoy's key once (follow its on-screen steps), or turn Secure Boot off for the restore.

The restore never offers the stick it booted from as the disk to restore to, a Ventoy stick or USB SSD included. This was tested from a Ventoy stick with UEFI and BIOS firmware, in Ventoy's normal and GRUB2 modes. Test the stick once on a spare machine or VM, as with any boot media. A Ventoy stick's data partition can also serve as scratch space for a restore onto a smaller disk, if it has room.

The ISO boots on both legacy BIOS and UEFI machines. It can also be network-booted like any Clonezilla Live image; setting that up is not covered here.

You can build your own ISO from a stock Clonezilla Live ISO with `patch-clonezilla.sh`. See [PATCH-CLONEZILLA.md](../../PATCH-CLONEZILLA.md).

## 4. Automated restore, step by step

This **overwrites the whole target disk**. Disconnect any disk you want to keep if you can.

Every box of the wizard says what happened, whether anything was written and what to do next. **Cancel** or **No** in any box before the restore starts stops it, and nothing is written ([4.9](#49-when-the-restore-stops)).

### 4.1 Boot

Plug in the boot media, power on and pick it from the boot menu. The first menu entry, **"Salvanza - PBS Bare Metal Restore"**, is the default: press Enter or let it start on its own. It skips Clonezilla's language and keyboard questions (it uses English with a UK keyboard layout). [Screenshot: the boot menu](../screenshots/bmr-01-boot-menu.png).

### 4.2 Network

DHCP is tried on every network adapter. If no address comes up, Clonezilla's own network screen appears so you can set a static address. The clock is set from the internet automatically. A wired connection is more reliable than Wi-Fi in the live system. After **Start again** ([4.9](#49-when-the-restore-stops)) the network is not set up again.

### 4.3 The backup server

Four boxes, each checked before moving on:

| Box | Example | Rule |
|---|---|---|
| Address of the backup server | `https://pbs.example.com:8007` | Must be complete and start with `https://` (already filled in) |
| User name | `restore@pbs` | Must include its realm after an @ (`@pam`, `@pbs`). `root@pam` is filled in. A user, not an API token |
| Password | | The user's password. Must not be empty. The box always starts empty |
| Datastore | `store1` or `store1/office` | Must not be empty or start with a `/`. Text after the first `/` is the namespace |

Screenshots: [the server address](../screenshots/bmr-02-url.png) and [the password](../screenshots/bmr-03-password.png) (the password is shown as stars while you type). While the wizard runs these details are kept in `/tmp/pbsnbd-credentials` (readable by root only); the file is removed whenever the wizard stops. **Start again** fills in the server, user name and datastore again, but never the password: type it again in full.

The backups are then listed. If that fails, a box says why in plain words, for example "The server refused the user name or password." or "The server could not be reached at pbs.example.com:8007.", and the restore stops without writing anything. The server's own message is kept in `/var/log/clonezilla/pbs-bare-metal-restore.log`. [Troubleshooting](#9-troubleshooting) lists the messages.

### 4.4 Encryption key

"Were these backups made with an encryption key?" ([screenshot](../screenshots/bmr-12-encryption-key.png))

| Answer | What happens |
|---|---|
| No, the backups are not encrypted | Carry on |
| Yes, the key file is on a USB stick or other disk | Pick the partition (each is listed with its file system, size and label). It is mounted read-only, the `*.json` files in its top three folder levels are listed (or the only one is used), the key is copied to `/tmp`, and the stick is unmounted. You can unplug it |
| Yes, I will type the path of the key file | Type the full path, for example `/run/live/medium/key.json` |

For a passphrase-protected key a hidden box asks for the passphrase. The passphrase is checked when the backups are listed: a wrong one is asked for again ("That passphrase did not unlock the key file (attempt 1 of 3). Please try again."), up to 3 attempts in total. The key is never written to the ISO or the target disk.

### 4.5 Pick the backup

Up to three menus; a menu with only one choice is skipped:

1. **Which machine** ("Which machine do you want to restore?"): each machine as its type and ID, `host/<name>` or `vm/<number>`, followed by its name. The name is the backup group's comment or, without one, the comment of its newest backup, which for a Salvanza machine backup is the Backup Set's name: `vm/9000198  Deepthought`. Machines with a name come first, sorted by name, then the others by type and ID. The names line up, and a long one is cut at the end of a word, with "..." after it ([screenshot](../screenshots/bmr-04-machine.png)).
2. **Which date** ("Backups of vm/9000198 (Deepthought - Full Machine). Choose the date to restore from:"): each backup as a day, date and time in the restore system's time zone, for example "Wed 11 Nov 2026 06:00 GMT". A backup that holds more than one disk image has "(2 disks)" after it ([screenshot](../screenshots/bmr-05-date.png)).
3. **Which disk image** ("The backup of ... from ... holds more than one disk image. Choose the one to restore:"): the disk images in their order, each with its size, for example `drive-scsi0.img.fidx  32 GiB`.

The wizard then opens the chosen image (it attaches it as `/dev/nbd0`). Until the server has it ready, a box says "Connecting to the backup on pbs.example.com:8007 ..." with the seconds so far. A busy server, for example one running a backup, verify or garbage collection job, can take a minute or more; the wizard waits up to 2 minutes.

If the image cannot be opened, the box "The chosen backup could not be opened for reading." gives the error and its likely cause: a key file that does not fit the backup, an encrypted backup with no key given, a refused sign-in, a server that cannot be reached or did not answer within 2 minutes. Nothing has been written to any disk. The full details are in the file the box names, `/tmp/pbsbmr-attach-<number>.log`.

### 4.6 Pick and confirm the target disk

The target is a local disk that is not marked removable, is not the stick or disc the restore started from (a Ventoy stick or a USB SSD included), and is not optical, a loop device or the backup itself. With exactly one such disk it is chosen for you. With several, a menu asks: "This computer has more than one disk. Choose the disk to restore onto. EVERYTHING on it will be PERMANENTLY OVERWRITTEN.", with each disk's size, for example `/dev/sda  (28 GiB)`. The menu and the confirmation list the boot stick or disc under "Not offered as the target:", with its size, so you can see it was recognised.

A target smaller than the backed up disk is no longer refused. The wizard checks whether the data fits and, if it does, shows its plan instead of the confirmation below: see [section 8](#8-restoring-onto-a-smaller-disk).

A target at least as large as the backed up disk is restored as before. The confirmation names the backup, its date and both sizes:

```text
Restore from : vm/9000198 (Deepthought), backup of Wed 11 Nov 2026 06:00 GMT  (29.9 GiB)
Restore onto : /dev/sda  (32 GiB), a disk in this computer

EVERYTHING on /dev/sda will be PERMANENTLY OVERWRITTEN. The backup on the server
is only read, never changed.

Clonezilla then asks you to confirm twice more before it writes anything.

Start the restore?
```

**No** stops the restore. After **Yes**, Clonezilla asks twice more before it writes. Answer `y` to each ([screenshot: Clonezilla's own warning and question](../screenshots/bmr-07-clonezilla-asks.png)).

### 4.7 The restore

Clonezilla copies the partition table, every partition and the data between the partition table and the first partition. That last part matters on legacy BIOS disks, where GRUB keeps part of itself there; without it the disk would stop at a bare `GRUB` prompt. UEFI (GPT) disks restore their EFI system partition like any other partition.

A Linux machine backup carries the block snapshot's copy-on-write file (`.pbs_snapshot_N.cow`, about a tenth of each snapshotted file system). Backups made with this version leave its contents out of the image, but the file itself is still there. After a successful restore the ISO deletes it from each ext2/3/4 and XFS partition. A test-build ISO from before 6 October 2026 does not: delete `/.pbs_snapshot_*.cow` (and the same name at the top of any other snapshotted file system, such as `/home`) yourself after the first start.

Restores of 32 to 40 GiB Windows and Linux test machines took 8 to 11 minutes from power-on to the login prompt on a local network. Time depends on disk size, network and PBS speed. [Screenshot: a partition being restored](../screenshots/bmr-08-progress.png).

A restore onto a smaller disk shows its own screens instead ([8.6](#86-the-restore-onto-a-smaller-disk)).

### 4.8 Finish

* **Success:** "The restore is complete." ([Screenshot](../screenshots/bmr-09-complete.png).) The box goes on: "Press OK to switch the computer off. Then remove the restore stick or disc and switch the computer on again: it starts from the restored disk /dev/sda." Choose **OK** and the machine switches off. Just before it does, the console may show a few lines about a device that is "busy" and the prompt "Please remove the live-medium, close the tray (if any) and press ENTER to continue" ([screenshot](../screenshots/bmr-10-remove-medium.png)). That is Clonezilla's normal last step: remove the boot stick or disc if you like and press Enter, and the machine switches off. If nobody is there to press Enter it waits. Then, with the boot stick or disc removed, switch the machine on: it starts from the restored disk. If it starts the restore system again, choose the restored disk in the firmware's boot menu. (Test-build ISOs from before 25 October 2026 restarted instead, and from a USB stick that started the restore system again.)
* **Failure:** "The restore did not finish (Clonezilla stopped with exit code N)." If you answered no to one of Clonezilla's questions, nothing has been written. Otherwise the disk may be partly written: do not use it until a restore has finished. The backup on the server was only read and has not changed. Choose **OK**; the machine does not switch off. Look in the log folder, `/var/log/clonezilla`, then run the restore again.

### 4.9 When the restore stops

Every stop before anything is written ends in the same menu, whether you chose **Cancel** or **No**, or the wizard could not go on (a refused password, no backups, a key that would not open, a backup that could not be opened): "The restore was stopped. Nothing was written. What would you like to do now?" ([screenshot](../screenshots/bmr-11-stopped.png)).

| Choice | What happens |
|---|---|
| a Start again | The wizard starts again at the server address, without setting up the network again. The server, user name and datastore are filled in; type the password again |
| b Power off | Switches the computer off |
| c Reboot | Restarts the computer |
| d Command line | Leaves you at the command line, which shows the command that starts the restore again |

The wizard lets go of the backup on every stop, so **Start again** can open it again straight away.

### After the restore

* The restored system is exactly as it was at the snapshot: files, settings and permissions, whatever the operating system. A Linux system keeps its machine ID (`/etc/machine-id`), from which systemd-networkd makes its DHCP client ID, so a DHCP server that keys leases by client ID still knows the machine. (Test-build ISOs from before 25 October 2026 gave it a new machine ID and rebuilt its initramfs; the rebuild is kept, as it helps on different hardware.)
* On different hardware, a Linux system's network settings may name the old network card (by its MAC address, as cloud images and netplan configurations often do, or by an interface name that has changed). The system starts but has no network until the settings name the new card.
* If it does not start, repair its boot loader with the operating system's own tools. This is more likely if the disk controller or boot mode (UEFI or BIOS) differs from the original machine.
  * Windows: boot a Windows installation USB and use **Repair your computer > Troubleshoot > Startup Repair**.
  * Linux: boot a live Linux system and reinstall the boot loader (usually GRUB) the way your distribution describes.
* Open the GUI and check that the Backup Sets run again.

## 5. The manual pbs-nbd entry

Clonezilla's own main menu (reached by choosing the normal Clonezilla entries at boot) has an extra mode, **pbs-nbd**. It asks for the same PBS details, key and snapshot as section 4 (the network comes from Clonezilla's own setup, and a wrong key passphrase stops it at once instead of asking again), attaches the image to `/dev/nbd0` in the background, and returns you to Clonezilla. Then either:

* use Clonezilla's own device-to-device wizard with `nbd0` as the source, for a restore where you choose every option yourself, or
* open a shell, `mount -o ro /dev/nbd0p1 /mnt` (after `partprobe /dev/nbd0`), and copy files out.

Running pbs-nbd again stops the previous background attach first.

On any Linux machine the same is done with `salvanza-cli attach-nbd` ([CLI.md](CLI.md#6-attach-nbd-attach-a-disk-image-linux)).

## 6. Restoring a vm snapshot onto Proxmox VE

A machine backup saved with **Backup as: Proxmox VE virtual machine** (or `salvanza-cli backup-disk -type vm`) is stored as `vm/<number>`. Proxmox VE sees it as an ordinary VM backup.

1. In Proxmox VE, add the PBS datastore as storage (**Datacenter > Storage > Add > Proxmox Backup Server**), with the same namespace.
2. Open that storage, **Backups**, select the `vm/<number>` snapshot, **Restore**.
3. Choose a free VM ID and a target **Storage**, and restore.

The VM is minimal: one SATA disk, OVMF (UEFI) firmware for GPT disks, no network card, no TPM, no Secure Boot keys. Add a network card and anything else the guest needs before starting it. A Windows Server UEFI disk restored this way booted without further changes.

The same `vm` snapshot can also be restored onto bare metal with the ISO; it appears in the machine menu as `vm/<number>`.

## 7. Restoring onto different hardware (P2V and replacement machines)

### Match the firmware to the disk

A disk only starts with the kind of firmware it was set up for. Whether you restore a physical machine into a VM (P2V) or onto a replacement computer, set the target's firmware to match the original machine:

| Original machine | Disk | Proxmox VE VM (Hardware > BIOS) | Replacement computer (firmware setup) |
|---|---|---|---|
| UEFI | GPT | **OVMF (UEFI)**, plus an **EFI Disk** with **Pre-Enroll keys** unticked (Secure Boot off) | UEFI boot; Secure Boot off if the restore will not start |
| Legacy BIOS | MBR | **SeaBIOS** (the default) | Legacy or CSM boot |

With the wrong firmware the restore itself works, but the restored disk does not start: the VM stops at "No bootable device" or the UEFI shell. Change the firmware setting and start it again; there is no need to restore again.

To find out which the original used: on Windows, **System Information** (`msinfo32`) shows **BIOS Mode: UEFI** or **Legacy**, and Disk Management shows a disk's partition style (GPT or MBR) under **Properties > Volumes**. On Linux, UEFI if the folder `/sys/firmware/efi` exists. Nearly every computer from 2015 on uses UEFI and GPT.

A `vm` snapshot restored through Proxmox VE ([section 6](#6-restoring-a-vm-snapshot-onto-proxmox-ve)) gets the right firmware by itself. Add an EFI Disk (Pre-Enroll keys unticked) if a UEFI one does not start.

### A physical machine into a VM with the ISO

1. Create an empty VM: **BIOS** as in the table above, a **SATA** disk at least as large as the original (or smaller, if the data fits: [section 8](#8-restoring-onto-a-smaller-disk)), an **Intel E1000** network card (Windows has the driver built in; VirtIO is fine for Linux), and the ISO in the CD drive, first in the boot order.
2. Run the restore ([section 4](#4-automated-restore-step-by-step)).
3. Take the CD out of the boot order and start the VM. Windows shows "Getting devices ready" while it sets up the new hardware, and may restart once.

### Tested routes

| From | To | Route | Result |
|---|---|---|---|
| Windows 10 laptop, UEFI/GPT (`vm` snapshot) | Proxmox VE VM, OVMF, SATA | ISO, booted from a disk | Started at the user's desktop |
| Windows Server, UEFI/GPT (`vm` snapshot) | Proxmox VE VM | Proxmox VE restore | Started without changes |
| Xubuntu desktop, UEFI/GPT (`vm` snapshot) | Proxmox VE VM, OVMF | Proxmox VE restore, and the ISO | Started at the desktop (Proxmox VE route: EFI Disk added first) |
| Linux, legacy BIOS/MBR (`host` snapshot) | Proxmox VE VM, SeaBIOS | ISO, and Proxmox VE restore of a `vm` copy | Started at the login prompt |
| Linux Mint, encrypted backup, passphrase-protected key | Proxmox VE VM, SeaBIOS | ISO, key from a USB stick | Started |
| Linux PC | the same physical PC (new disk) | ISO | Started |
| Debian 13, UEFI and BIOS boot, encrypted backup, passphrase-protected key | Proxmox VE VM with two internal disks, OVMF and SeaBIOS; disk of the same size, and larger | ISO on a non-removable USB disk, key from a USB stick | Started at the login prompt, files, UUIDs, host keys and machine ID unchanged |

Restores onto smaller disks are listed in [section 8.9](#89-tested).

### Drivers

A disk image carries the drivers of the machine it came from.

* **Linux** usually starts on different hardware.
* **Windows** may stop with `INACCESSIBLE_BOOT_DEVICE` if the disk controller is very different. The SATA disk that a `vm` restore creates avoids the most common case, because Windows has built-in SATA drivers. Install VirtIO drivers inside the VM afterwards if you want to switch to faster virtual hardware.
* When a whole-disk restore will not start, the reliable route is a fresh Windows install followed by a folder restore of your data with the GUI.

## 8. Restoring onto a smaller disk

A new disk is often a little smaller than the old one, even when both are sold as the same size: two "1 TB" drives can differ by 22 GiB. A disk image holds the whole disk, partition table and all, so a block for block copy needs a disk at least as large. When the new disk is smaller but the data on it fits, the restore ISO shrinks the backup on the way.

An ISO of version 2.0 or older refuses any smaller disk ("The target disk is smaller than the backed up disk"). Build or download a current one.

### 8.1 When it applies

* **The new disk is smaller than the backed up disk and the data fits:** the wizard checks, shows a plan and asks before it writes anything.
* **The data does not fit:** the wizard says so, with the size of the smallest disk the data could fit onto, and stops. Nothing is written.
* **The new disk is at least as large as the backed up disk:** the restore runs as in [section 4](#4-automated-restore-step-by-step), through Clonezilla, as before. The disk is not grown: the extra space is left free at the end of the disk. You can extend a partition into it afterwards with the operating system's own tools (Disk Management on Windows, for example).

### 8.2 How it works

The backup on the server is only read, never changed. The restore works out how small each partition can get, then takes the partition with the most room and shrinks it just enough for the new disk, keeping some free space in it. The partitions in front of it keep their places, and the ones after it move down. The file system is shrunk in a temporary writable layer over the read-only backup, whose changes are held in memory or in scratch space ([8.4](#84-scratch-space)). The new disk gets the same partition table type (GPT or MBR), the same disk ID and the same partition IDs, and each file system keeps its UUID or serial number and its label, so the restored system finds its disks and partitions as before. The boot code and the area before the first partition, where GRUB keeps part of itself on a BIOS disk, are copied from the backup; a moved NTFS or FAT partition gets its boot sector updated with its new place, and on a UEFI computer the boot entry for the new disk is updated afterwards. Each partition is then copied with partclone, used blocks only, as Clonezilla does.

### 8.3 The check and the plan

When the target is smaller, a box says "The new disk /dev/sda (24 GiB) is smaller than the backed up disk (29.9 GiB). Checking whether the data fits on it. This reads the backup and takes about a minute. Nothing is written yet." ([screenshot](../screenshots/bmr-20-smaller-checking.png)).

The plan follows: a table of the partitions, then lettered points. It is shown in the confirmation ([screenshot](../screenshots/bmr-23-smaller-confirm.png)), in the question about scratch space ([8.4](#84-scratch-space)) and in the box that says the disk cannot be used. Above the table are the two disks, for example `Backup disk: 29.9 GiB (GPT), /dev/nbd0` and `New disk   : 24 GiB, /dev/sda`.

None of these boxes scrolls, so Enter always works on the highlighted button. When the plan and the question under it do not fit on the screen together (a small screen, such as 800x600, with a long plan), the plan comes first in a box of its own with **Next**, and the question follows in a small box with **Yes** and **No**, **Yes** highlighted. A plan taller than the screen is shown in pages, "(1 of 2)" after the title, each with **Next**. On a larger screen it is all one box, as in the screenshots.

| Column | Meaning |
|---|---|
| No | The partition number. Rows marked `lv` under an LVM partition are its logical volumes |
| Type | The file system: NTFS, ext4, btrfs, XFS, FAT, exFAT, f2fs, swap or LVM; `-` for none, `(ext.)` for the extended partition that holds the logical ones |
| Size | Its size in the backup |
| Needs | The smallest size its data needs; `-` for a partition that is kept |
| New size | Its size on the new disk |
| What happens | **shrinks** for the one that gets smaller (for FAT, exFAT, f2fs and XFS "a new, smaller ..., its files copied"; for swap "made again, smaller"); **kept as it is**; **moves, otherwise kept as it is** for one after it; **made again at its size** for swap; or why it is kept, for example "kept: encrypted, its contents cannot be read" |

Under the table:

* **a)** which partition shrinks, and to what: "Partition 3 (NTFS) holds 20 GiB of data, so the new partition will be 23.3 GiB instead of 29.1 GiB." (For LVM it names the logical volume.)
* **b)** the smallest disk this backup fits onto: "21.8 GiB (23,321,395,712 bytes) is the smallest disk this data could fit onto."
* **c)** the data that must be stored while the partition shrinks, where it is held and how much room there is: "3,957 MB has to be stored while it shrinks. It will be held in /tmp/pbs-shrink (memory), which has ... MB available." It is left out when nothing has to move.
* **d)** only when that room is too small: "Scratch space of at least 4,608 MB is needed, in a folder on another volume (a USB stick or another disk)."

If the data fits and the memory has room, the confirmation comes next, titled **confirm (smaller disk)**: "The new disk is smaller than the backed up disk, but the data fits. This is the plan:", the plan, then "The backup on the server is only read, never changed. EVERYTHING on /dev/sda (24 GiB) will be PERMANENTLY OVERWRITTEN. Start the restore?" On a small screen the plan comes first with **Next**, and this question follows in a small box. **Yes** starts the restore; Clonezilla's two questions are not asked on this route. **No** stops it ([4.9](#49-when-the-restore-stops)).

If the data does not fit on the new disk even with a partition shrunk, the box **cannot use this disk** shows the plan with the smallest disk it would fit onto, and the other ways: a disk at least the size of the backed up one, or installing the operating system and the Salvanza backup client and restoring files from the backup with the app's Restore page. Nothing is written.

### 8.4 Scratch space

While a file system shrinks, the data past its new end is moved inside it. The backup cannot be written to, so every block that changes is kept elsewhere until the partition has been copied to the new disk. When it fits, that is the restore system's memory, and nothing is asked. When it does not, the restore needs scratch space: a partition on another disk or a USB stick with a little more free space than the data that moves. Point d) of the plan gives the figure. In the tests a Windows disk cut by 6 GiB needed 4,608 MB, and a Linux Mint disk cut from 119 to 40 GiB needed 17,663 MB.

The box **scratch space needed** shows the plan and asks: "Do you have a suitable volume with at least 4,608 MB available to use as temporary scratch space? If so, please insert it now if required and select Yes." ([screenshot](../screenshots/bmr-21-smaller-scratch-needed.png)). On a small screen the plan comes first, in its own box titled **scratch space needed**: press Enter for **Next**, and the question follows in a small box of its own ([8.3](#83-the-check-and-the-plan)). Plug the stick in first, then choose **Yes**.

A menu then lists the partitions that can be used, each with its file system, size and free space in MB, as the plan gives it, for example `/dev/sdb1  ext4, 16 GiB, 15,182 MB free` ([screenshot](../screenshots/bmr-22-smaller-scratch-pick.png)). "A folder named pbs-shrink-scratch is made on it for the restore and removed afterwards. Nothing else on it is touched."

* Use a stick or disk formatted as **exFAT, NTFS or ext4**. ext2, ext3, XFS and btrfs work too. **FAT32** only works when less than 4 GB is needed, because a FAT32 file cannot be larger.
* The disk being restored onto cannot be used, and is not listed. A Ventoy stick's data partition can be, if it has room; it is marked "the restore stick".
* An NTFS disk from a Windows computer that was not shut down fully (Fast Startup or hibernation) cannot be opened. The box says so, and **OK** brings back the list to choose again.
* If no usable partition is found yet, for example because the stick was plugged in a moment ago: "No USB stick or other disk with a usable partition was found yet. Plug it in, wait a few seconds, then choose Look again." Choose **Look again**, or **Stop**.

After the choice, "Checking the scratch space on /dev/sdb1 (15,182 MB free) ...", the same figure as in the menu, takes a few seconds (the backup is not read again), and the confirmation follows, with point c) naming the scratch folder. If the partition has too little room, the question comes back.

### 8.5 Without scratch space

**No** to the question, **Stop**, or **Cancel** in the list shows the box **no scratch space** ([screenshot](../screenshots/bmr-26-smaller-no-scratch.png)): "Without suitable scratch space this restore cannot be done automatically onto the 22 GiB disk. Nothing has been written to it." It lists what you can do instead:

* a) Start the restore again with a USB stick or another disk that has the space needed free (exFAT, NTFS or ext4).
* b) Restore onto a disk of at least the size of the backed up disk. That needs no scratch space.
* c) Install Windows (or the operating system the backup holds) and the Salvanza backup client on this disk, then use the app's Restore page to open the disk image from this backup and restore your folders and files from it. Programs must be reinstalled, and the disk and the system get new identities.

**OK** stops the restore and shows the stop menu ([4.9](#49-when-the-restore-stops)). The backup itself is not changed.

### 8.6 The restore onto a smaller disk

From **Yes** on, every step is shown in a blue box, with a gauge for the long ones:

1. "Getting ready: reading the backup's partitions and checking the new disk ..." The results of the check are kept, so this takes seconds.
2. **Step 1 of 4**: the partition shrinks, for example "partition 1 (ext4) shrinks from 230.7 GiB to 16.2 GiB". For NTFS the gauge follows ntfsresize; for ext4 and btrfs it shows the data moved, "Moving data: 7,914 of about 11,308 MB (80%), 2 min 04 s so far", and for ext4 then the later passes of the resize in plain words, "Now: updating references (10%)" ([screenshot](../screenshots/bmr-24-smaller-shrinking.png)). This is the longest step when much data moves.
3. **Step 2 of 4**: "writing the new partition table and boot code to /dev/sda ...".
4. **Step 3 of 4**: each partition in turn. Before each copy a box says what is copied: "copying partition 3 (NTFS, 23.3 GiB) from the backup onto the new disk. The next screen is the copy program's own progress (it calls this a backup of one device to another)." Then comes partclone's own screen, as in a normal restore ([screenshot](../screenshots/bmr-25-smaller-partclone.png)); it names loop devices (`/dev/loop1` and so on), which is expected. A FAT, exFAT, f2fs or XFS partition is made again and its files are copied, with a gauge of the data copied; swap is made again; an LVM volume group is written again.
5. **Step 4 of 4**: "finishing (writing everything out to /dev/sda) ...", then "Finishing: tidying up the scratch space and the restored partitions ...".
6. "The restore is complete." as in [4.8](#48-finish).

In the tests the whole restore took 2 to 16 minutes ([8.9](#89-tested)).

If it does not finish, the box **restore failed** says "The restore onto the smaller disk did not finish." with the last lines of its log. The new disk may be partly written and will probably not start; the backup on the server was only read. Start the restore again, or restore onto a disk at least the size of the backed up one. The logs are in `/var/log/clonezilla/pbs-shrink-logs` (`run.log` first); the plan of each check is in `/var/log/clonezilla/pbs-shrink-plan.log`.

### 8.7 What can shrink

| On the backed up disk | What happens |
|---|---|
| NTFS | Shrunk with ntfsresize |
| ext2, ext3, ext4 | Shrunk with resize2fs (after a file system check). A Linux machine backup's snapshot store file (`.pbs_snapshot_N.cow`) is removed first |
| btrfs | Shrunk with btrfs resize |
| XFS | XFS cannot shrink: a new, smaller XFS with the same UUID, label and features is made, and the files are copied into it |
| FAT, exFAT, f2fs | Made again smaller, with the same serial number or UUID and label (f2fs also its features), and the files copied into it |
| swap | Made again, smaller if need be, with the same UUID and label |
| LVM | A volume group on one physical volume (a partition) with linear logical volumes: it is written to the new disk again with the same UUIDs and names, the logical volumes packed together, and one of them shrunk as above if need be |
| Partition tables | MBR, with extended and logical partitions, and GPT |
| Firmware | UEFI and legacy BIOS |

Not yet handled. Such partitions are kept at their size, and the restore still works when another partition can give up the room:

* encrypted partitions, such as BitLocker or LUKS (their contents cannot be read);
* software RAID members and Windows dynamic disks;
* LVM spread over several disks, and thin or RAID logical volumes;
* an NTFS partition from a Windows that was hibernated or not shut down fully may not shrink; the plan says why.

Also not handled: a cut shared between two partitions (one partition must be able to give up all the room), disks with sectors other than 512 bytes, and growing onto a bigger disk ([8.1](#81-when-it-applies)).

### 8.8 After the restore

Remove the boot stick and switch the computer on.

* **Windows** may check the shrunk disk on its first start: let it finish. It may restart once. If a recovery partition was moved, `reagentc /info` in an administrator command prompt shows whether the recovery environment is still enabled; `reagentc /enable` turns it on again.
* **Linux** needs nothing: the file systems keep their UUIDs, so `/etc/fstab` and the boot loader find them as before.

The rest of section 4's [After the restore](#after-the-restore) applies as well.

### 8.9 Tested

Each was restored with the ISO in a Proxmox VE VM and started with its network off.

| Backed up machine | Firmware, partition table | Backed up disk to new disk | What shrank | Scratch space | Result |
|---|---|---|---|---|---|
| Windows | UEFI, GPT | 29.9 GiB to 24 GiB | NTFS C: 29.1 to 23.3 GiB (20 GiB data) | 4,608 MB on a USB disk | Lock screen, about 16 minutes |
| Windows 10 | BIOS, MBR | 32 GiB to 28 GiB | NTFS C: 31.5 to 27.5 GiB (26 GiB data) | None (memory) | Lock screen, about 16 minutes |
| Linux Mint | UEFI, GPT | 119.2 GiB to 40 GiB | ext4 118.8 to 39.5 GiB (20.8 GiB data) | 17,663 MB on a second disk | Login screen, about 16 minutes |
| Debian 13 | BIOS, MBR with a logical swap partition | 238.5 GiB to 24 GiB | ext4 230.7 to 16.2 GiB; swap made again | Yes | Login prompt, about 6 minutes |
| Fedora | BIOS, GPT | Onto 20 GiB | btrfs 31.9 to 19.9 GiB (15.6 GiB data) | 12,288 MB moved through scratch space | Desktop, about 9 minutes |
| Ubuntu 22.04 | BIOS, GPT, LVM | Onto 20 GiB | ext4 logical volume 30.3 to 18.3 GiB (10.8 GiB data) | None (memory) | Login prompt, about 3 minutes |
| CentOS 5 | BIOS, MBR, LVM | Onto 5 GiB | ext3 logical volume 7.4 to 4.4 GiB (3.4 GiB data); swap logical volume made again | None (memory) | Login prompt, about 2 minutes |
| Data disks | GPT | 2 GiB to 1 GiB each | FAT32, exFAT and f2fs, each made again | None | Files identical, same IDs and labels, file system checks clean |
| Test file system | | 3 GiB to 1.9 GiB | XFS made again | None | 27,726 files identical, same UUID and features |

## 9. Troubleshooting

| Message or symptom | What to check |
|---|---|
| "The restore system could not load the part of Linux it needs to read the backup (the nbd kernel module)." | The ISO's kernel lacks the module. Rebuild from a current Clonezilla ISO |
| "The server refused the user name or password." | The user name with its realm (`@pam`, `@pbs`) and the password. The password box starts empty on **Start again**: type the whole password |
| "The server could not be reached at ..." | The address and port, the network cable, and that the server is running |
| "A secure connection to the server at ... could not be made: the server's certificate was not accepted." | The address; the certificate of the server at that address |
| "The server at ... did not answer as a secure (https) server." | The port in the address: a Proxmox Backup Server uses 8007 |
| "There is no datastore called ... on that server." or "There is no namespace called ..." | The datastore name, and the namespace after the first `/` |
| "The user ... may not see the backups in datastore ..." | The PBS user needs the DatastoreReader role (or higher) on the datastore |
| "The backups could not be listed. The server or the network said: ..." | The server address, user name, password and datastore. The full message is in `/var/log/clonezilla/pbs-bare-metal-restore.log` |
| "No backups were found in datastore ..." | Datastore and namespace (text after the first `/`), and the user's rights on it |
| "That passphrase did not unlock the key file" | Retype the passphrase; 3 attempts in total |
| "No USB stick or disk with a readable partition was found, so the key file cannot be loaded." | Plug in the key USB stick, then **Start again** |
| "The chosen backup could not be opened for reading." | The box gives the error and the likely cause: a key file that does not fit the backup (the attach log then says "this snapshot was encrypted with another key" and shows the fingerprint of the key the snapshot needs and of the key given), an encrypted backup with no key given, a refused sign-in, or a server that cannot be reached. Full details in `/tmp/pbsbmr-attach-<number>.log` |
| "... did not open the backup within 2 minutes. It may be busy ..." | The server is busy with a backup, verify or garbage collection job. Start again later |
| "compressed encrypted blobs (zstd block format) are not supported" | The ISO was built from version 2.0 or older and the encrypted backup was made by version 2.1 or later with **Compress encrypted backups** on ([GUI.md](GUI.md#encryption-tab)), or by proxmox-backup-client. Build or download a current ISO |
| "this snapshot was written by version ... of this client in a format this version cannot read" | The backup was made by a newer version than the ISO. The message names the version needed: build or download an ISO of that version or later |
| "No disk was found to restore onto." | The new disk is missing, unsupported by the live system, or seen as removable. Check cabling and BIOS settings. The stick the restore booted from is never offered |
| Several disks and unsure which is the target | Compare sizes in the menu; if still unsure, check disk models and serials in the firmware setup first, or disconnect the others |
| "The target disk is smaller than the backed up disk" | An ISO of version 2.0 or older. A current ISO restores onto a smaller disk when the data fits ([section 8](#8-restoring-onto-a-smaller-disk)) |
| "This backup cannot be restored onto the ... disk" (cannot use this disk) | The data does not fit, even with a partition shrunk. Point b) or the last line of the plan gives the smallest disk it fits onto. Nothing has been written |
| "Without suitable scratch space this restore cannot be done automatically" | Start again with a USB stick or disk that has the space the plan asks for, formatted as exFAT, NTFS or ext4 ([8.4](#84-scratch-space)) |
| A partition offered for scratch space "could not be opened (mounted)" | An NTFS disk from a Windows that was not shut down fully, or an unknown format. Choose another, or use a stick formatted as exFAT, NTFS or ext4 |
| "The restore onto the smaller disk did not finish." | The box shows the last lines of the log; the logs are in `/var/log/clonezilla/pbs-shrink-logs` (`run.log` first). Do not use the disk until a restore has finished |
| Stops at a `GRUB` prompt after restore | Should not happen with this ISO (it restores the area GRUB uses). Make sure you used the "PBS Bare Metal Restore" entry, not a hand-run restore without Clonezilla's `-j2` option |
| A restored VM stops at "No bootable device" or the UEFI shell | The firmware does not match the disk: OVMF (UEFI) for a GPT disk, SeaBIOS for MBR ([section 7](#match-the-firmware-to-the-disk)). Change it and start again |
| The restored system will not start | Check the firmware first (row above). Then repair the boot loader: Startup Repair from Windows installation media, or the distribution's GRUB repair on Linux (see "After the restore") |
| A large `.pbs_snapshot_N.cow` file on a restored Linux system | Restored with a test-build ISO from before 6 October 2026; delete the file (see [4.7](#47-the-restore)) |
