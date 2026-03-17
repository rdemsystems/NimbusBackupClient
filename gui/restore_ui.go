package main

import (
	"fmt"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/data/binding"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
)

// RestoreUI provides the restore/browse interface
type RestoreUI struct {
	window         fyne.Window
	restoreManager *RestoreManager

	snapshotsList *widget.List
	snapshotsData binding.StringList
	snapshots     []BackupSnapshot

	selectedSnapshot *BackupSnapshot
	filesList        *widget.List
	filesData        binding.StringList
}

func NewRestoreUI(window fyne.Window, config *Config) *RestoreUI {
	ui := &RestoreUI{
		window:         window,
		restoreManager: NewRestoreManager(config),
		snapshotsData:  binding.NewStringList(),
		filesData:      binding.NewStringList(),
		snapshots:      []BackupSnapshot{},
	}

	return ui
}

func (ui *RestoreUI) BuildRestoreTab() fyne.CanvasObject {
	// Snapshots list
	ui.snapshotsList = widget.NewListWithData(
		ui.snapshotsData,
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

	ui.snapshotsList.OnSelected = func(id widget.ListItemID) {
		if id < len(ui.snapshots) {
			ui.selectedSnapshot = &ui.snapshots[id]
			ui.loadSnapshotFiles()
		}
	}

	// Files list
	ui.filesList = widget.NewListWithData(
		ui.filesData,
		func() fyne.CanvasObject {
			return container.NewHBox(
				widget.NewIcon(theme.FileIcon()),
				widget.NewLabel("template"),
			)
		},
		func(item binding.DataItem, obj fyne.CanvasObject) {
			strItem := item.(binding.String)
			str, _ := strItem.Get()
			obj.(*fyne.Container).Objects[1].(*widget.Label).SetText(str)
		},
	)

	// Buttons
	refreshBtn := widget.NewButtonWithIcon("Rafraîchir", theme.ViewRefreshIcon(), func() {
		ui.refreshSnapshots()
	})

	browseBtn := widget.NewButtonWithIcon("Parcourir (Terminal)", theme.FolderOpenIcon(), func() {
		ui.browseInteractive()
	})

	restoreBtn := widget.NewButtonWithIcon("Restaurer", theme.DownloadIcon(), func() {
		ui.showRestoreDialog()
	})
	restoreBtn.Importance = widget.HighImportance

	infoBtn := widget.NewButtonWithIcon("Infos", theme.InfoIcon(), func() {
		ui.showSnapshotInfo()
	})

	buttons := container.NewHBox(refreshBtn, browseBtn, infoBtn, restoreBtn)

	// Layout
	snapshotsSection := container.NewBorder(
		widget.NewLabelWithStyle("Snapshots disponibles :", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
		nil, nil, nil,
		ui.snapshotsList,
	)

	filesSection := container.NewBorder(
		widget.NewLabelWithStyle("Fichiers dans le snapshot :", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
		nil, nil, nil,
		ui.filesList,
	)

	split := container.NewHSplit(snapshotsSection, filesSection)
	split.SetOffset(0.4)

	infoLabel := widget.NewLabel(
		"💡 Sélectionnez un snapshot pour voir son contenu.\n" +
			"Utilisez 'Parcourir' pour une exploration interactive via NBD.\n" +
			"Restaurez des fichiers individuels ou montez le disque complet.",
	)
	infoLabel.Wrapping = fyne.TextWrapWord

	return container.NewBorder(
		container.NewVBox(infoLabel, widget.NewSeparator(), buttons, widget.NewSeparator()),
		nil, nil, nil,
		split,
	)
}

func (ui *RestoreUI) refreshSnapshots() {
	snapshots, err := ui.restoreManager.ListSnapshots()
	if err != nil {
		dialog.ShowError(err, ui.window)
		return
	}

	ui.snapshots = snapshots
	snapshotStrings := []string{}

	for _, snapshot := range snapshots {
		age := formatAge(snapshot.Timestamp)
		snapshotStr := fmt.Sprintf("%s/%s - %s (%s)",
			snapshot.Type,
			snapshot.ID,
			snapshot.Timestamp.Format("2006-01-02 15:04"),
			age,
		)
		snapshotStrings = append(snapshotStrings, snapshotStr)
	}

	_ = ui.snapshotsData.Set(snapshotStrings)

	dialog.ShowInformation(
		"Rafraîchi",
		fmt.Sprintf("%d snapshot(s) trouvé(s)", len(snapshots)),
		ui.window,
	)
}

func (ui *RestoreUI) loadSnapshotFiles() {
	if ui.selectedSnapshot == nil {
		return
	}

	fileStrings := []string{}
	for _, file := range ui.selectedSnapshot.Files {
		fileStr := fmt.Sprintf("%s (%s) - %s",
			file.Name,
			file.Type,
			formatBytes(file.Size),
		)
		fileStrings = append(fileStrings, fileStr)
	}

	_ = ui.filesData.Set(fileStrings)
}

func (ui *RestoreUI) browseInteractive() {
	dialog.ShowInformation(
		"Mode terminal",
		"L'outil de parcours s'ouvrira dans un terminal séparé.\n\n"+
			"Utilisez les flèches pour naviguer et Entrée pour sélectionner.\n"+
			"Le device NBD sera monté sur /dev/nbd0.\n\n"+
			"IMPORTANT: Démontez proprement avec 'nbd-client -d /dev/nbd0' avant de quitter.",
		ui.window,
	)

	go func() {
		if err := ui.restoreManager.BrowseBackup(); err != nil {
			dialog.ShowError(fmt.Errorf("échec parcours: %v", err), ui.window)
		}
	}()
}

func (ui *RestoreUI) showRestoreDialog() {
	if ui.selectedSnapshot == nil {
		dialog.ShowInformation("Aucune sélection", "Sélectionnez d'abord un snapshot", ui.window)
		return
	}

	// File selection
	fileSelect := widget.NewSelect([]string{}, nil)
	fileNames := []string{}
	for _, file := range ui.selectedSnapshot.Files {
		fileNames = append(fileNames, file.Name)
	}
	fileSelect.Options = fileNames
	if len(fileNames) > 0 {
		fileSelect.SetSelected(fileNames[0])
	}

	// Target path
	targetEntry := widget.NewEntry()
	targetEntry.SetPlaceHolder("/restore/target/path")

	browseBtn := widget.NewButton("Parcourir", func() {
		dialog.ShowFolderOpen(func(dir fyne.ListableURI, err error) {
			if err == nil && dir != nil {
				targetEntry.SetText(dir.Path())
			}
		}, ui.window)
	})

	form := &widget.Form{
		Items: []*widget.FormItem{
			{Text: "Fichier à restaurer:", Widget: fileSelect},
			{Text: "Destination:", Widget: container.NewBorder(nil, nil, nil, browseBtn, targetEntry)},
		},
		OnSubmit: func() {
			// Find selected file
			var selectedFile *BackupFile
			for i, file := range ui.selectedSnapshot.Files {
				if file.Name == fileSelect.Selected {
					selectedFile = &ui.selectedSnapshot.Files[i]
					break
				}
			}

			if selectedFile == nil {
				dialog.ShowError(fmt.Errorf("fichier non trouvé"), ui.window)
				return
			}

			// Restore
			err := ui.restoreManager.RestoreFile(*ui.selectedSnapshot, *selectedFile, targetEntry.Text)
			if err != nil {
				dialog.ShowError(err, ui.window)
			} else {
				dialog.ShowInformation("Succès", "Restauration lancée avec succès!", ui.window)
			}
		},
	}

	dialog.ShowForm("Restaurer depuis backup", "Restaurer", "Annuler", form.Items, func(confirmed bool) {
		if confirmed {
			form.OnSubmit()
		}
	}, ui.window)
}

func (ui *RestoreUI) showSnapshotInfo() {
	if ui.selectedSnapshot == nil {
		dialog.ShowInformation("Aucune sélection", "Sélectionnez d'abord un snapshot", ui.window)
		return
	}

	info, err := ui.restoreManager.GetSnapshotInfo(*ui.selectedSnapshot)
	if err != nil {
		dialog.ShowError(err, ui.window)
		return
	}

	dialog.ShowInformation("Informations du snapshot", info, ui.window)
}

func formatAge(timestamp time.Time) string {
	duration := time.Since(timestamp)

	if duration < time.Hour {
		return fmt.Sprintf("%d min", int(duration.Minutes()))
	}
	if duration < 24*time.Hour {
		return fmt.Sprintf("%d h", int(duration.Hours()))
	}
	if duration < 7*24*time.Hour {
		return fmt.Sprintf("%d j", int(duration.Hours()/24))
	}
	if duration < 30*24*time.Hour {
		return fmt.Sprintf("%d sem", int(duration.Hours()/(24*7)))
	}
	return fmt.Sprintf("%d mois", int(duration.Hours()/(24*30)))
}
