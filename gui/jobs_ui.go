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

// JobsUI handles the jobs management interface
type JobsUI struct {
	window     fyne.Window
	jobManager *JobManager
	scheduler  *Scheduler

	jobsList *widget.List
	jobsData binding.StringList
}

func NewJobsUI(window fyne.Window, jm *JobManager) *JobsUI {
	ui := &JobsUI{
		window:     window,
		jobManager: jm,
		scheduler:  NewScheduler(jm),
		jobsData:   binding.NewStringList(),
	}

	ui.refreshJobsList()

	return ui
}

func (ui *JobsUI) BuildJobsTab() fyne.CanvasObject {
	// Jobs list
	ui.jobsList = widget.NewListWithData(
		ui.jobsData,
		func() fyne.CanvasObject {
			return container.NewHBox(
				widget.NewCheck("", nil),
				widget.NewLabel("template"),
				widget.NewButton("Éditer", nil),
				widget.NewButton("Lancer", nil),
				widget.NewButton("Supprimer", nil),
			)
		},
		func(item binding.DataItem, obj fyne.CanvasObject) {
			strItem := item.(binding.String)
			str, _ := strItem.Get()

			hbox := obj.(*fyne.Container)
			hbox.Objects[1].(*widget.Label).SetText(str)
		},
	)

	addJobBtn := widget.NewButtonWithIcon("Nouveau Job", theme.ContentAddIcon(), func() {
		ui.showNewJobDialog()
	})

	refreshBtn := widget.NewButtonWithIcon("Rafraîchir", theme.ViewRefreshIcon(), func() {
		ui.refreshJobsList()
	})

	exportBtn := widget.NewButtonWithIcon("Exporter", theme.DocumentSaveIcon(), func() {
		ui.showExportDialog()
	})

	buttons := container.NewHBox(addJobBtn, refreshBtn, exportBtn)

	statsLabel := widget.NewLabel(fmt.Sprintf(
		"📊 %d job(s) total, %d activé(s)",
		len(ui.jobManager.Jobs),
		len(ui.jobManager.GetEnabledJobs()),
	))

	return container.NewBorder(
		container.NewVBox(statsLabel, widget.NewSeparator(), buttons),
		nil, nil, nil,
		ui.jobsList,
	)
}

func (ui *JobsUI) refreshJobsList() {
	jobStrings := []string{}
	for _, job := range ui.jobManager.Jobs {
		status := "❌"
		if job.Enabled {
			status = "✅"
		}

		lastRun := "Jamais"
		if !job.LastRun.IsZero() {
			lastRun = job.LastRun.Format("02/01 15:04")
		}

		jobStr := fmt.Sprintf("%s %s | %s | Dernière exec: %s",
			status, job.Name, job.Schedule, lastRun)
		jobStrings = append(jobStrings, jobStr)
	}

	_ = ui.jobsData.Set(jobStrings)
}

func (ui *JobsUI) showNewJobDialog() {
	// Name
	nameEntry := widget.NewEntry()
	nameEntry.SetPlaceHolder("Backup Web Server")

	// Description
	descEntry := widget.NewEntry()
	descEntry.SetPlaceHolder("Sauvegarde quotidienne du serveur web")

	// Schedule
	scheduleSelect := widget.NewSelect(
		[]string{
			"Toutes les heures",
			"Toutes les 6 heures",
			"Quotidien (2h du matin)",
			"Hebdomadaire (Dimanche 2h)",
			"Mensuel (1er jour du mois)",
		},
		nil,
	)
	scheduleSelect.SetSelected("Quotidien (2h du matin)")

	// Folder
	folderEntry := widget.NewEntry()
	folderEntry.SetPlaceHolder("/var/www/html")

	browseBtn := widget.NewButton("Parcourir", func() {
		dialog.ShowFolderOpen(func(dir fyne.ListableURI, err error) {
			if err == nil && dir != nil {
				folderEntry.SetText(dir.Path())
			}
		}, ui.window)
	})

	form := &widget.Form{
		Items: []*widget.FormItem{
			{Text: "Nom du job:", Widget: nameEntry},
			{Text: "Description:", Widget: descEntry},
			{Text: "Planification:", Widget: scheduleSelect},
			{Text: "Dossier:", Widget: container.NewBorder(nil, nil, nil, browseBtn, folderEntry)},
		},
		OnSubmit: func() {
			// Create new job
			job := &Job{
				Name:         nameEntry.Text,
				Description:  descEntry.Text,
				Schedule:     scheduleSelect.Selected,
				ScheduleCron: GetCronExpression(scheduleSelect.Selected),
				Folders:      []string{folderEntry.Text},
				Enabled:      true,
				KeepLast:     7,
				KeepDaily:    14,
				KeepWeekly:   8,
				KeepMonthly:  12,
				Compression:  "zstd",
				ChunkSize:    "4M",
			}

			if err := ui.jobManager.AddJob(job); err != nil {
				dialog.ShowError(err, ui.window)
				return
			}

			// Schedule the job
			if err := ui.scheduler.ScheduleJob(job); err != nil {
				dialog.ShowError(fmt.Errorf("job créé mais échec planification: %v", err), ui.window)
			} else {
				dialog.ShowInformation("Succès", "Job créé et planifié avec succès!", ui.window)
			}

			ui.refreshJobsList()
		},
		OnCancel: func() {
			// Dialog will close automatically
		},
	}

	dialogWindow := dialog.NewForm("Nouveau Job", "Créer", "Annuler", form.Items, func(confirmed bool) {
		if confirmed {
			form.OnSubmit()
		}
	}, ui.window)

	dialogWindow.Resize(fyne.NewSize(500, 400))
	dialogWindow.Show()
}

func (ui *JobsUI) showExportDialog() {
	if len(ui.jobManager.Jobs) == 0 {
		dialog.ShowInformation("Aucun job", "Créez d'abord un job à exporter", ui.window)
		return
	}

	// Select job to export
	jobNames := []string{}
	for _, job := range ui.jobManager.Jobs {
		jobNames = append(jobNames, job.Name)
	}

	jobSelect := widget.NewSelect(jobNames, nil)
	jobSelect.SetSelected(jobNames[0])

	formatSelect := widget.NewRadioGroup([]string{"JSON", "INI"}, nil)
	formatSelect.SetSelected("JSON")

	form := &widget.Form{
		Items: []*widget.FormItem{
			{Text: "Job à exporter:", Widget: jobSelect},
			{Text: "Format:", Widget: formatSelect},
		},
		OnSubmit: func() {
			// Find selected job
			var selectedJob *Job
			for _, job := range ui.jobManager.Jobs {
				if job.Name == jobSelect.Selected {
					selectedJob = job
					break
				}
			}

			if selectedJob == nil {
				return
			}

			// Export
			dialog.ShowFileSave(func(writer fyne.URIWriteCloser, err error) {
				if err != nil || writer == nil {
					return
				}
				defer writer.Close()

				var exportErr error
				if formatSelect.Selected == "JSON" {
					exportErr = selectedJob.ExportToJSON(writer.URI().Path())
				} else {
					exportErr = selectedJob.ExportToINI(writer.URI().Path())
				}

				if exportErr != nil {
					dialog.ShowError(exportErr, ui.window)
				} else {
					dialog.ShowInformation("Succès", "Job exporté avec succès!", ui.window)
				}
			}, ui.window)
		},
	}

	dialog.ShowForm("Exporter Job", "Exporter", "Annuler", form.Items, func(confirmed bool) {
		if confirmed {
			form.OnSubmit()
		}
	}, ui.window)
}
