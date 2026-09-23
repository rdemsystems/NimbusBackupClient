%global debug_package %{nil}

# Version is overridden by packaging/fedora/build-rpm.sh from gui/wails.json
Name:          pbsgo
Version:       0.2.119
Release:       1%{?dist}
Summary:       Backup client for Proxmox Backup Server (CLI + GUI)

License:       GPLv3+
URL:           https://github.com/tizbac/proxmoxbackupclient_go
Source0:       pbsgo-%{version}.tar.gz

BuildRequires: golang
BuildRequires: gcc
BuildRequires: gtk3-devel
BuildRequires: webkit2gtk4.1-devel

%description
Go-based backup client for Proxmox Backup Server:
 - pbsgo: directory and stream backup with deduplication
 - pbsgo-machine: whole-machine (raw disk) backup, using VSS on
   Windows and block-level snapshotting on Linux
 - pbsgo-nbd: mount fidx images as read-only NBD block devices
 - pbsgo-gui: graphical interface; machine (whole-disk) backup
   requires root, use the pbsgo-gui-root launcher for that
The GUI frontend is embedded in the binary, no Node.js is needed at
runtime.

%prep
%setup -q

%build
# directorybackup is intentionally not part of go.work
( cd directorybackup && GOWORK=off go build -trimpath \
    -ldflags "-s -w -X main.version=%{version}" -o pbsgo . )
( cd machinebackup && go build -trimpath \
    -ldflags "-s -w -X main.version=%{version}" -o pbsgo-machine . )
( cd nbd && go build -trimpath \
    -ldflags "-s -w -X main.version=%{version}" -o pbsgo-nbd . )
# the wails desktop frontend needs the production build tag; webkit2_41
# selects webkit2gtk-4.1 (wails defaults to the EOL webkit2gtk-4.0)
( cd gui && go build -tags production,webkit2_41 \
    -ldflags "-s -w -X main.appVersion=%{version}" -o pbsgo-gui . )

%install
rm -rf %{buildroot}
install -d -m 0755 %{buildroot}%{_libdir}/pbsgo \
                  %{buildroot}%{_bindir} \
                  %{buildroot}%{_datadir}/applications \
                  %{buildroot}%{_datadir}/icons/hicolor/256x256/apps

install -m 0755 directorybackup/pbsgo      %{buildroot}%{_bindir}/pbsgo
install -m 0755 machinebackup/pbsgo-machine %{buildroot}%{_bindir}/pbsgo-machine
install -m 0755 nbd/pbsgo-nbd              %{buildroot}%{_bindir}/pbsgo-nbd
install -m 0755 gui/pbsgo-gui              %{buildroot}%{_libdir}/pbsgo/pbsgo-gui

install -m 0644 gui/Icon.png \
    %{buildroot}%{_datadir}/icons/hicolor/256x256/apps/pbsgo.png

cat > %{buildroot}%{_datadir}/applications/pbsgo-gui.desktop <<'EOF'
[Desktop Entry]
Name=Proxmox Backup Client
Comment=Backup client for Proxmox Backup Server
Exec=pbsgo-gui
Icon=pbsgo
Terminal=false
Type=Application
Categories=System;Utility;
EOF
chmod 0644 %{buildroot}%{_datadir}/applications/pbsgo-gui.desktop

cat > %{buildroot}%{_bindir}/pbsgo-gui <<'EOF'
#!/bin/sh
# pbsgo-gui: launch the Proxmox Backup Client GUI as the current user.
#
# Note: machine (whole-disk) backup needs root privileges. To run the
# GUI elevated, use pbsgo-gui-root instead.
exec @LIBDIR@/pbsgo/pbsgo-gui "$@"
EOF
sed -i "s|@LIBDIR@|%{_libdir}|g" %{buildroot}%{_bindir}/pbsgo-gui
chmod 0755 %{buildroot}%{_bindir}/pbsgo-gui

cat > %{buildroot}%{_bindir}/pbsgo-gui-root <<'EOF'
#!/bin/sh
# pbsgo-gui-root: launch the Proxmox Backup Client GUI with root
# privileges, which machine (whole-disk) backup requires.
if [ "$(id -u)" = "0" ]; then
    exec @LIBDIR@/pbsgo/pbsgo-gui "$@"
fi

if command -v sudo >/dev/null 2>&1; then
    if [ -n "${DISPLAY:-}" ]; then
        # Keep the X11 session reachable from the elevated process.
        exec sudo env "DISPLAY=$DISPLAY" "XAUTHORITY=${XAUTHORITY:-$HOME/.Xauthority}" @LIBDIR@/pbsgo/pbsgo-gui "$@"
    fi
    exec sudo @LIBDIR@/pbsgo/pbsgo-gui "$@"
fi

exec pkexec @LIBDIR@/pbsgo/pbsgo-gui "$@"
EOF
sed -i "s|@LIBDIR@|%{_libdir}|g" %{buildroot}%{_bindir}/pbsgo-gui-root
chmod 0755 %{buildroot}%{_bindir}/pbsgo-gui-root

%files
%license LICENSE
%doc README.md CHANGELOG.md
%{_bindir}/pbsgo
%{_bindir}/pbsgo-machine
%{_bindir}/pbsgo-nbd
%{_bindir}/pbsgo-gui
%{_bindir}/pbsgo-gui-root
%{_libdir}/pbsgo/pbsgo-gui
%{_datadir}/applications/pbsgo-gui.desktop
%{_datadir}/icons/hicolor/256x256/apps/pbsgo.png

%changelog
* Mon Sep 07 2026 Proxmox Backup Client contributors <noreply@localhost> - 0.2.119-1
- Initial RPM package
