Salvanza for Proxmox Backup Server: the background service and the command line
for a Linux computer that has no Salvanza package (Slackware, Alpine, Gentoo,
Void and others). Both programs are static: they run with glibc and musl.

Install or update, as root:

    sh install.sh

This puts salvanza-svc in /usr/libexec/salvanza and salvanza-cli in /usr/bin,
then sets the service up for the computer's service manager and starts it:

    systemd      /etc/systemd/system/salvanza.service
    SysV init    /etc/init.d/salvanza, started at boot (update-rc.d, chkconfig
                 or the /etc/rcN.d links)
    Slackware    /etc/rc.d/rc.salvanza, started from /etc/rc.d/rc.local and
                 stopped from /etc/rc.d/rc.local_shutdown
    OpenRC       /etc/init.d/salvanza, in the default run level

Then, as root, salvanza-cli tui sets up the servers and Backup Sets.

Remove the service (the settings, Backup Sets and keys in /var/lib/salvanza
are kept):

    sh install.sh remove

Sleeping or hibernating the computer after a backup uses systemd, else
elogind's loginctl, else the kernel's /sys/power/state.

The manuals: https://github.com/salvanza/salvanza_client/tree/master/docs/manual
