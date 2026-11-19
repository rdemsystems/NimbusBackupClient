package tui

import (
	"fmt"
	"os"
	"regexp"
	"runtime"
	"strconv"
	"strings"
	"time"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
	"pbscommon"
)

var (
	hhmm      = regexp.MustCompile(`^([01]?\d|2[0-3]):[0-5]\d$`)
	weekdays  = []string{"Mon", "Tue", "Wed", "Thu", "Fri", "Sat", "Sun"}
	unitNames = []string{"minutes", "hours", "days", "weeks", "months"}
)

// editor is the Backup Set form: three steps like the desktop app's wizard.
type editor struct {
	a     *App
	isNew bool
	orig  map[string]interface{}
	set   Set
	pages *tview.Pages
	step  int
	title *tview.TextView

	// step 1
	name, backupID *tview.InputField
	dirs           *tview.TextArea
	server         *tview.DropDown
	serverIDs      []string
	// step 2
	mode                *tview.DropDown
	unit                *tview.DropDown
	every, at, weekdayF *tview.InputField
	wake                *tview.Checkbox // wakeToRun: wake the computer from sleep for its runs (#140)
	// the set's On battery rule: start anyway, only at a charge of at least
	// a level, or wait for mains; and cancel a running backup below a level
	battery                    *tview.DropDown
	batteryStart, batteryBelow *tview.InputField
	batteryCancel              *tview.Checkbox
	// step 3
	detect   *tview.DropDown
	vss      *tview.Checkbox
	exclude  *tview.TextArea
	mailFail *tview.Checkbox
	mailTo   *tview.InputField
	// what the computer does after the backup (the app's Alerts tab): stays
	// as it is, sleeps, hibernates or shuts down, after a wait, and maybe only
	// after a run it was woken for
	after        *tview.DropDown
	afterWait    *tview.InputField
	afterWoken   *tview.Checkbox
	afterStopped *tview.Checkbox // not after a run stopped by hand
	afterInUse   *tview.Checkbox // not when someone used the computer meanwhile
	canHibernate bool
	noHibernate  string // why not, in English, when !canHibernate

	// a machine (disk image) set
	sel        machineSel
	f1         *tview.Form
	freeSpace  *tview.Checkbox
	asVM       *tview.Checkbox
	vmID       *tview.InputField
	vmStyle    *tview.DropDown
	snapsOff   *tview.Checkbox
	liveRead   *tview.Checkbox
	imgFolders *tview.TextArea
}

var vmStyles = []string{"9000000 + number", "000 + number"}

// What the computer does after the backup, as the set stores it ("" stays as
// it is) and as the form lists it.
var (
	afterModes   = []string{"", "sleep", "hibernate", "shutdown"}
	afterChoices = []string{"Stays as it is (its own sleep settings apply)", "Goes to sleep", "Hibernates", "Shuts down"}
)

// afterChoiceOf is a set's choice. shutdownAfter, the older form, decides
// whether it shuts down, as the app and service read it.
func afterChoiceOf(doc map[string]interface{}) string {
	if boolean(doc, "shutdownAfter") {
		return "shutdown"
	}
	switch c := str(doc, "afterBackup"); c {
	case "sleep", "hibernate":
		return c
	}
	return ""
}

// The On battery choices, as the set stores them ("" is start anyway) and as
// the form lists them, in the app's order.
var (
	batteryModes   = []string{"plugged", "above", ""}
	batteryChoices = []string{"Wait for the computer to be plugged in before starting the backup", "Start backup only if battery charge is at least the start level", "Start backup anyway"}
)

// batteryModeIndex is the form's choice for a set's onBattery: a value the
// form does not know starts anyway, as the backup does.
func batteryModeIndex(mode string) int {
	for i, m := range batteryModes {
		if m == mode {
			return i
		}
	}
	return len(batteryModes) - 1
}

// vmBackupID is the numeric PBS vm/<id> a machine set backs up as: 9000000+N
// cannot collide with a real VM's ID, 000N is the other style.
func vmBackupID(n string, style int) string {
	digits := strings.Map(func(r rune) rune {
		if r >= '0' && r <= '9' {
			return r
		}
		return -1
	}, n)
	v, err := strconv.Atoi(digits)
	if err != nil || v < 1 || v > 999999 {
		return ""
	}
	if style == 1 {
		return "000" + strconv.Itoa(v)
	}
	return strconv.Itoa(9000000 + v)
}

