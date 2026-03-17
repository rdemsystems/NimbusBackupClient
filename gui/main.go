package main

import (
	"fmt"
	"image/color"
	"log"
	"net/url"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/data/binding"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
)

const (
	appName    = "Proxmox Backup Guardian"
	appVersion = "1.0.0"
)

type BackupGUI struct {
	app    fyne.App
	window fyne.Window
	config *Config

	// Bindings for UI updates
	statusText    binding.String
	progressValue binding.Float

	// UI Elements
	urlEntry         *widget.Entry
	fingerprintEntry *widget.Entry
	authIDEntry      *widget.Entry
	secretEntry      *widget.Entry
	datastoreEntry   *widget.Entry
	namespaceEntry   *widget.Entry
	backupDirEntry   *widget.Entry
	progressBar      *widget.ProgressBar
	statusLabel      *widget.Label
	startButton      *widget.Button
	stopButton       *widget.Button
}

func main() {
	myApp := app.NewWithID("com.rdem-systems.backup-guardian")
	myApp.Settings().SetTheme(&customTheme{})

	gui := &BackupGUI{
		app:           myApp,
		statusText:    binding.NewString(),
		progressValue: binding.NewFloat(),
	}

	// Load or create config
	gui.config = LoadConfig()

	gui.window = myApp.NewWindow(fmt.Sprintf("%s v%s", appName, appVersion))
	gui.window.Resize(fyne.NewSize(700, 600))
	gui.window.SetMaster()

	gui.buildUI()
	gui.window.ShowAndRun()
}

func (g *BackupGUI) buildUI() {
	// Header
	header := container.NewVBox(
		widget.NewLabelWithStyle(
			"Proxmox Backup Guardian",
			fyne.TextAlignCenter,
			fyne.TextStyle{Bold: true},
		),
		widget.NewLabelWithStyle(
			"Client de sauvegarde pour Proxmox Backup Server",
			fyne.TextAlignCenter,
			fyne.TextStyle{Italic: true},
		),
		widget.NewSeparator(),
	)

	// Tabs
	tabs := container.NewAppTabs(
		container.NewTabItem("Configuration PBS", g.buildPBSConfigTab()),
		container.NewTabItem("Sauvegarde", g.buildBackupTab()),
		container.NewTabItem("À propos", g.buildAboutTab()),
	)

	content := container.NewBorder(header, nil, nil, nil, tabs)
	g.window.SetContent(content)
}

func (g *BackupGUI) buildPBSConfigTab() fyne.CanvasObject {
	// PBS Server Configuration
	g.urlEntry = widget.NewEntry()
	g.urlEntry.SetPlaceHolder("https://pbs.example.com:8007")
	g.urlEntry.SetText(g.config.BaseURL)

	g.fingerprintEntry = widget.NewEntry()
	g.fingerprintEntry.SetPlaceHolder("AA:BB:CC:DD:...")
	g.fingerprintEntry.SetText(g.config.CertFingerprint)

	g.authIDEntry = widget.NewEntry()
	g.authIDEntry.SetPlaceHolder("backup@pbs!token-name")
	g.authIDEntry.SetText(g.config.AuthID)

	g.secretEntry = widget.NewPasswordEntry()
	g.secretEntry.SetPlaceHolder("xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx")
	g.secretEntry.SetText(g.config.Secret)

	g.datastoreEntry = widget.NewEntry()
	g.datastoreEntry.SetPlaceHolder("backup-prod")
	g.datastoreEntry.SetText(g.config.Datastore)

	g.namespaceEntry = widget.NewEntry()
	g.namespaceEntry.SetPlaceHolder("production (optionnel)")
	g.namespaceEntry.SetText(g.config.Namespace)

	form := &widget.Form{
		Items: []*widget.FormItem{
			{Text: "URL du serveur PBS:", Widget: g.urlEntry},
			{Text: "Empreinte certificat SSL:", Widget: g.fingerprintEntry},
			{Text: "Authentication ID:", Widget: g.authIDEntry},
			{Text: "Secret (API Token):", Widget: g.secretEntry},
			{Text: "Datastore:", Widget: g.datastoreEntry},
			{Text: "Namespace:", Widget: g.namespaceEntry},
		},
		OnSubmit: func() {
			g.saveConfig()
		},
	}

	testButton := widget.NewButtonWithIcon("Tester la connexion", theme.ConfirmIcon(), func() {
		g.testConnection()
	})

	saveButton := widget.NewButtonWithIcon("Enregistrer", theme.DocumentSaveIcon(), func() {
		g.saveConfig()
	})

	buttons := container.NewHBox(testButton, saveButton)

	infoLabel := widget.NewLabel("💡 Obtenez votre API Token depuis l'interface PBS:\nConfiguration → Access Control → API Tokens")
	infoLabel.Wrapping = fyne.TextWrapWord

	return container.NewVBox(
		form,
		widget.NewSeparator(),
		buttons,
		widget.NewSeparator(),
		infoLabel,
	)
}

