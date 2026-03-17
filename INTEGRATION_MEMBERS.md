# Intégration members.rdem-systems.com ↔ GUI Backup

## 🎯 Objectif

Le site **members.rdem-systems.com** génère des fichiers de configuration (JSON/INI) que les clients téléchargent et importent dans la GUI Backup.

## 📊 Workflow complet

```
┌─────────────────────────────────────────────────────────────────┐
│ 1. Client achète un abonnement sur vault-backup-guardian       │
│    → Choisit offre: Drive Bank PBS 2TB                         │
└──────────────────────────┬──────────────────────────────────────┘
                           │
                           ▼
┌─────────────────────────────────────────────────────────────────┐
│ 2. RDEM Systems configure le compte PBS côté serveur           │
│    → Crée datastore: acme-corp-2tb                             │
│    → Crée API Token: acme-corp@pbs!backup-production           │
│    → Enregistre credentials en BDD Laravel                     │
└──────────────────────────┬──────────────────────────────────────┘
                           │
                           ▼
┌─────────────────────────────────────────────────────────────────┐
│ 3. Client se connecte sur members.rdem-systems.com             │
│    → Onglet "Mes Services" → "Backup PBS"                      │
│    → Bouton "Télécharger configuration"                        │
│    → Choix format: JSON ou INI                                 │
└──────────────────────────┬──────────────────────────────────────┘
                           │
                           ▼
┌─────────────────────────────────────────────────────────────────┐
│ 4. Laravel génère le fichier avec credentials PBS              │
│    → backup-acme-corp.json                                      │
│    → Contient: URL PBS, token, datastore, etc.                 │
└──────────────────────────┬──────────────────────────────────────┘
                           │
                           ▼
┌─────────────────────────────────────────────────────────────────┐
│ 5. Client télécharge aussi la GUI Backup                       │
│    → proxmox-backup-gui.exe (Windows)                          │
│    → proxmox-backup-gui (Linux)                                │
└──────────────────────────┬──────────────────────────────────────┘
                           │
                           ▼
┌─────────────────────────────────────────────────────────────────┐
│ 6. Client installe la GUI sur son serveur                      │
│    → Lance proxmox-backup-gui.exe                              │
│    → Importe le fichier backup-acme-corp.json                  │
│    → ✅ Config PBS pré-remplie automatiquement                  │
└──────────────────────────┬──────────────────────────────────────┘
                           │
                           ▼
┌─────────────────────────────────────────────────────────────────┐
│ 7. Client configure son backup dans la GUI                     │
│    → Sélectionne dossiers: C:\Data, C:\WebServer               │
│    → Définit exclusions: *.tmp, node_modules/                  │
│    → Choisit planification: Quotidien 2h                       │
│    → Active le job                                              │
└──────────────────────────┬──────────────────────────────────────┘
                           │
                           ▼
┌─────────────────────────────────────────────────────────────────┐
│ 8. GUI créé une tâche planifiée (cron/Task Scheduler)          │
│    → Windows: Task Scheduler                                    │
│    → Linux: crontab                                             │
│    → Backup s'exécute automatiquement à l'heure prévue         │
└─────────────────────────────────────────────────────────────────┘
```

## 🔧 Implémentation Laravel

### 1. Migration BDD

```php
<?php

use Illuminate\Database\Migrations\Migration;
use Illuminate\Database\Schema\Blueprint;
use Illuminate\Support\Facades\Schema;

return new class extends Migration
{
    public function up()
    {
        Schema::create('backup_subscriptions', function (Blueprint $table) {
            $table->id();
            $table->foreignId('client_context_id')->constrained()->onDelete('cascade');

            // PBS Server Info
            $table->string('pbs_server_url'); // pbs-fr-paris.rdem-systems.com
            $table->text('pbs_ssl_fingerprint');

            // Client credentials
            $table->string('pbs_username');      // acme-corp
            $table->string('pbs_token_name');    // backup-production
            $table->text('pbs_api_secret');      // UUID
            $table->string('datastore_name');    // acme-corp-2tb
            $table->string('namespace')->nullable();

            // Subscription details
            $table->string('plan_name');         // Drive Bank PBS 2TB
            $table->integer('storage_quota_gb'); // 2048
            $table->date('expires_at')->nullable();

            // Retention policy (optionnel, peut override)
            $table->integer('retention_last')->default(7);
            $table->integer('retention_daily')->default(14);
            $table->integer('retention_weekly')->default(8);
            $table->integer('retention_monthly')->default(12);

            $table->timestamps();
        });
    }

    public function down()
    {
        Schema::dropIfExists('backup_subscriptions');
    }
};
```