// vmFromBackupID is the reverse: the number and style of a saved vm ID.
func vmFromBackupID(id string) (string, int) {
	if v, err := strconv.Atoi(id); err == nil && v > 9000000 && v <= 9999999 {
		return strconv.Itoa(v - 9000000), 0
	}
	if strings.HasPrefix(id, "000") && len(id) > 3 {
		if _, err := strconv.Atoi(id[3:]); err == nil {
			return id[3:], 1
		}
	}
	return "", 0
}

func styleForm(f *tview.Form) *tview.Form {
	f.SetBorder(true).SetBackgroundColor(colBg)
	f.SetFieldBackgroundColor(colSel).SetFieldTextColor(colWhite).SetFieldStyle(onColour(colSel))
	f.SetLabelColor(colDim).SetButtonBackgroundColor(colAccent).SetButtonStyle(onColour(colAccent)).SetButtonTextColor(colWhite)
	return f
}

// editSet opens the form for the selected set; newSet opens it empty.
func (a *App) editSet() {
	s, ok := a.selectedSet()
	if !ok {
		return
	}
	a.openEditor(s, false)
}

// newSet asks what the set backs up, then opens the form.
func (a *App) newSet() {
	m := tview.NewModal().SetText("What should the new Backup Set back up?").
		AddButtons([]string{"Folders", "Disk image", "Cancel"})
	m.SetBackgroundColor(colPanel).SetTextColor(colWhite).SetButtonBackgroundColor(colAccent).SetButtonStyle(onColour(colAccent))
	m.SetDoneFunc(func(_ int, label string) {
		a.pages.RemovePage("newtype")
		a.modal = false
		switch label {
		case "Folders":
			a.openEditor(toSet(a.newSetDoc(false)), true)
		case "Disk image":
			a.openEditor(toSet(a.newSetDoc(true)), true)
		default:
			a.switchTab(a.tab)
		}
	})
	a.modal = true
	a.pages.AddPage("newtype", m, true, true)
	a.app.SetFocus(m)
}

func (a *App) newSetDoc(machine bool) map[string]interface{} {
	raw := map[string]interface{}{
		"id": strconv.FormatInt(time.Now().UnixMilli(), 10), "backupType": "directory", "enabled": true,
		"scheduleMode": "scheduled", "useVSS": true, "changeDetectionMode": "legacy", "excludeList": []interface{}{},
		"schedules": []interface{}{map[string]interface{}{"id": "s1", "enabled": true, "unit": "days", "every": 1.0, "at": "02:00"}},
		// A new set on a laptop's battery waits to be plugged in, and a
		// running backup is cancelled below 10%, as in the app.
		"onBattery": "plugged", "batteryStartPercent": 30.0, "batteryCancel": true, "batteryCancelBelow": 10.0,
	}
	if machine {
		host, _ := os.Hostname()
		raw["backupType"], raw["backupId"] = "machine", host
		raw["changeDetectionMode"] = ""
		raw["driveLetters"], raw["diskIds"], raw["skipPartitions"] = []interface{}{}, []interface{}{}, []interface{}{}
	}
	return raw
}

func (a *App) openEditor(s Set, isNew bool) {
	e := &editor{a: a, isNew: isNew, set: s, orig: copyMap(s.Raw)}
	if e.orig == nil {
		e.orig = map[string]interface{}{}
	}
	e.build()
	a.modal = true
	a.pages.AddPage("edit", e.layout(), true, false)
	a.pages.SwitchToPage("edit") // replaces the screen below, it does not draw over it
	e.show(0)
}

func copyMap(m map[string]interface{}) map[string]interface{} {
	if m == nil {
		return nil
	}
	out := make(map[string]interface{}, len(m))
	for k, v := range m {
		out[k] = v
	}
	return out
}

func indexOf(list []string, v string) int {
	for i, s := range list {
		if s == v {
			return i
		}
	}
	return 0
}

func (e *editor) firstSchedule() map[string]interface{} {
	if arr, ok := e.orig["schedules"].([]interface{}); ok && len(arr) > 0 {
		if m, ok := arr[0].(map[string]interface{}); ok {
			return m
		}
	}
	return map[string]interface{}{"id": "s1", "enabled": true, "unit": "days", "every": 1.0, "at": "02:00"}
}