func (g *BackupGUI) buildBackupTab() fyne.CanvasObject {
	// Backup Directory Selection
	g.backupDirEntry = widget.NewEntry()
	g.backupDirEntry.SetPlaceHolder("/path/to/backup ou C:\\Data")
	g.backupDirEntry.SetText(g.config.BackupDir)

	browseButton := widget.NewButtonWithIcon("Parcourir", theme.FolderOpenIcon(), func() {
		dialog.ShowFolderOpen(func(dir fyne.ListableURI, err error) {
			if err == nil && dir != nil {
				g.backupDirEntry.SetText(dir.Path())
			}
		}, g.window)
	})

	backupDirRow := container.NewBorder(nil, nil, nil, browseButton, g.backupDirEntry)

	// Backup ID
	backupIDEntry := widget.NewEntry()
	backupIDEntry.SetPlaceHolder("Laissez vide pour utiliser le hostname")
	backupIDEntry.SetText(g.config.BackupID)

	// VSS Option (Windows)
	vssCheck := widget.NewCheck("Utiliser VSS (Windows)", nil)
	vssCheck.SetChecked(g.config.UseVSS)

	form := &widget.Form{
		Items: []*widget.FormItem{
			{Text: "Répertoire à sauvegarder:", Widget: backupDirRow},
			{Text: "Backup ID:", Widget: backupIDEntry},
			{Text: "Options:", Widget: vssCheck},
		},
	}

	// Progress
	g.progressBar = widget.NewProgressBar()
	g.progressBar.Bind(g.progressValue)

	g.statusLabel = widget.NewLabel("Prêt")
	g.statusLabel.Bind(g.statusText)

	// Buttons
	g.startButton = widget.NewButtonWithIcon("Démarrer la sauvegarde", theme.MediaPlayIcon(), func() {
		g.startBackup()
	})
	g.startButton.Importance = widget.HighImportance

	g.stopButton = widget.NewButtonWithIcon("Arrêter", theme.MediaStopIcon(), func() {
		g.stopBackup()
	})
	g.stopButton.Disable()

	buttons := container.NewHBox(g.startButton, g.stopButton)

	// Stats placeholder
	statsLabel := widget.NewLabel("Statistiques apparaîtront ici pendant le backup...")
	statsLabel.Wrapping = fyne.TextWrapWord

	return container.NewVBox(
		form,
		widget.NewSeparator(),
		widget.NewLabel("Progression:"),
		g.progressBar,
		g.statusLabel,
		widget.NewSeparator(),
		buttons,
		widget.NewSeparator(),
		statsLabel,
	)
}