### 2. Modèle Eloquent

```php
<?php

namespace App\Models;

use Illuminate\Database\Eloquent\Model;

class BackupSubscription extends Model
{
    protected $fillable = [
        'client_context_id',
        'pbs_server_url',
        'pbs_ssl_fingerprint',
        'pbs_username',
        'pbs_token_name',
        'pbs_api_secret',
        'datastore_name',
        'namespace',
        'plan_name',
        'storage_quota_gb',
        'expires_at',
        'retention_last',
        'retention_daily',
        'retention_weekly',
        'retention_monthly',
    ];

    protected $casts = [
        'expires_at' => 'date',
    ];

    public function clientContext()
    {
        return $this->belongsTo(ClientContext::class);
    }

    public function isActive(): bool
    {
        return !$this->expires_at || $this->expires_at->isFuture();
    }
}
```

### 3. Contrôleur

Voir **CONFIG_FORMAT_SPEC.md** section "Contrôleur Laravel" pour le code complet.

### 4. Routes

```php
<?php

// routes/web.php

Route::middleware(['auth'])->group(function () {
    Route::prefix('services/backup')->group(function () {
        Route::get('/', [BackupController::class, 'index'])->name('backup.index');
        Route::get('/download-config', [BackupController::class, 'downloadConfig'])->name('backup.download-config');
        Route::get('/download-client/{platform}', [BackupController::class, 'downloadClient'])->name('backup.download-client');
    });
});
```

### 5. Vue Blade

```blade
{{-- resources/views/services/backup.blade.php --}}

@extends('layouts.app')

@section('content')
<div class="container">
    <h1>Mon Service de Backup PBS</h1>

    @if($subscription && $subscription->isActive())
        <div class="card mb-4">
            <div class="card-header bg-primary text-white">
                <h3>{{ $subscription->plan_name }}</h3>
            </div>
            <div class="card-body">
                <div class="row">
                    <div class="col-md-6">
                        <p><strong>Serveur PBS:</strong> {{ $subscription->pbs_server_url }}</p>
                        <p><strong>Datastore:</strong> {{ $subscription->datastore_name }}</p>
                        <p><strong>Quota:</strong> {{ $subscription->storage_quota_gb }} GB</p>
                        <p><strong>Expire le:</strong> {{ $subscription->expires_at?->format('d/m/Y') ?? 'Jamais' }}</p>
                    </div>
                    <div class="col-md-6">
                        <p><strong>Rétention:</strong></p>
                        <ul>
                            <li>{{ $subscription->retention_last }} derniers backups</li>
                            <li>{{ $subscription->retention_daily }} backups quotidiens</li>
                            <li>{{ $subscription->retention_weekly }} backups hebdomadaires</li>
                            <li>{{ $subscription->retention_monthly }} backups mensuels</li>
                        </ul>
                    </div>
                </div>
            </div>
        </div>

        <div class="card mb-4">
            <div class="card-header">
                <h4>📥 Télécharger le client de backup</h4>
            </div>
            <div class="card-body">
                <p>Téléchargez le client Proxmox Backup Guardian pour votre système d'exploitation :</p>

                <div class="btn-group mb-3" role="group">
                    <a href="{{ route('backup.download-client', 'windows') }}"
                       class="btn btn-lg btn-primary">
                        <i class="fab fa-windows"></i> Windows (.exe)
                    </a>
                    <a href="{{ route('backup.download-client', 'linux') }}"
                       class="btn btn-lg btn-success">
                        <i class="fab fa-linux"></i> Linux
                    </a>
                    <a href="{{ route('backup.download-client', 'macos') }}"
                       class="btn btn-lg btn-secondary">
                        <i class="fab fa-apple"></i> macOS
                    </a>
                </div>

                <div class="alert alert-info">
                    <strong>💡 Deux versions disponibles :</strong>
                    <ul>
                        <li><strong>CLI</strong> (léger, ~10MB) : pour scripts et automatisation</li>
                        <li><strong>GUI</strong> (avec interface, ~40MB) : pour configuration visuelle</li>
                    </ul>
                </div>
            </div>
        </div>

        <div class="card mb-4">
            <div class="card-header">
                <h4>⚙️ Fichier de configuration</h4>
            </div>
            <div class="card-body">
                <p>Téléchargez votre fichier de configuration pré-rempli :</p>

                <div class="btn-group" role="group">
                    <a href="{{ route('backup.download-config', ['format' => 'json']) }}"
                       class="btn btn-primary">
                        <i class="fas fa-download"></i> Format JSON (recommandé)
                    </a>
                    <a href="{{ route('backup.download-config', ['format' => 'ini']) }}"
                       class="btn btn-outline-primary">
                        <i class="fas fa-download"></i> Format INI
                    </a>
                </div>

                <div class="mt-3">
                    <small class="text-muted">
                        Ce fichier contient vos identifiants PBS. Importez-le dans la GUI pour configuration automatique.
                    </small>
                </div>
            </div>
        </div>

        <div class="card">
            <div class="card-header">
                <h4>📖 Guide d'installation</h4>
            </div>
            <div class="card-body">
                <ol>
                    <li>
                        <strong>Téléchargez le client</strong> pour votre OS (Windows/Linux/macOS)
                    </li>
                    <li>
                        <strong>Installez-le</strong> sur le serveur à sauvegarder
                    </li>
                    <li>
                        <strong>Téléchargez votre fichier de configuration</strong> (JSON recommandé)
                    </li>
                    <li>
                        <strong>Lancez la GUI</strong> : <code>proxmox-backup-gui</code>
                    </li>
                    <li>
                        <strong>Importez la config</strong> : Fichier → Importer configuration
                    </li>
                    <li>
                        <strong>Configurez vos backups</strong> :
                        <ul>
                            <li>Sélectionnez les dossiers à sauvegarder</li>
                            <li>Définissez des exclusions (*.tmp, logs, etc.)</li>
                            <li>Choisissez la planification (quotidien, hebdo...)</li>
                        </ul>
                    </li>
                    <li>
                        <strong>Activez le job</strong> : la sauvegarde se fera automatiquement
                    </li>
                </ol>

                <div class="alert alert-success mt-3">
                    <strong>✅ Backup-ID automatique :</strong><br>
                    Format: <code>{{ strtolower(Str::slug($context->company_name)) }}-[hostname]</code><br>
                    Exemple: <code>acme-corp-srv-web-01</code>
                </div>
            </div>
        </div>

    @else
        <div class="alert alert-warning">
            <h4>Aucun abonnement backup actif</h4>
            <p>Souscrivez à une offre de backup pour accéder à cette fonctionnalité.</p>
            <a href="{{ route('tarifs') }}" class="btn btn-primary">Voir les offres</a>
        </div>
    @endif
</div>
@endsection
```