func (e *editor) build() {
	a := e.a
	e.title = tview.NewTextView().SetDynamicColors(true)
	e.pages = tview.NewPages()

	// ---- step 1: general
	e.name = tview.NewInputField().SetLabel("Name ").SetText(e.set.Name).SetFieldWidth(48)
	e.backupID = tview.NewInputField().SetLabel("Backup ID ").SetText(e.set.BackupID).SetFieldWidth(48)
	e.dirs = tview.NewTextArea().SetLabel("Folders ").SetText(strings.Join(strList(e.orig, "backupDirs"), "\n"), false).SetSize(5, 48)
	e.dirs.SetBorder(false)
	a.mu.Lock()
	options := []string{"(default server)"}
	e.serverIDs = []string{""}
	for _, sv := range a.servers {
		options = append(options, sv.Name)
		e.serverIDs = append(e.serverIDs, sv.ID)
	}
	a.mu.Unlock()
	e.server = tview.NewDropDown().SetLabel("PBS server ").SetOptions(options, nil).SetCurrentOption(indexOf(e.serverIDs, e.set.ServerID))
	f1 := styleForm(tview.NewForm()).AddFormItem(e.name).AddFormItem(e.backupID)
	e.f1 = f1
	if e.set.Type == "machine" {
		e.sel = selOf(e.orig)
		f1.AddTextView("Disks ", e.sel.summary(), 66, 3, true, false)
	} else {
		f1.AddFormItem(e.dirs)
	}
	f1.AddFormItem(e.server)
	f1.SetTitle(" General ")

	// ---- step 2: schedule
	sc := e.firstSchedule()
	e.mode = tview.NewDropDown().SetLabel("Run ").SetOptions([]string{"On a schedule", "Manually only"}, nil)
	if e.set.Manual {
		e.mode.SetCurrentOption(1)
	} else {
		e.mode.SetCurrentOption(0)
	}
	e.unit = tview.NewDropDown().SetLabel("Repeat every ").SetOptions(unitNames, nil).SetCurrentOption(indexOf(unitNames, str(sc, "unit")))
	e.every = tview.NewInputField().SetLabel("Every (number) ").SetText(strconv.Itoa(every(sc))).SetFieldWidth(6)
	e.at = tview.NewInputField().SetLabel("At (HH:MM) ").SetText(str(sc, "at")).SetFieldWidth(8)
	e.weekdayF = tview.NewInputField().SetLabel("Only on days ").SetText(strings.Join(strList(sc, "weekdays"), ",")).SetFieldWidth(32)
	e.wake = tview.NewCheckbox().SetLabel("Wake the computer from sleep or hibernate to run it ").SetChecked(boolean(e.orig, "wakeToRun"))
	// A set saved before the rule existed has none of its fields: start
	// anyway, never cancel.
	e.battery = tview.NewDropDown().SetLabel("On battery ").SetOptions(batteryChoices, nil).
		SetCurrentOption(batteryModeIndex(str(e.orig, "onBattery")))
	start := int(num(e.orig, "batteryStartPercent"))
	if start == 0 {
		start = 30
	}
	below := int(num(e.orig, "batteryCancelBelow"))
	if _, set := e.orig["batteryCancelBelow"]; !set {
		below = 10
	}
	e.batteryStart = tview.NewInputField().SetLabel("  Start level (5 to 95 %) ").SetText(strconv.Itoa(start)).SetFieldWidth(4)
	e.batteryCancel = tview.NewCheckbox().SetLabel("Cancel a running backup on battery below a charge ").SetChecked(boolean(e.orig, "batteryCancel"))
	e.batteryBelow = tview.NewInputField().SetLabel("  Cancel level (0 to 99 %, 0 never) ").SetText(strconv.Itoa(below)).SetFieldWidth(4)
	f2 := styleForm(tview.NewForm()).AddFormItem(e.mode).AddFormItem(e.unit).AddFormItem(e.every).AddFormItem(e.at).AddFormItem(e.weekdayF).AddFormItem(e.wake)
	e.addWakeAlert(f2)
	f2.AddFormItem(e.battery).AddFormItem(e.batteryStart).AddFormItem(e.batteryCancel).AddFormItem(e.batteryBelow)
	f2.AddTextView("", tag(colDim)+"Days: Mon,Tue,Wed,Thu,Fri,Sat,Sun (empty = every day). Hours and minutes need no time of day.\nOnly the first schedule is edited here; others the set has are kept.\nWaking works from sleep (usually hibernate too), not from shut down.\nOn battery: laptops only; on mains the set always runs. A cancelled backup is not saved.", 70, 5, true, false)
	f2.SetTitle(" Schedule ")

	// ---- step 3: options
	e.detect = tview.NewDropDown().SetLabel("Change detection ").SetOptions([]string{"legacy", "data", "metadata"}, nil).
		SetCurrentOption(indexOf([]string{"legacy", "data", "metadata"}, str(e.orig, "changeDetectionMode")))
	e.vss = tview.NewCheckbox().SetLabel("Use VSS snapshot (Windows) ").SetChecked(boolean(e.orig, "useVSS"))
	e.exclude = tview.NewTextArea().SetLabel("Leave out ").SetText(strings.Join(strList(e.orig, "excludeList"), "\n"), false).SetSize(4, 48)
	e.mailFail = tview.NewCheckbox().SetLabel("E-mail when it fails ").SetChecked(boolean(e.orig, "emailOnFailure"))
	e.mailTo = tview.NewInputField().SetLabel("  to ").SetText(str(e.orig, "emailOnFailureTo")).SetFieldWidth(40)
	f3 := styleForm(tview.NewForm())
	if e.set.Type != "machine" {
		f3.AddFormItem(e.detect).AddFormItem(e.exclude)
	} else {
		e.freeSpace = tview.NewCheckbox().SetLabel("Read free space too (a forensic copy) ").SetChecked(boolean(e.orig, "readFreeSpace"))
		vmNum, vmSty := "", 0
		isVM := boolean(e.orig, "machineAsVm")
		if isVM {
			vmNum, vmSty = vmFromBackupID(e.set.BackupID)
		}
		e.asVM = tview.NewCheckbox().SetLabel("Back up as a Proxmox VM (vm/ID, restorable from Proxmox VE) ").SetChecked(isVM)
		e.vmID = tview.NewInputField().SetLabel("  VM number (1 to 999999) ").SetText(vmNum).SetFieldWidth(8)
		e.vmStyle = tview.NewDropDown().SetLabel("  Backup ID in PBS ").SetOptions(vmStyles, nil).SetCurrentOption(vmSty)
		e.imgFolders = tview.NewTextArea().SetLabel("Undelete and Roll back folders ").
			SetText(strings.Join(strList(e.orig, "imageFolders"), "\n"), false).SetSize(3, 48)
		e.imgFolders.SetBorder(false)
		f3.AddFormItem(e.freeSpace).AddFormItem(e.asVM).AddFormItem(e.vmID).AddFormItem(e.vmStyle)
		if runtime.GOOS == "linux" {
			e.snapsOff = tview.NewCheckbox().SetLabel("Do not use snapshots (read the disks live) ").SetChecked(boolean(e.orig, "snapshotsOff"))
			e.liveRead = tview.NewCheckbox().SetLabel("  I accept reading mounted disks live ").SetChecked(boolean(e.orig, "liveRead"))
			f3.AddFormItem(e.snapsOff).AddFormItem(e.liveRead)
		}
		f3.AddFormItem(e.imgFolders)
	}
	f3.AddFormItem(e.vss).AddFormItem(e.mailFail).AddFormItem(e.mailTo)
	e.buildAfterBackup(f3)
	f3.SetTitle(" Options ")

	forms := []*tview.Form{f1, f2, f3}
	for i, f := range forms {
		i := i
		if i == 0 && e.set.Type == "machine" {
			f.AddButton("Choose disks", e.chooseDisks)
		}
		if i > 0 {
			f.AddButton("Back", func() { e.show(i - 1) })
		}
		if i < len(forms)-1 {
			f.AddButton("Next", func() { e.show(i + 1) })
		}
		f.AddButton("Save", e.save)
		f.AddButton("Cancel", e.close)
		f.SetInputCapture(func(ev *tcell.EventKey) *tcell.EventKey {
			if d := altStep(ev); d != 0 {
				if j := i + d; j >= 0 && j < len(forms) {
					e.show(j)
				}
				return nil
			}
			switch ev.Key() {
			case tcell.KeyEscape:
				e.close()
				return nil
			case tcell.KeyPgDn:
				if i < len(forms)-1 {
					e.show(i + 1)
				}
				return nil
			case tcell.KeyPgUp:
				if i > 0 {
					e.show(i - 1)
				}
				return nil
			case tcell.KeyCtrlS:
				e.save()
				return nil
			case tcell.KeyCtrlD:
				if e.set.Type == "machine" {
					e.chooseDisks()
					return nil
				}
			}
			return ev
		})
		e.pages.AddPage(fmt.Sprint(i), f, true, i == 0)
	}
}

