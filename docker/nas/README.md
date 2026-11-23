# Salvanza on a NAS (Docker)

Folder backups of a NAS's shares to a Proxmox Backup Server, managed from a web browser.

The container runs the Salvanza **background service**, which keeps the Backup Sets and runs them on schedule, and serves the **text window** (`salvanza-cli tui`, see [docs/manual/TUI.md](../../docs/manual/TUI.md)) on port 7681 behind a login. From the text window you can see the dashboard, the Backup Sets and their schedules, backups running now, history, restores, servers and keys, reports and logs.

Tested on: see the supported NAS list in #164.

## Synology (Container Manager)

1. Copy this folder to the NAS, for example `/volume1/docker/salvanza-nas`.
2. Put the web password in a file named `ui-password` in that folder, readable only by root (`chmod 600`).
3. Edit `docker-compose.yml`: add one volume line per share to back up, as `/volume1/<share>:/data/<share>:ro`, and set `TZ`.
4. **Container Manager > Project > Create**, with that folder as the path. It builds the image the first time. Or pull a prebuilt image and change `build: .` to `image: ghcr.io/<owner>/salvanza-nas:dev`.
5. Open `http://<nas>:7681`, user `admin`. For HTTPS, put it behind **Control Panel > Login Portal > Advanced > Reverse Proxy**. The web window has full control of the backups.

In the text window:

- **Servers:** add your PBS (URL, API token, fingerprint).
- **Backup Sets:** create a set with the folders under `/data`, and a schedule. Turn **Use snapshots** off, because there is no snapshot module in a container.
- **Restores:** write to `/restore`, the share mounted read-write for them.

## Other commands

```sh
docker exec -it salvanza-nas /usr/local/bin/entrypoint.sh tui   # the window in a terminal
docker run --rm -it salvanza-nas:dev demo                       # look around, invented data
docker logs salvanza-nas                                        # service and window log
```

## Notes

- **Consistency:** files are read live. Schedule backups when the shares are quiet, or back up a Btrfs snapshot that the NAS has mounted.
- **Permissions:** the container runs as root to read every file of the mounted shares. Mount them read-only.
- **Architecture:** tested on linux/amd64. The Dockerfile should build on linux/arm64 too, but that is untested. 32-bit ARM NAS models are not covered.
