package main

import (
	"fmt"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/data/binding"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
)

// BackupConfigUI provides advanced backup configuration
type BackupConfigUI struct {
	window fyne.Window

	// Backup type
	backupType *widget.Select

	// Multi-folder selection
	folders      []string
	foldersList  *widget.List
	foldersData  binding.StringList

	// Multi-disk selection (for machine backup)
	disks      []DiskInfo
	disksList  *widget.List
	disksData  binding.StringList

	// Exclusions
	exclusions     []string
	exclusionsList *widget.List
	exclusionsData binding.StringList

	// Schedule
	scheduleEnabled *widget.Check
	scheduleSelect  *widget.Select
	customCron      *widget.Entry

	// Retention
	keepLast    *widget.Entry
	keepDaily   *widget.Entry
	keepWeekly  *widget.Entry
	keepMonthly *widget.Entry

	// Advanced options
	compressionSelect *widget.Select
	chunkSizeSelect   *widget.Select
	bandwidthLimit    *widget.Entry
	parallelUploads   *widget.Entry
}

type DiskInfo struct {
	DevicePath string
	Name       string
	Size       string
	Model      string
}

func NewBackupConfigUI(window fyne.Window) *BackupConfigUI {
	ui := &BackupConfigUI{
		window:         window,
		folders:        []string{},
		foldersData:    binding.NewStringList(),
		disks:          []DiskInfo{},
		disksData:      binding.NewStringList(),
		exclusions:     []string{},
		exclusionsData: binding.NewStringList(),
	}

	return ui
}

func (ui *BackupConfigUI) BuildAdvancedBackupTab() fyne.CanvasObject {
	// Backup type selection
	ui.backupType = widget.NewSelect(
		[]string{"Répertoires", "Disque complet", "Stream (SQL, etc.)"},
		func(selected string) {
			// Update UI based on type
		},
	)
	ui.backupType.SetSelected("Répertoires")

	// Tabs for different sections
	tabs := container.NewAppTabs(
		container.NewTabItem("Source", ui.buildSourceTab()),
		container.NewTabItem("Exclusions", ui.buildExclusionsTab()),
		container.NewTabItem("Planification", ui.buildScheduleTab()),
		container.NewTabItem("Rétention", ui.buildRetentionTab()),
		container.NewTabItem("Avancé", ui.buildAdvancedTab()),
	)

	header := container.NewVBox(
		widget.NewLabel("Type de sauvegarde :"),
		ui.backupType,
		widget.NewSeparator(),
	)

	return container.NewBorder(header, nil, nil, nil, tabs)
}