// addWakeAlert puts under Wake the alert the app shows there when the power
// plan stops a timer waking the computer (read from the service; an older
// one has none), while Wake is ticked, and a button that turns wake timers
// on. The app's "Leave" button has no counterpart here: not pressing the
// button leaves the plan as it is.
func (e *editor) addWakeAlert(f *tview.Form) {
	info, err := e.a.be.WakeTimers()
	if err != nil {
		return
	}
	text, button := wakeAlertText(info.Alert)
	// On Modern Standby the app's best effort statement goes under it (#142).
	rest, rows := "", 5
	if info.ModernStandby {
		rest, rows = modernStandbyText, 11
		if text != "" {
			rows = 16
		}
	}
	if text == "" && rest == "" {
		return
	}
	// As wide as the form allows (0), and tall enough for the text wrapped
	// beside the form's longest label.
	note := tview.NewTextView().SetLabel("").SetSize(rows, 0).SetDynamicColors(true).SetScrollable(false)
	full := strings.TrimSpace(text + "\n\n" + rest)
	show := func(on bool) {
		if on {
			note.SetText(tag(colWarn) + full)
		} else {
			note.SetText("")
		}
	}
	show(e.wake.IsChecked())
	e.wake.SetChangedFunc(show)
	f.AddFormItem(note)
	if button != "" {
		f.AddButton(button, func() { e.enableWake(f, note, info.Alert, button, rest) })
	}
}