func (g *BackupGUI) buildAboutTab() fyne.CanvasObject {
	logo := widget.NewLabelWithStyle(
		"🛡️ Proxmox Backup Guardian",
		fyne.TextAlignCenter,
		fyne.TextStyle{Bold: true},
	)

	version := widget.NewLabel(fmt.Sprintf("Version %s", appVersion))
	version.Alignment = fyne.TextAlignCenter

	description := widget.NewLabel(
		"Client de sauvegarde pour Proxmox Backup Server\n\n" +
			"Fonctionnalités:\n" +
			"• Sauvegarde de répertoires\n" +
			"• Support VSS (Windows)\n" +
			"• Déduplication et compression\n" +
			"• Interface utilisateur simple\n\n" +
			"Basé sur proxmoxbackupclient_go\n" +
			"https://github.com/tizbac/proxmoxbackupclient_go",
	)
	description.Wrapping = fyne.TextWrapWord
	description.Alignment = fyne.TextAlignCenter

	copyright := widget.NewLabel("© 2024 RDEM Systems")
	copyright.Alignment = fyne.TextAlignCenter

	website := widget.NewHyperlink(
		"backup.rdem-systems.com",
		parseURL("https://backup.rdem-systems.com"),
	)
	website.Alignment = fyne.TextAlignCenter

	return container.NewVBox(
		widget.NewSeparator(),
		logo,
		version,
		widget.NewSeparator(),
		description,
		widget.NewSeparator(),
		copyright,
		website,
		widget.NewSeparator(),
	)
}

func (g *BackupGUI) saveConfig() {
	g.config.BaseURL = g.urlEntry.Text
	g.config.CertFingerprint = g.fingerprintEntry.Text
	g.config.AuthID = g.authIDEntry.Text
	g.config.Secret = g.secretEntry.Text
	g.config.Datastore = g.datastoreEntry.Text
	g.config.Namespace = g.namespaceEntry.Text
	g.config.BackupDir = g.backupDirEntry.Text

	if err := g.config.Save(); err != nil {
		dialog.ShowError(err, g.window)
		return
	}

	dialog.ShowInformation("Succès", "Configuration enregistrée avec succès!", g.window)
}

func (g *BackupGUI) testConnection() {
	_ = g.statusText.Set("Test de connexion en cours...")

	// TODO: Implement actual PBS connection test
	// For now, just show a placeholder

	dialog.ShowInformation(
		"Test de connexion",
		"Fonctionnalité à implémenter:\n"+
			"• Connexion au serveur PBS\n"+
			"• Vérification du certificat SSL\n"+
			"• Test de l'authentification\n"+
			"• Liste des datastores disponibles",
		g.window,
	)

	_ = g.statusText.Set("Prêt")
}

func (g *BackupGUI) startBackup() {
	g.startButton.Disable()
	g.stopButton.Enable()
	_ = g.statusText.Set("Démarrage de la sauvegarde...")
	_ = g.progressValue.Set(0.0)

	// TODO: Implement actual backup logic by calling directorybackup code
	// For now, simulate progress
	go g.simulateBackup()
}

func (g *BackupGUI) stopBackup() {
	_ = g.statusText.Set("Arrêt de la sauvegarde...")
	// TODO: Implement backup cancellation
	g.resetBackupUI()
}

func (g *BackupGUI) simulateBackup() {
	// Placeholder simulation
	for i := 0; i <= 100; i++ {
		_ = g.progressValue.Set(float64(i) / 100.0)
		_ = g.statusText.Set(fmt.Sprintf("Sauvegarde en cours... %d%%", i))
		time.Sleep(100 * time.Millisecond)
	}

	_ = g.statusText.Set("Sauvegarde terminée avec succès!")
	g.resetBackupUI()

	dialog.ShowInformation("Succès", "Sauvegarde terminée avec succès!", g.window)
}

func (g *BackupGUI) resetBackupUI() {
	g.startButton.Enable()
	g.stopButton.Disable()
}

func parseURL(urlStr string) *url.URL {
	link, err := url.Parse(urlStr)
	if err != nil {
		log.Println("Could not parse URL:", err)
	}
	return link
}

// Custom theme
type customTheme struct{}

func (t *customTheme) Color(name fyne.ThemeColorName, variant fyne.ThemeVariant) color.Color {
	if name == theme.ColorNamePrimary {
		return color.NRGBA{R: 41, G: 128, B: 185, A: 255} // Blue
	}
	return theme.DefaultTheme().Color(name, variant)
}

func (t *customTheme) Icon(name fyne.ThemeIconName) fyne.Resource {
	return theme.DefaultTheme().Icon(name)
}

func (t *customTheme) Font(style fyne.TextStyle) fyne.Resource {
	return theme.DefaultTheme().Font(style)
}

func (t *customTheme) Size(name fyne.ThemeSizeName) float32 {
	return theme.DefaultTheme().Size(name)
}
