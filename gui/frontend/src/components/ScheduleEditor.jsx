import { useState } from 'react'
import { useTranslation } from '../i18n/i18nContext'
import {
  WEEK, UNITS, MIN_REPEAT_MINUTES, newSchedule, nextScheduleId, describe, nextRuns,
  nextSetRun, scheduleProblem, weekdayName, monthName, tp, nthWord,
} from './schedules'

// Where the app keeps which wake alert the user chose to leave as it is.
const WAKE_ALERT_LEFT = 'wakeAlertLeft'
const readWakeAlertLeft = () => {
  try { return localStorage.getItem(WAKE_ALERT_LEFT) || '' } catch { return '' }
}

// The Schedule tab of the Backup Set editor: "On a schedule" with a list of
// schedules (each in plain words, with Edit, Delete and an on/off tick), or
// "Only when I click Run Now". Editing a schedule shows only the fields its
// unit needs and its next 3 runs.
// wakeToRun: wake the computer from sleep for the set's runs (#140); wakeInfo
// is App.GetWakeTimerInfo, what this computer allows (null while unknown).
// battery is the set's On battery rule ({ mode: 'anyway' | 'above' | 'plugged',
// start, cancel, cancelBelow }), setBattery changes part of it, and
// batteryInfo is App.GetBatteryInfo: with no battery the section is greyed.
// onEnableWake(scope) is App.EnableWakeTimers, for the alert below.
export default function ScheduleEditor({ scheduleMode, setScheduleMode, schedules, setSchedules, runAtStartup, setRunAtStartup, wakeToRun = false, setWakeToRun = () => {}, wakeInfo = null,
  battery = null, setBattery = () => {}, batteryInfo = null, onEnableWake = null }) {
  const { t, language } = useTranslation()
  // The alert under the Wake option when the power plan stops a timer waking
  // this computer (wakeInfo.alert: 'all', or 'battery' on a laptop). Only its
  // first button changes anything. "Leave" hides it on this computer: the
  // choice belongs to the computer, not the set, so it is kept in the app's
  // local storage, and for that alert only, so a plan that later stops
  // waking the computer altogether asks again.
  const [wakeLeft, setWakeLeft] = useState(readWakeAlertLeft)
  const [wakeBusy, setWakeBusy] = useState(false)
  const [wakeError, setWakeError] = useState('')
  const wakeAlert = wakeToRun && wakeInfo && wakeInfo.alert && wakeInfo.alert !== wakeLeft ? wakeInfo.alert : ''
  const leaveWake = () => {
    try { localStorage.setItem(WAKE_ALERT_LEFT, wakeAlert) } catch { /* shown again next time */ }
    setWakeLeft(wakeAlert)
    setWakeError('')
  }
  const enableWake = () => {
    if (!onEnableWake || wakeBusy) return
    setWakeBusy(true)
    setWakeError('')
    Promise.resolve(onEnableWake(wakeAlert))
      .then(() => setWakeBusy(false))
      .catch(err => {
        setWakeBusy(false)
        setWakeError(String((err && err.message) || err || ''))
      })
  }
  const noBattery = !!batteryInfo && !batteryInfo.hasBattery
  const cancelTooHigh = !!battery && battery.mode === 'above' && battery.cancel && battery.cancelBelow > 0 && battery.cancelBelow >= battery.start
  // A sentence with a number box in it: the text before and after {n} or {m}.
  const around = (key, p) => {
    const [before, after = ''] = t(key).split(`{${p}}`)
    return [before, after]
  }
  const numberBox = (id, value, min, max, disabled, onChange) => (
    <input id={id} type="number" min={min} max={max} value={value} disabled={disabled} className="sched-field narrow"
      onChange={e => onChange(Math.max(0, Math.min(999, parseInt(e.target.value, 10) || 0)))}
      onBlur={e => onChange(Math.min(max, Math.max(min, parseInt(e.target.value, 10) || min)))} />
  )
  const batterySummary = () => {
    if (!battery || noBattery) return null
    const cancel = battery.cancel && battery.cancelBelow > 0 ? t('batterySummaryCancel', { m: battery.cancelBelow }) : ''
    if (battery.mode === 'above') return t('batterySummaryAbove', { n: battery.start, cancel })
    if (battery.mode === 'plugged') return t('batterySummaryPlugged', { cancel })
    return t('batterySummaryAnyway', { cancel })
  }
  const batteryLine = batterySummary()
  const [editingId, setEditingId] = useState(null)
  const editing = schedules.find(s => s.id === editingId)

  const update = (id, patch) => setSchedules(schedules.map(s => s.id === id ? { ...s, ...patch } : s))
  const add = (unit) => {
    const s = newSchedule(unit, nextScheduleId(schedules))
    setSchedules([...schedules, s])
    setEditingId(s.id)
  }
  const remove = (id) => {
    setSchedules(schedules.filter(s => s.id !== id))
    if (editingId === id) setEditingId(null)
  }
  const fmt = d => d.toLocaleString(undefined, { weekday: 'short', day: 'numeric', month: 'short', hour: '2-digit', minute: '2-digit' })
  const setNext = scheduleMode === 'manual' ? null : nextSetRun(schedules)
  const enabledCount = schedules.filter(s => s.enabled).length

  const dayChips = (s) => (
    <div className="sched-days" role="group" aria-label={t('daysOfWeekLabel')}>
      {WEEK.map(day => {
        const selected = (s.weekdays || []).includes(day)
        return (
          <button key={day} type="button" aria-pressed={selected} className={selected ? 'sched-chip selected' : 'sched-chip'}
            onClick={() => update(s.id, { weekdays: selected ? s.weekdays.filter(d => d !== day) : [...(s.weekdays || []), day] })}>
            {t(`day${day}`)}
          </button>
        )
      })}
    </div>
  )

  const editor = (s) => {
    const problem = scheduleProblem(s)
    const runs = problem ? [] : nextRuns(s, new Date(), 3)
    const windowed = !!(s.windowStart && s.windowEnd)
    return (
      <div className="schedule-editor" id="schedule-editor">
        <strong>{t('schedEditTitle')}</strong>
        <div className="sched-row">
          <span className="sched-key">{t('schedRepeatEvery')}</span>
          <input id="sched-every" type="number" min={s.unit === 'minutes' ? MIN_REPEAT_MINUTES : 1} value={s.every} className="sched-field narrow"
            onChange={e => update(s.id, { every: Math.max(1, parseInt(e.target.value, 10) || 1) })} />
          <select id="sched-unit" value={s.unit} className="sched-field" aria-label={t('schedUnitLabel')}
            onChange={e => {
              const unit = e.target.value
              // A fresh schedule of the new unit, keeping what carries over.
              const fresh = newSchedule(unit, s.id)
              const keepDays = unit === 'weeks' ? ((s.weekdays || []).length ? s.weekdays : fresh.weekdays) : (s.weekdays || [])
              setSchedules(schedules.map(x => x.id === s.id
                ? { ...fresh, enabled: s.enabled, start: s.start, at: fresh.at && s.at ? s.at : fresh.at, weekdays: keepDays }
                : x))
            }}>
            {UNITS.map(u => <option key={u} value={u}>{t(`schedUnit_${u}`)}</option>)}
          </select>
        </div>

        {(s.unit === 'minutes' || s.unit === 'hours') && (
          <>
            <div className="sched-row" role="radiogroup" aria-label={t('schedWhen')}>
              <span className="sched-key">{t('schedWhen')}</span>
              <label className="sched-radio">
                <input type="radio" name="sched-window" id="sched-allday" checked={!windowed}
                  onChange={() => update(s.id, { windowStart: '', windowEnd: '' })} /> {t('windowAllDay')}
              </label>
              <label className="sched-radio">
                <input type="radio" name="sched-window" id="sched-between" checked={windowed}
                  onChange={() => update(s.id, { windowStart: '08:00', windowEnd: '18:00' })} /> {t('windowBetween')}
              </label>
              <input id="sched-win-start" type="time" value={s.windowStart || '08:00'} disabled={!windowed} className="sched-field" aria-label={t('schedFrom')}
                onChange={e => e.target.value && update(s.id, { windowStart: e.target.value })} />
              <span>{t('windowAnd')}</span>
              <input id="sched-win-end" type="time" value={s.windowEnd || '18:00'} disabled={!windowed} className="sched-field" aria-label={t('schedTo')}
                onChange={e => e.target.value && update(s.id, { windowEnd: e.target.value })} />
            </div>
            {s.unit === 'hours' && !windowed && (
              <div className="sched-row">
                <span className="sched-key">{t('schedAtMinute')}</span>
                <input id="sched-minute" type="number" min="0" max="59" value={s.minute || 0} className="sched-field narrow"
                  onChange={e => update(s.id, { minute: Math.min(59, Math.max(0, parseInt(e.target.value, 10) || 0)) })} />
                <span>{t('schedPastHour')}</span>
              </div>
            )}
            <div className="sched-row"><span className="sched-key">{t('schedOnlyOn')}</span>{dayChips(s)}</div>
            <div className="sched-hint">{t('schedNoDayEveryDay')}</div>
          </>
        )}

        {s.unit === 'days' && (
          <>
            <div className="sched-row"><span className="sched-key">{t('schedAt')}</span>
              <input id="sched-at" type="time" value={s.at} className="sched-field" onChange={e => e.target.value && update(s.id, { at: e.target.value })} /></div>
            <div className="sched-row"><span className="sched-key">{t('schedOnlyOn')}</span>{dayChips(s)}</div>
            <div className="sched-hint">{t('schedNoDayEveryDay')}</div>
          </>
        )}

        {s.unit === 'weeks' && (
          <>
            <div className="sched-row"><span className="sched-key">{t('schedOn')}</span>{dayChips(s)}</div>
            <div className="sched-row"><span className="sched-key">{t('schedAt')}</span>
              <input id="sched-at" type="time" value={s.at} className="sched-field" onChange={e => e.target.value && update(s.id, { at: e.target.value })} /></div>
          </>
        )}

        {s.unit === 'months' && (
          <>
            <div role="radiogroup" aria-label={t('schedOn')}>
              <div className="sched-row">
                <span className="sched-key">{t('schedOn')}</span>
                <label className="sched-radio">
                  <input type="radio" name="sched-mm" id="sched-mm-day" checked={s.monthMode === 'day'} onChange={() => update(s.id, { monthMode: 'day' })} /> {t('schedMonthDay')}
                </label>
                <input id="sched-monthday" type="number" min="1" max="31" value={s.monthDay || 1} disabled={s.monthMode !== 'day'} className="sched-field narrow"
                  aria-label={t('schedMonthDay')} onChange={e => update(s.id, { monthDay: Math.min(31, Math.max(1, parseInt(e.target.value, 10) || 1)) })} />
              </div>
              <div className="sched-row"><span className="sched-key"></span>
                <label className="sched-radio">
                  <input type="radio" name="sched-mm" id="sched-mm-last" checked={s.monthMode === 'last'} onChange={() => update(s.id, { monthMode: 'last' })} /> {t('schedMonthLast')}
                </label>
              </div>
              <div className="sched-row"><span className="sched-key"></span>
                <label className="sched-radio">
                  <input type="radio" name="sched-mm" id="sched-mm-nth" checked={s.monthMode === 'nth'} onChange={() => update(s.id, { monthMode: 'nth' })} /> {t('schedMonthThe')}
                </label>
                <select id="sched-nth" value={s.nth || 1} disabled={s.monthMode !== 'nth'} className="sched-field" aria-label={t('schedWhich')}
                  onChange={e => update(s.id, { nth: Number(e.target.value) })}>
                  {[1, 2, 3, 4, -1].map(n => <option key={n} value={n}>{nthWord(t, n, s.nthWeekday || 'Sun')}</option>)}
                </select>
                <select id="sched-nthwd" value={s.nthWeekday || 'Sun'} disabled={s.monthMode !== 'nth'} className="sched-field" aria-label={t('daysOfWeekLabel')}
                  onChange={e => update(s.id, { nthWeekday: e.target.value })}>
                  {WEEK.map(w => <option key={w} value={w}>{weekdayName(language, w)}</option>)}
                </select>
              </div>
            </div>
            <div className="sched-row"><span className="sched-key">{t('schedAt')}</span>
              <input id="sched-at" type="time" value={s.at} className="sched-field" onChange={e => e.target.value && update(s.id, { at: e.target.value })} /></div>
            {s.monthMode === 'day' && s.monthDay > 28 && <div className="sched-hint">{t('schedShortMonthHint', { d: s.monthDay })}</div>}
          </>
        )}

        {s.unit === 'years' && (
          <>
            <div className="sched-row"><span className="sched-key">{t('schedOn')}</span>
              <input id="sched-yearday" type="number" min="1" max="31" value={s.yearDay} className="sched-field narrow" aria-label={t('schedMonthDay')}
                onChange={e => update(s.id, { yearDay: Math.min(31, Math.max(1, parseInt(e.target.value, 10) || 1)) })} />
              <select id="sched-yearmonth" value={s.yearMonth} className="sched-field" aria-label={t('schedMonth')}
                onChange={e => update(s.id, { yearMonth: Number(e.target.value) })}>
                {Array.from({ length: 12 }, (_, i) => <option key={i + 1} value={i + 1}>{monthName(language, i + 1)}</option>)}
              </select>
            </div>
            <div className="sched-row"><span className="sched-key">{t('schedAt')}</span>
              <input id="sched-at" type="time" value={s.at} className="sched-field" onChange={e => e.target.value && update(s.id, { at: e.target.value })} /></div>
            {s.yearMonth === 2 && s.yearDay === 29 && <div className="sched-hint">{t('schedFeb29Hint')}</div>}
          </>
        )}

        {s.every > 1 && ['days', 'weeks', 'months', 'years'].includes(s.unit) && (
          <div className="sched-row"><span className="sched-key">{t('schedStarting')}</span>
            <input id="sched-start" type="date" value={s.start || ''} className="sched-field" onChange={e => e.target.value && update(s.id, { start: e.target.value })} />
            <span className="sched-hint">{tp(t, language, `schedCountsFrom_${s.unit}`, s.every)}</span>
          </div>
        )}

        {problem && <div className="sched-warn">{t(problem, { n: MIN_REPEAT_MINUTES })}</div>}

        <div className="sched-next">
          <strong>{t('schedNext3')}</strong>
          {runs.length > 0
            ? <ol>{runs.map(d => <li key={d.getTime()}>{fmt(d)}</li>)}</ol>
            : !problem && <div className="sched-hint">{t('schedNoneSoon')}</div>}
        </div>
        <div className="sched-done">
          <button type="button" className="btn btn-secondary" id="sched-done" onClick={() => setEditingId(null)}>{t('schedDone')}</button>
        </div>
      </div>
    )
  }

  return (
    <div>
      <div className="form-group">
        <label id="sched-mode-label">{t('schedWhenLabel')}</label>
        <div role="radiogroup" aria-labelledby="sched-mode-label" className="sched-modes">
          <label className="sched-radio">
            <input type="radio" name="sched-mode" id="sched-mode-scheduled" checked={scheduleMode !== 'manual'} onChange={() => setScheduleMode('scheduled')} />
            {t('schedModeScheduled')}
          </label>
          <label className="sched-radio">
            <input type="radio" name="sched-mode" id="sched-mode-manual" checked={scheduleMode === 'manual'} onChange={() => setScheduleMode('manual')} />
            {t('schedModeManual')}
          </label>
        </div>
      </div>

      {scheduleMode !== 'manual' && (
        <>
          <div className="form-group">
            <label>{t('schedListLabel')}</label>
            <ul className="sched-list" id="schedule-list">
              {schedules.length === 0 && <li className="sched-list-empty">{t('schedNone')}</li>}
              {schedules.map(s => (
                <li key={s.id} className={s.id === editingId ? 'schedule-row editing' : 'schedule-row'}>
                  <input type="checkbox" id={`sched-on-${s.id}`} checked={s.enabled} aria-label={t('schedOnLabel')}
                    onChange={e => update(s.id, { enabled: e.target.checked })} />
                  <span className={s.enabled ? 'sched-text' : 'sched-text off'}>
                    {describe(t, language, s)}
                    {!s.enabled && <span className="sched-note"> ({t('schedOff')})</span>}
                    {scheduleProblem(s) && <span className="sched-note problem"> ({t(scheduleProblem(s), { n: MIN_REPEAT_MINUTES })})</span>}
                  </span>
                  <span className="sched-buttons">
                    <button type="button" className="btn btn-secondary" onClick={() => setEditingId(s.id === editingId ? null : s.id)}>{t('edit')}</button>
                    <button type="button" className="btn btn-secondary" onClick={() => remove(s.id)}>{t('delete')}</button>
                  </span>
                </li>
              ))}
            </ul>
            <div className="sched-add-row">
              <button type="button" className="btn btn-secondary" id="sched-add" onClick={() => add('days')}>{t('schedAdd')}</button>
              <span>{t('schedAddFrom')}</span>
              <button type="button" className="btn btn-secondary" id="sched-add-weekly" onClick={() => add('weeks')}>{t('schedBadgeWeekly')}</button>
              <button type="button" className="btn btn-secondary" id="sched-add-monthly" onClick={() => add('months')}>{t('schedBadgeMonthly')}</button>
              <button type="button" className="btn btn-secondary" id="sched-add-yearly" onClick={() => add('years')}>{t('schedBadgeYearly')}</button>
            </div>
            {editing && editor(editing)}
          </div>

          <div className="form-group">
            <label className="sched-startup">
              <input type="checkbox" id="sched-startup" checked={runAtStartup} onChange={e => setRunAtStartup(e.target.checked)} />
              <span>{t('runAtStartup')}</span>
            </label>
          </div>

          <div className="form-group">
            <label className="sched-startup">
              <input type="checkbox" id="sched-wake" checked={wakeToRun} onChange={e => setWakeToRun(e.target.checked)} />
              <span>{t('wakeToRun')}</span>
            </label>
            {wakeAlert && (
              <div className="sched-warn" id="sched-wake-alert" role="alert">
                <div>{t(wakeAlert === 'all' ? 'wakeAlertAll' : 'wakeAlertBattery')}</div>
                {wakeError && <div id="sched-wake-alert-error">{t('wakeEnableFailed', { error: wakeError })}</div>}
                <div className="sched-add-row">
                  <button type="button" className="btn btn-primary" id="sched-wake-enable" disabled={wakeBusy || !onEnableWake} onClick={enableWake}>
                    {t(wakeAlert === 'all' ? 'wakeEnableAll' : 'wakeEnableBattery')}
                  </button>
                  <button type="button" className="btn btn-secondary" id="sched-wake-leave" disabled={wakeBusy} onClick={leaveWake}>
                    {t(wakeAlert === 'all' ? 'wakeLeaveOff' : 'wakeLeaveMainsOnly')}
                  </button>
                </div>
              </div>
            )}
            {wakeToRun && (
              <p className="sched-hint" id="sched-wake-note">
                {t('wakeNote')}
                {wakeInfo && !wakeInfo.supported && <><br />{t('wakeUnsupported')}</>}
                {/* The short notes say what the alert above already asks about: shown once it is left. */}
                {wakeInfo && wakeInfo.supported && !wakeInfo.onMains && wakeAlert !== 'all' && <><br />{t('wakeBlocked')}</>}
                {wakeInfo && wakeInfo.supported && wakeInfo.onMains && !wakeInfo.onBattery && wakeAlert !== 'battery' && <><br />{t('wakeMainsOnly')}</>}
              </p>
            )}
            {/* Modern Standby (S0 low power idle): Windows may ignore the wake,
                keep awake and sleep requests, so waking is best effort (#142). */}
            {wakeToRun && wakeInfo && wakeInfo.supported && wakeInfo.modernStandby && (
              <div className="sched-warn" id="sched-wake-modern" role="note" style={{ marginTop: '6px' }}>
                <strong>{t('wakeModernStandbyTitle')}</strong>
                <div>{t('wakeModernStandby')}</div>
              </div>
            )}
          </div>
        </>
      )}

      {battery && (() => {
        const [aboveBefore, aboveAfter] = around('batteryAbove', 'n')
        const [cancelBefore, cancelAfter] = around('batteryCancel', 'm')
        return (
          <div className={noBattery ? 'form-group sched-battery off' : 'form-group sched-battery'} id="sched-battery"
            style={noBattery ? { opacity: 0.55 } : undefined}>
            <label id="sched-battery-label">
              {t('batteryHeading')} <span className="sched-hint" style={{ fontWeight: 'normal' }}>{t('batteryHeadingNote')}</span>
            </label>
            <div role="radiogroup" aria-labelledby="sched-battery-label">
              <div className="sched-row">
                <label className="sched-radio">
                  <input type="radio" name="sched-battery" id="sched-battery-plugged" disabled={noBattery}
                    checked={battery.mode === 'plugged'} onChange={() => setBattery({ mode: 'plugged' })} /> {t('batteryPlugged')}
                </label>
              </div>
              <div className="sched-row">
                <label className="sched-radio">
                  <input type="radio" name="sched-battery" id="sched-battery-above" disabled={noBattery}
                    checked={battery.mode === 'above'} onChange={() => setBattery({ mode: 'above' })} /> {aboveBefore}
                </label>
                {numberBox('sched-battery-start', battery.start, 5, 95, noBattery || battery.mode !== 'above', v => setBattery({ start: v }))}
                <span>{aboveAfter}</span>
              </div>
              <div className="sched-row">
                <label className="sched-radio">
                  <input type="radio" name="sched-battery" id="sched-battery-anyway" disabled={noBattery}
                    checked={battery.mode === 'anyway'} onChange={() => setBattery({ mode: 'anyway' })} /> {t('batteryAnyway')}
                </label>
              </div>
            </div>
            <hr style={{ border: 0, borderTop: '1px solid #e2e8f0', margin: '8px 0' }} />
            <div className="sched-row">
              <label className="sched-startup">
                <input type="checkbox" id="sched-battery-cancel" disabled={noBattery} checked={battery.cancel}
                  onChange={e => setBattery({ cancel: e.target.checked })} />
                <span>{cancelBefore}</span>
              </label>
              {numberBox('sched-battery-cancel-below', battery.cancelBelow, 0, 99, noBattery || !battery.cancel, v => setBattery({ cancelBelow: v }))}
              <span>{cancelAfter}</span>
            </div>
            <p className="sched-hint" id="sched-battery-hint">{t('batteryCancelHint')}</p>
            {cancelTooHigh && !noBattery && <div className="sched-warn" id="sched-battery-warn">{t('batteryCancelTooHigh', { n: battery.start })}</div>}
            {noBattery && <p className="sched-hint" id="sched-battery-none">{t('batteryNone')}</p>}
          </div>
        )
      })()}

      <div className="info-box schedule-summary" id="schedule-summary">
        {scheduleMode === 'manual'
          ? <>{t('schedulingInfoManual')}{schedules.length > 0 && <><br />{t('schedManualKept')}</>}</>
          : setNext
            ? <>{t('schedSetNext', { time: fmt(setNext.time) })} ({describe(t, language, setNext.schedule)})
                {enabledCount > 1 && <><br />{tp(t, language, 'schedCountOn', enabledCount)}</>}
                {runAtStartup && <><br />{t('andAtStartup')}</>}
                {wakeToRun && <><br />{t('andWakes')}</>}
                {batteryLine && <><br /><span id="schedule-summary-battery">{batteryLine}</span></>}
                <br />{t('schedCatchUp')}</>
            : <>{t('schedNoneOn')}{runAtStartup && <><br />{t('andAtStartup')}</>}
                {batteryLine && <><br /><span id="schedule-summary-battery">{batteryLine}</span></>}</>}
      </div>
    </div>
  )
}