// enableWake has the service turn wake timers on, the user having pressed
// the alert's button: scope as the alert, "battery" or "all". rest is the
// text that stays under Wake afterwards (the Modern Standby statement).
func (e *editor) enableWake(f *tview.Form, note *tview.TextView, scope, button, rest string) {
	e.a.setFlash("turning on wake timers...")
	go func() {
		_, err := e.a.be.EnableWakeTimers(scope)
		e.a.app.QueueUpdateDraw(func() {
			if err != nil {
				e.a.setFlash("wake timers were not turned on")
				e.a.errorBox("The power plan was not changed: " + describeErr(err))
				return
			}
			if rest == "" {
				e.wake.SetChangedFunc(nil)
				note.SetText("")
			} else {
				e.wake.SetChangedFunc(func(on bool) {
					if on {
						note.SetText(tag(colWarn) + rest)
					} else {
						note.SetText("")
					}
				})
				note.SetText(tag(colWarn) + rest)
			}
			if i := f.GetButtonIndex(button); i >= 0 {
				f.RemoveButton(i)
			}
			f.SetFocus(f.GetFormItemIndex(e.wake.GetLabel()))
			e.a.app.SetFocus(f)
			e.a.setFlash("wake timers turned on in the power plan")
		})
	}()
}

// buildAfterBackup adds to f what the computer does after the backup. This
// computer's hibernation is read here: the form runs on the computer whose
// service runs the set.
func (e *editor) buildAfterBackup(f *tview.Form) {
	choice := afterChoiceOf(e.orig)
	hib := pbscommon.CanHibernate()
	e.canHibernate, e.noHibernate = hib.OK, hib.Reason()
	choices := append([]string{}, afterChoices...)
	if !e.canHibernate {
		switch hib.Why {
		case pbscommon.HibernateOff:
			choices[2] += " (turned off on this computer)"
		case pbscommon.HibernateNone:
			choices[2] += " (not on this computer)"
		default:
			choices[2] += " (unavailable: " + hib.Reason() + ")"
		}
	}
	wait := int(num(e.orig, "afterBackupWait"))
	if choice == "" {
		// A minute to begin with, once one is chosen: time to cancel.
		wait = 1
	}
	// Ticked to begin with: it counts only while the set wakes the computer.
	woken := choice == "" || boolean(e.orig, "afterBackupOnlyIfWoken")
	e.after = tview.NewDropDown().SetLabel("When the backup finishes, the computer ").SetOptions(choices, nil).SetCurrentOption(indexOf(afterModes, choice))
	e.afterWait = tview.NewInputField().SetLabel("  After (0 to 240 minutes, 0 at once) ").SetText(strconv.Itoa(wait)).SetFieldWidth(4)
	e.afterWoken = tview.NewCheckbox().SetLabel("  Only if this backup woke the computer (needs Wake, on Schedule) ").SetChecked(woken)
	// Ticked unless the set turned them off: they keep the computer as it is
	// for someone at it.
	e.afterStopped = tview.NewCheckbox().SetLabel("  Not if the backup was stopped by hand ").SetChecked(!boolean(e.orig, "afterBackupEvenIfStopped"))
	e.afterInUse = tview.NewCheckbox().SetLabel("  Not if someone used the computer since it woke or the backup started ").SetChecked(!boolean(e.orig, "afterBackupEvenIfInUse"))
	f.AddFormItem(e.after).AddFormItem(e.afterWait).AddFormItem(e.afterWoken).AddFormItem(e.afterStopped).AddFormItem(e.afterInUse)
	f.AddTextView("", tag(colDim)+"It waits for any other backup still running; the app shows a notice with Cancel.\nShutting down closes every program, with any unsaved work, and a computer\nthat is shut down cannot be woken for the next backup.", 78, 3, true, false)
}

