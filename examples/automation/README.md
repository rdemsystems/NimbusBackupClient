# Unattended deployment & automation

🇬🇧 English | *(résumé FR en bas)*

Nimbus Backup can be deployed and configured entirely from files — no clicks
required. This is designed for Ansible, but any config-management tool (Salt,
DSC, GPO, a login script…) works the same way. There are two paths; pick one.

---

## Path A — Command-line (`directorybackup.exe`) — recommended for IaC

One JSON file carries **everything for a backup**; scheduling is delegated to the
Windows Task Scheduler.

```powershell
directorybackup.exe --config "C:\ProgramData\NimbusBackup\backup.json"
```

- Config: [`cli-single-config.json`](cli-single-config.json) (all fields).
- Every field can be overridden by a CLI flag (flags win over the file) — handy
  for a shared template with per-host overrides.
- **Exit codes are reliable**, so your orchestrator can detect failures:

  | code | meaning |
  |------|---------|
  | `0`  | success |
  | `1`  | fatal failure (connection, auth, mail, …) |
  | `2`  | another backup is already running (lock) |
  | `3`  | completed **partially** — some files were unreadable; the snapshot is incomplete |

Ansible example: [`ansible/deploy-cli.yml`](ansible/deploy-cli.yml)
(+ template [`cli-single-config.json.j2`](ansible/cli-single-config.json.j2)).
It copies the binary, renders the config, and registers a daily
`win_scheduled_task` running as `SYSTEM` at `run_level: highest` (required for VSS).

---

## Path B — GUI/service (Nimbus Backup MSI) — one file for connection **and** schedule

The MSI ships a Windows service. A **single `config.json`** can now carry the PBS
connection, the backup settings **and** the schedule, via a `scheduled_jobs`
array. On (re)start the service reconciles those jobs into its store and
**auto-computes each `nextRun`** — you don't compute timestamps in your template.

- Config: [`gui-service-config.json`](gui-service-config.json).
- Location: `C:\ProgramData\NimbusBackup\config.json`.
- Drop the file, restart the `NimbusBackup` service, done.

Ansible example: [`ansible/deploy-gui-service.yml`](ansible/deploy-gui-service.yml)
(+ template [`ansible/gui-service-config.json.j2`](ansible/gui-service-config.json.j2)).

### How provisioning behaves (idempotency)

- Jobs are matched **by `id`**. Re-pushing the same config **upserts** — it does
  not duplicate, and an unchanged schedule is **not** re-fired (a still-valid
  `nextRun` is preserved).
- Jobs created interactively in the GUI (whose ids you don't list) are **left
  untouched**.
- If `id` is omitted, a stable one is derived from `name`.
- `nextRun` is optional in your file: leave it out and the service fills it in.

---

## Field reference

### Connection (`pbs_servers[*]` or top-level legacy keys)

| key | notes |
|-----|-------|
| `baseurl` | `https://host:8007` |
| `certfingerprint` | SHA-256 fingerprint (`AA:BB:…`); required for self-signed certs |
| `authid` | PBS API token, e.g. `user@pbs!tokenname` |
| `secret` | token secret (keep in Ansible Vault) |
| `datastore` | target datastore |
| `namespace` | optional namespace |

### Scheduled job (`scheduled_jobs[*]`, GUI/service path)

| key | notes |
|-----|-------|
| `id` | stable id; derived from `name` if omitted |
| `name` | display name |
| `scheduleTime` | `HH:MM`, 24h, local time |
| `enabled` | must be `true` to run |
| `backupType` | `host` (folders) or `vm` (whole machine) |
| `backupDirs` | list of paths, e.g. `["C:"]` |
| `backupId` | snapshot id (defaults to hostname) |
| `useVSS` | VSS-consistent snapshot |
| `compression` | `fastest` \| `default` \| `better` \| `best` |
| `excludeList` | optional exclusion patterns |
| `runAtStartup` | also run once at service start |
| `nextRun` | optional — auto-computed if empty |

> **Secrets:** never commit real tokens. Use Ansible Vault
> (`{{ vault_pbs_secret }}` in the examples). On disk, `config.json` is written
> `0600` and the token is stripped before it ever reaches the GUI frontend.

---

## Résumé (FR)

Deux voies pour un déploiement sans clic :

- **Voie A — ligne de commande** (`directorybackup.exe --config fichier.json`) :
  un seul fichier JSON pour toute la sauvegarde, planification via le
  Planificateur de tâches Windows. Codes de retour fiables (`0` OK, `≠0` échec ou
  sauvegarde partielle). Idéale pour l'infrastructure-as-code.
- **Voie B — service/GUI (MSI)** : un **unique `config.json`** contenant la
  connexion PBS **et** la planification (`scheduled_jobs`). Poussez le fichier,
  redémarrez le service `NimbusBackup` : les jobs sont pris en compte et le
  `nextRun` est calculé automatiquement. Réconciliation idempotente (upsert par
  `id`, ne touche pas aux jobs créés dans l'interface).

Exemples Ansible prêts à l'emploi dans [`ansible/`](ansible/).