func (ui *BackupConfigUI) buildSourceTab() fyne.CanvasObject {
	// Multi-folder list
	ui.foldersList = widget.NewListWithData(
		ui.foldersData,
		func() fyne.CanvasObject {
			return container.NewHBox(
				widget.NewIcon(theme.FolderIcon()),
				widget.NewLabel("template"),
			)
		},
		func(item binding.DataItem, obj fyne.CanvasObject) {
			strItem := item.(binding.String)
			str, _ := strItem.Get()
			obj.(*fyne.Container).Objects[1].(*widget.Label).SetText(str)
		},
	)

	addFolderBtn := widget.NewButtonWithIcon("Ajouter un dossier", theme.FolderOpenIcon(), func() {
		dialog.ShowFolderOpen(func(dir fyne.ListableURI, err error) {
			if err == nil && dir != nil {
				ui.folders = append(ui.folders, dir.Path())
				_ = ui.foldersData.Append(dir.Path())
			}
		}, ui.window)
	})

	removeFolderBtn := widget.NewButtonWithIcon("Retirer", theme.DeleteIcon(), func() {
		if len(ui.folders) > 0 {
			// Remove selected folder (simplified)
			ui.folders = ui.folders[:len(ui.folders)-1]
			_ = ui.foldersData.Set(ui.folders)
		}
	})

	folderButtons := container.NewHBox(addFolderBtn, removeFolderBtn)

	// Disk selection (for machine backup)
	ui.disksList = widget.NewListWithData(
		ui.disksData,
		func() fyne.CanvasObject {
			return container.NewHBox(
				widget.NewIcon(theme.StorageIcon()),
				widget.NewLabel("template"),
			)
		},
		func(item binding.DataItem, obj fyne.CanvasObject) {
			strItem := item.(binding.String)
			str, _ := strItem.Get()
			obj.(*fyne.Container).Objects[1].(*widget.Label).SetText(str)
		},
	)

	detectDisksBtn := widget.NewButtonWithIcon("Détecter les disques", theme.SearchIcon(), func() {
		ui.detectDisks()
	})

	diskButtons := container.NewHBox(detectDisksBtn)

	// Show folders or disks based on backup type
	foldersSection := container.NewBorder(
		widget.NewLabelWithStyle("Dossiers à sauvegarder :", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
		folderButtons,
		nil, nil,
		ui.foldersList,
	)

	disksSection := container.NewBorder(
		widget.NewLabelWithStyle("Disques à sauvegarder :", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
		diskButtons,
		nil, nil,
		ui.disksList,
	)

	return container.NewVBox(
		foldersSection,
		widget.NewSeparator(),
		disksSection,
	)
}

func (ui *BackupConfigUI) buildExclusionsTab() fyne.CanvasObject {
	// Exclusions list
	ui.exclusionsList = widget.NewListWithData(
		ui.exclusionsData,
		func() fyne.CanvasObject {
			return container.NewHBox(
				widget.NewIcon(theme.CancelIcon()),
				widget.NewLabel("template"),
			)
		},
		func(item binding.DataItem, obj fyne.CanvasObject) {
			strItem := item.(binding.String)
			str, _ := strItem.Get()
			obj.(*fyne.Container).Objects[1].(*widget.Label).SetText(str)
		},
	)

	exclusionEntry := widget.NewEntry()
	exclusionEntry.SetPlaceHolder("*.tmp, *.log, node_modules/")

	addExclusionBtn := widget.NewButtonWithIcon("Ajouter", theme.ContentAddIcon(), func() {
		if exclusionEntry.Text != "" {
			ui.exclusions = append(ui.exclusions, exclusionEntry.Text)
			_ = ui.exclusionsData.Append(exclusionEntry.Text)
			exclusionEntry.SetText("")
		}
	})

	removeExclusionBtn := widget.NewButtonWithIcon("Retirer", theme.DeleteIcon(), func() {
		if len(ui.exclusions) > 0 {
			ui.exclusions = ui.exclusions[:len(ui.exclusions)-1]
			_ = ui.exclusionsData.Set(ui.exclusions)
		}
	})

	// Presets
	presetsLabel := widget.NewLabel("Préréglages :")
	presetButtons := container.NewGridWithColumns(2,
		widget.NewButton("Dev (node_modules, .git)", func() {
			ui.addExclusionPreset([]string{"node_modules/", ".git/", "dist/", "build/", "*.log"})
		}),
		widget.NewButton("Temp files", func() {
			ui.addExclusionPreset([]string{"*.tmp", "*.temp", "~*", ".DS_Store"})
		}),
		widget.NewButton("Caches", func() {
			ui.addExclusionPreset([]string{".cache/", "__pycache__/", "*.pyc"})
		}),
		widget.NewButton("Media", func() {
			ui.addExclusionPreset([]string{"*.mp4", "*.mkv", "*.avi", "*.mov"})
		}),
	)

	inputRow := container.NewBorder(nil, nil, nil, addExclusionBtn, exclusionEntry)

	return container.NewVBox(
		widget.NewLabelWithStyle("Patterns d'exclusion :", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
		inputRow,
		removeExclusionBtn,
		widget.NewSeparator(),
		ui.exclusionsList,
		widget.NewSeparator(),
		presetsLabel,
		presetButtons,
		widget.NewSeparator(),
		widget.NewLabel("💡 Utilisez les wildcards : *, ?, [] pour les patterns"),
	)
}

func (ui *BackupConfigUI) buildScheduleTab() fyne.CanvasObject {
	ui.scheduleEnabled = widget.NewCheck("Activer la planification automatique", nil)

	ui.scheduleSelect = widget.NewSelect(
		[]string{
			"Toutes les heures",
			"Toutes les 6 heures",
			"Quotidien (2h du matin)",
			"Hebdomadaire (Dimanche 2h)",
			"Mensuel (1er jour du mois)",
			"Personnalisé (cron)",
		},
		func(selected string) {
			ui.customCron.Show()
		},
	)
	ui.scheduleSelect.SetSelected("Quotidien (2h du matin)")

	ui.customCron = widget.NewEntry()
	ui.customCron.SetPlaceHolder("0 2 * * * (format cron)")
	ui.customCron.Hide()

	cronHelp := widget.NewHyperlink(
		"Aide sur la syntaxe cron",
		parseURL("https://crontab.guru"),
	)

	return container.NewVBox(
		ui.scheduleEnabled,
		widget.NewSeparator(),
		widget.NewLabel("Fréquence :"),
		ui.scheduleSelect,
		ui.customCron,
		cronHelp,
		widget.NewSeparator(),
		widget.NewLabel("ℹ️ La planification utilise le planificateur système (cron/Task Scheduler)"),
	)
}

func (ui *BackupConfigUI) buildRetentionTab() fyne.CanvasObject {
	ui.keepLast = widget.NewEntry()
	ui.keepLast.SetPlaceHolder("7")
	ui.keepLast.SetText("7")

	ui.keepDaily = widget.NewEntry()
	ui.keepDaily.SetPlaceHolder("14")
	ui.keepDaily.SetText("14")

	ui.keepWeekly = widget.NewEntry()
	ui.keepWeekly.SetPlaceHolder("8")
	ui.keepWeekly.SetText("8")

	ui.keepMonthly = widget.NewEntry()
	ui.keepMonthly.SetPlaceHolder("12")
	ui.keepMonthly.SetText("12")

	form := &widget.Form{
		Items: []*widget.FormItem{
			{Text: "Conserver les derniers :", Widget: ui.keepLast},
			{Text: "Conserver quotidiens :", Widget: ui.keepDaily},
			{Text: "Conserver hebdomadaires :", Widget: ui.keepWeekly},
			{Text: "Conserver mensuels :", Widget: ui.keepMonthly},
		},
	}

	infoLabel := widget.NewLabel(
		"💡 Politique de rétention :\n" +
			"- 'Derniers' : garde les N derniers backups\n" +
			"- 'Quotidiens' : garde un backup par jour pendant N jours\n" +
			"- 'Hebdomadaires' : garde un backup par semaine pendant N semaines\n" +
			"- 'Mensuels' : garde un backup par mois pendant N mois",
	)
	infoLabel.Wrapping = fyne.TextWrapWord

	return container.NewVBox(
		form,
		widget.NewSeparator(),
		infoLabel,
	)
}

func (ui *BackupConfigUI) buildAdvancedTab() fyne.CanvasObject {
	// Compression
	ui.compressionSelect = widget.NewSelect(
		[]string{"zstd (recommandé)", "lz4 (rapide)", "none"},
		nil,
	)
	ui.compressionSelect.SetSelected("zstd (recommandé)")

	// Chunk size
	ui.chunkSizeSelect = widget.NewSelect(
		[]string{"4 MB (recommandé)", "2 MB", "8 MB", "16 MB"},
		nil,
	)
	ui.chunkSizeSelect.SetSelected("4 MB (recommandé)")

	// Bandwidth limit
	ui.bandwidthLimit = widget.NewEntry()
	ui.bandwidthLimit.SetPlaceHolder("100 (MB/s, 0 = illimité)")
	ui.bandwidthLimit.SetText("0")

	// Parallel uploads
	ui.parallelUploads = widget.NewEntry()
	ui.parallelUploads.SetPlaceHolder("2")
	ui.parallelUploads.SetText("2")

	form := &widget.Form{
		Items: []*widget.FormItem{
			{Text: "Compression :", Widget: ui.compressionSelect},
			{Text: "Taille des chunks :", Widget: ui.chunkSizeSelect},
			{Text: "Limite de bande passante (MB/s) :", Widget: ui.bandwidthLimit},
			{Text: "Uploads parallèles :", Widget: ui.parallelUploads},
		},
	}

	return container.NewVBox(
		form,
		widget.NewSeparator(),
		widget.NewLabel("⚠️ Paramètres avancés - Modifier seulement si vous savez ce que vous faites"),
	)
}

func (ui *BackupConfigUI) detectDisks() {
	// TODO: Implement actual disk detection using system calls
	// For now, show mock data
	ui.disks = []DiskInfo{
		{DevicePath: "\\\\.\\PhysicalDisk0", Name: "Disk 0", Size: "500 GB", Model: "Samsung SSD"},
		{DevicePath: "\\\\.\\PhysicalDisk1", Name: "Disk 1", Size: "1 TB", Model: "WD HDD"},
		{DevicePath: "/dev/sda", Name: "sda", Size: "500 GB", Model: "Samsung SSD"},
	}

	diskStrings := []string{}
	for _, disk := range ui.disks {
		diskStrings = append(diskStrings, fmt.Sprintf("%s - %s (%s)", disk.Name, disk.Size, disk.Model))
	}

	_ = ui.disksData.Set(diskStrings)

	dialog.ShowInformation(
		"Disques détectés",
		fmt.Sprintf("%d disque(s) détecté(s)", len(ui.disks)),
		ui.window,
	)
}

func (ui *BackupConfigUI) addExclusionPreset(patterns []string) {
	for _, pattern := range patterns {
		// Check if not already in list
		found := false
		for _, existing := range ui.exclusions {
			if existing == pattern {
				found = true
				break
			}
		}
		if !found {
			ui.exclusions = append(ui.exclusions, pattern)
		}
	}
	_ = ui.exclusionsData.Set(ui.exclusions)
}

func (ui *BackupConfigUI) GetFolders() []string {
	return ui.folders
}

func (ui *BackupConfigUI) GetExclusions() []string {
	return ui.exclusions
}

func (ui *BackupConfigUI) GetSchedule() string {
	if !ui.scheduleEnabled.Checked {
		return ""
	}
	return ui.scheduleSelect.Selected
}