// applyAfterBackup puts what the computer does after the backup in doc. Call
// it after the schedule: "only if woken" counts only for a set that wakes the
// computer. shutdownAfter is written too, for an older service.
func (e *editor) applyAfterBackup(doc map[string]interface{}) error {
	ai, _ := e.after.GetCurrentOption()
	if ai < 0 || ai >= len(afterModes) {
		ai = 0
	}
	choice := afterModes[ai]
	wait, err := strconv.Atoi(strings.TrimSpace(e.afterWait.GetText()))
	if err != nil || wait < 0 || wait > 240 {
		return fmt.Errorf("the wait after the backup must be a whole number of minutes from 0 to 240 (0 at once)")
	}
	if choice == "hibernate" && !e.canHibernate && afterChoiceOf(e.orig) != "hibernate" {
		return fmt.Errorf("this computer cannot hibernate (%s): choose another thing for it to do after the backup", e.noHibernate)
	}
	if choice == "" {
		wait = 0
	}
	wakes, _ := doc["wakeToRun"].(bool)
	doc["afterBackup"] = choice
	doc["afterBackupWait"] = float64(wait)
	doc["afterBackupOnlyIfWoken"] = choice != "" && wakes && e.afterWoken.IsChecked()
	doc["afterBackupEvenIfStopped"] = choice != "" && !e.afterStopped.IsChecked()
	doc["afterBackupEvenIfInUse"] = choice != "" && !e.afterInUse.IsChecked()
	doc["shutdownAfter"] = choice == "shutdown"
	return nil
}

// chooseDisks opens the disk picker on top of the form.
func (e *editor) chooseDisks() {
	back := func() {
		e.a.pages.SwitchToPage("edit")
		e.show(e.step)
	}
	e.a.pickDisks(e.sel, true, func(s machineSel) {
		e.sel = s
		if tv, ok := e.f1.GetFormItemByLabel("Disks ").(*tview.TextView); ok {
			tv.SetText(s.summary())
		}
		back()
	}, back)
}

func (e *editor) layout() tview.Primitive {
	body := tview.NewFlex().SetDirection(tview.FlexRow).
		AddItem(e.title, 1, 0, false).
		AddItem(e.pages, 0, 1, true)
	return tview.NewFlex().AddItem(nil, 0, 1, false).AddItem(body, 90, 0, true).AddItem(nil, 0, 1, false)
}

func (e *editor) show(step int) {
	e.step = step
	e.pages.SwitchToPage(fmt.Sprint(step))
	b := strings.Builder{}
	b.WriteString(stepBar([]string{"General", "Schedule", "Options"}, step, false))
	word := "Edit Backup Set"
	if e.isNew {
		word = "New Backup Set"
	}
	b.WriteString(tag(colDim) + "   " + word + "   PgDn next, PgUp back, Ctrl+S save, Esc cancel")
	if e.set.Type == "machine" {
		b.WriteString(", Ctrl+D disks")
	}
	e.title.SetText(b.String())
	_, front := e.pages.GetFrontPage()
	e.a.app.SetFocus(front)
}