## 📝 Backup-ID : Format recommandé

### Structure
```
{company-slug}-{hostname}-{description}
```

### Exemples

| Client | Serveur | Backup-ID |
|--------|---------|-----------|
| ACME Corp | Serveur Web | `acme-corp-srv-web-01` |
| ACME Corp | Base de données | `acme-corp-db-prod` |
| TechStart | Application | `techstart-app-prod` |
| Worldline | Backup complet | `worldline-full-backup` |

### Génération automatique

**Option 1 : Préfixe seulement** (hostname ajouté par client)
```php
'backup-id' => Str::slug($context->company_name)
// → "acme-corp"
// Le client ajoutera: "acme-corp-srv-web-01" dans la GUI
```

**Option 2 : Complet avec suggestion**
```php
'backup-id' => Str::slug($context->company_name) . '-server-01'
// → "acme-corp-server-01"
// Le client peut modifier dans la GUI
```

**Option 3 : Vide (auto-détection)**
```php
'backup-id' => ''
// → La GUI détectera automatiquement le hostname: "WIN-ABC123" ou "debian-server"
```

## 🎨 Format final du JSON téléchargé

```json
{
  "id": "job_acme_1710700800",
  "name": "Backup ACME Corp",
  "description": "Configuration automatique générée par members.rdem-systems.com",
  "enabled": true,

  "pbs_config": {
    "baseurl": "https://pbs-fr-paris.rdem-systems.com:8007",
    "certfingerprint": "5A:3B:...",
    "authid": "acme-corp@pbs!backup-production",
    "secret": "a1b2c3d4-...",
    "datastore": "acme-corp-2tb",
    "namespace": "production",
    "backup-id": "acme-corp",
    "backupdir": "",
    "usevss": true
  },

  "folders": [],
  "exclusions": ["*.tmp", "*.log", "~*"],
  "schedule": "Quotidien (2h du matin)",
  "schedule_cron": "0 2 * * *",

  "keep_last": 7,
  "keep_daily": 30,
  "keep_weekly": 12,
  "keep_monthly": 24
}
```

---

**Intégration complète prête ! 🚀**