func (e *editor) close() {
	e.a.pages.RemovePage("edit")
	e.a.modal = false
	e.a.switchTab(e.a.tab)
}

func splitLines(s string) []interface{} {
	out := []interface{}{}
	for _, l := range strings.Split(strings.ReplaceAll(s, "\r", ""), "\n") {
		if l = strings.TrimSpace(l); l != "" {
			out = append(out, l)
		}
	}
	return out
}

func parseWeekdays(s string) ([]interface{}, error) {
	out := []interface{}{}
	for _, p := range strings.FieldsFunc(s, func(r rune) bool { return r == ',' || r == ' ' || r == ';' }) {
		found := ""
		for _, d := range weekdays {
			if strings.EqualFold(d, p) || strings.EqualFold(d, p[:min(3, len(p))]) {
				found = d
			}
		}
		if found == "" {
			return nil, fmt.Errorf("%q is not a day (use Mon, Tue, Wed, Thu, Fri, Sat, Sun)", p)
		}
		out = append(out, found)
	}
	return out, nil
}

// document builds the Backup Set document to send: the original with the
// form's fields applied, so every field the form does not show is kept.
func (e *editor) document() (map[string]interface{}, error) {
	doc := copyMap(e.orig)
	name := strings.TrimSpace(e.name.GetText())
	if name == "" {
		return nil, fmt.Errorf("the Backup Set needs a name")
	}
	doc["name"] = name
	doc["backupId"] = strings.TrimSpace(e.backupID.GetText())
	if e.set.Type != "machine" {
		dirs := splitLines(e.dirs.GetText())
		if len(dirs) == 0 {
			return nil, fmt.Errorf("add at least one folder to back up (one per line)")
		}
		doc["backupDirs"] = dirs
		doc["backupType"] = "directory"
		_, det := e.detect.GetCurrentOption()
		doc["changeDetectionMode"] = det
		doc["excludeList"] = splitLines(e.exclude.GetText())
	} else {
		if len(e.sel.paths) == 0 {
			return nil, fmt.Errorf("choose at least one disk (Choose disks on the General page)")
		}
		doc["backupType"] = "machine"
		doc["backupDirs"] = []interface{}{}
		e.sel.apply(doc)
		doc["readFreeSpace"] = e.freeSpace.IsChecked()
		doc["machineAsVm"] = e.asVM.IsChecked()
		if e.asVM.IsChecked() {
			si, _ := e.vmStyle.GetCurrentOption()
			id := vmBackupID(e.vmID.GetText(), si)
			if id == "" {
				return nil, fmt.Errorf("enter a VM number between 1 and 999999")
			}
			doc["backupId"] = id
		}
		if e.snapsOff != nil {
			doc["snapshotsOff"] = e.snapsOff.IsChecked()
			doc["liveRead"] = e.snapsOff.IsChecked() && e.liveRead.IsChecked()
			if e.snapsOff.IsChecked() && !e.liveRead.IsChecked() {
				return nil, fmt.Errorf("snapshots are off but reading the disks live is not accepted, so every run would stop; accept it or use snapshots")
			}
		}
		doc["imageFolders"] = splitLines(e.imgFolders.GetText())
	}
	idx, _ := e.server.GetCurrentOption()
	doc["pbsServerId"] = e.serverIDs[idx]
	doc["useVSS"] = e.vss.IsChecked()
	doc["emailOnFailure"] = e.mailFail.IsChecked()
	doc["emailOnFailureTo"] = strings.TrimSpace(e.mailTo.GetText())
	if e.mailFail.IsChecked() && !strings.Contains(e.mailTo.GetText(), "@") {
		return nil, fmt.Errorf("give an e-mail address to send the failure message to")
	}
	if err := e.applyBattery(doc); err != nil {
		return nil, err
	}

	// schedule
	if mi, _ := e.mode.GetCurrentOption(); mi == 1 {
		doc["scheduleMode"] = "manual"
		doc["wakeToRun"] = false
	} else {
		doc["scheduleMode"] = "scheduled"
		doc["wakeToRun"] = e.wake.IsChecked()
		sc := copyMap(e.firstSchedule())
		_, unit := e.unit.GetCurrentOption()
		sc["unit"] = unit
		n, err := strconv.Atoi(strings.TrimSpace(e.every.GetText()))
		if err != nil || n < 1 {
			return nil, fmt.Errorf("\"Every\" must be a whole number, 1 or more")
		}
		sc["every"] = float64(n)
		at := strings.TrimSpace(e.at.GetText())
		if unit == "days" || unit == "weeks" || unit == "months" {
			if !hhmm.MatchString(at) {
				return nil, fmt.Errorf("the time of day must look like 02:00")
			}
			sc["at"] = at
		}
		days, err := parseWeekdays(e.weekdayF.GetText())
		if err != nil {
			return nil, err
		}
		if len(days) > 0 {
			sc["weekdays"] = days
		} else {
			delete(sc, "weekdays")
		}
		sc["enabled"] = true
		var list []interface{}
		if arr, ok := e.orig["schedules"].([]interface{}); ok && len(arr) > 0 {
			list = append([]interface{}{sc}, arr[1:]...)
		} else {
			list = []interface{}{sc}
		}
		doc["schedules"] = list
	}
	if err := e.applyAfterBackup(doc); err != nil {
		return nil, err
	}
	return doc, nil
}

// applyBattery puts the form's On battery rule in doc. It applies to a set
// run manually too (its cancel level), so it is kept whatever the mode.
func (e *editor) applyBattery(doc map[string]interface{}) error {
	bi, _ := e.battery.GetCurrentOption()
	if bi < 0 || bi >= len(batteryModes) {
		bi = batteryModeIndex("")
	}
	start, err := strconv.Atoi(strings.TrimSpace(e.batteryStart.GetText()))
	if err != nil || start < 5 || start > 95 {
		return fmt.Errorf("the battery start level must be a whole number from 5 to 95")
	}
	below, err := strconv.Atoi(strings.TrimSpace(e.batteryBelow.GetText()))
	if err != nil || below < 0 || below > 99 {
		return fmt.Errorf("the battery cancel level must be a whole number from 0 to 99 (0 never cancels)")
	}
	if batteryModes[bi] == "above" && e.batteryCancel.IsChecked() && below > 0 && below >= start {
		return fmt.Errorf("the battery cancel level (%d%%) must be lower than the start level (%d%%)", below, start)
	}
	doc["onBattery"] = batteryModes[bi]
	doc["batteryStartPercent"] = float64(start)
	doc["batteryCancel"] = e.batteryCancel.IsChecked()
	doc["batteryCancelBelow"] = float64(below)
	return nil
}

func (e *editor) save() {
	doc, err := e.document()
	if err != nil {
		e.a.errorBox(err.Error())
		return
	}
	go func() {
		err := e.a.be.SaveSet(doc, e.isNew)
		e.a.app.QueueUpdateDraw(func() {
			if err != nil {
				e.a.errorBox(describeErr(err))
				return
			}
			e.close()
			e.a.setFlash("saved " + str(doc, "name"))
			go e.a.refresh(true)
		})
	}()
}

// errorBox shows a message above the window; OK closes it.
func (a *App) errorBox(text string) {
	m := tview.NewModal().SetText(text).AddButtons([]string{"OK"})
	m.SetBackgroundColor(colPanel).SetTextColor(colErr).SetButtonBackgroundColor(colAccent).SetButtonStyle(onColour(colAccent))
	prev := a.modal
	m.SetDoneFunc(func(int, string) {
		a.pages.RemovePage("error")
		a.modal = prev
		if prev {
			if _, front := a.pages.GetFrontPage(); front != nil {
				a.app.SetFocus(front)
			}
		}
	})
	a.modal = true
	a.pages.AddPage("error", m, true, true)
	a.app.SetFocus(m)
}

// actDelete removes the selected set after asking.
func (a *App) actDelete() {
	s, ok := a.selectedSet()
	if !ok {
		return
	}
	a.confirm(fmt.Sprintf("Delete the Backup Set %q?\nIts backups on the server stay.", s.Name), func() {
		go func() {
			err := a.be.DeleteSet(s.ID)
			msg := "deleted " + s.Name
			if err != nil {
				msg = describeErr(err)
			}
			a.app.QueueUpdateDraw(func() { a.setFlash(msg) })
			a.refresh(true)
		}()
	})
}
