import { useEffect, useState } from 'react'
import { useTranslation } from '../i18n/i18nContext'
import {
  WEEK, UNITS, MIN_REPEAT_MINUTES, newSchedule, nextScheduleId, describe, nextRuns,
  nextSetRun, scheduleProblem, weekdayName, monthName, tp, nthWord,
  newServerTrigger, CONNECTIONS,
} from './schedules'

// The longest a schedule's run waits for its server, and a server trigger's
// "at most every" (gui/schedules.go maxWaitHours, server_trigger.go
// maxServerEveryHours).
const MAX_WAIT_HOURS = 168
const MAX_EVERY_HOURS = 24 * 31

// connLabel is the name of a kind of connection ("wifi" -> "Wi-Fi").
const connLabel = (t, kind) => t(`connKind_${CONNECTIONS.includes(kind) ? kind : 'other'}`)

// serverSummary says a server trigger in plain words: "Every 24 hours at
// most, on Home, not on metered connections".
export function serverSummary(t, language, st) {
  const h = st.everyHours > 0 ? st.everyHours : 24
  const every = h >= 48 && h % 24 === 0 ? tp(t, language, 'schedEvery_days', h / 24) : tp(t, language, 'schedEvery_hours', h)
  const parts = [t('trigServerSummary', { every })]
  if ((st.networks || []).length) parts.push(t('trigSumNetworks', { names: st.networks.join(', ') }))
  if ((st.connections || []).length) parts.push((st.connections || []).map(k => connLabel(t, k)).join(', '))
  if (st.notMetered) parts.push(t('trigSumNotMetered'))
  return parts.join(', ')
}

// timesExtra says a schedule's server and network choices, after its times.
function timesExtra(t, s) {
  const parts = []
  if (s.ifUnreachable === 'wait') parts.push(t('trigSumWaits', { n: s.waitHours > 0 ? s.waitHours : 12, unit: t('schedUnit_hours') }))
  if ((s.networks || []).length) parts.push(t('trigSumNetworks', { names: s.networks.join(', ') }))
  if (s.notMetered) parts.push(t('trigSumNotMetered'))
  return parts.length ? ', ' + parts.join(', ') : ''
}

// NetworkList is a list of the networks this computer has joined, each with
// a tick, and whether it is connected now or when it was last seen. Names a
// set already has but the computer does not know are listed too.
function NetworkList({ t, idPrefix, info, chosen, disabled, onChange }) {
  const known = (info && info.known) || []
  const rows = [...known]
  for (const n of chosen) {
    if (!known.some(k => k.name.toLowerCase() === n.toLowerCase())) rows.push({ name: n, kind: '', connected: false, lastSeen: '', unknown: true })
  }
  const isOn = name => chosen.some(n => n.toLowerCase() === name.toLowerCase())
  const toggle = name => onChange(isOn(name) ? chosen.filter(n => n.toLowerCase() !== name.toLowerCase()) : [...chosen, name])
  const seen = iso => {
    const d = new Date(iso)
    if (Number.isNaN(d.getTime())) return ''
    try { return d.toLocaleDateString(undefined, { weekday: 'short', day: 'numeric', month: 'short' }) } catch { return iso }
  }
  if (rows.length === 0) return <div className="sched-hint" id={`${idPrefix}-none`}>{t('trigNetEmpty')}</div>
  return (
    <ul className="trig-nets" id={`${idPrefix}-list`}>
      {rows.map((n, i) => (
        <li key={n.name}>
          <label className="trig-net">
            <input type="checkbox" id={`${idPrefix}-${i}`} checked={isOn(n.name)} disabled={disabled} onChange={() => toggle(n.name)} />
            <span className="trig-net-name"><strong>{n.name}</strong>{n.kind && <span className="sched-hint"> {connLabel(t, n.kind)}</span>}</span>
            <span className={n.connected ? 'trig-net-seen now' : 'trig-net-seen'}>
              {n.connected ? t('trigNetConnected') : n.unknown ? t('trigNetNotSeen') : n.lastSeen ? t('trigNetLastSeen', { date: seen(n.lastSeen) }) : ''}
            </span>
          </label>
        </li>
      ))}
    </ul>
  )
}

// Where the app keeps which wake alert the user chose to leave as it is.
const WAKE_ALERT_LEFT = 'wakeAlertLeft'
const readWakeAlertLeft = () => {
  try { return localStorage.getItem(WAKE_ALERT_LEFT) || '' } catch { return '' }
}

// The When to run tab of the Backup Set editor: a list of the set's triggers,
// each in plain words with an on/off tick, Edit and Delete. The first row,
// "By hand: When I click Run Now", is always there and always on (it is not
// stored: every set can be run with Run Now). Then:
//   - At set times: each of the set's schedules. Editing one shows only the
//     fields its unit needs, its next 3 runs, and what a run does when the
//     server cannot be reached at its time (fail, or wait up to N hours),
//     and the networks it may run on;
//   - When the server can be reached (serverTrigger, null for none): at most
//     every N hours, on which networks and connections, and a "Right now"
//     panel from getTriggerStatus(trigger) (App.GetServerTriggerStatus);
//   - Computer events: when the computer starts (runAtStartup).
// getNetworkInfo is App.GetNetworkInfo, the networks the computer knows.
// wakeToRun: wake the computer from sleep for the set's runs (#140); wakeInfo
// is App.GetWakeTimerInfo, what this computer allows (null while unknown).
// battery is the set's On battery rule ({ mode: 'anyway' | 'above' | 'plugged',
// start, cancel, cancelBelow }), setBattery changes part of it, and
// batteryInfo is App.GetBatteryInfo: with no battery the section is greyed.
// onEnableWake(scope) is App.EnableWakeTimers, for the alert below.
export default function ScheduleEditor({ schedules, setSchedules, runAtStartup, setRunAtStartup,
  serverTrigger = null, setServerTrigger = () => {}, getNetworkInfo = null, getTriggerStatus = null,
  wakeToRun = false, setWakeToRun = () => {}, wakeInfo = null,
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
  // The trigger being edited: a schedule's ID, 'server' or 'events'.
  const [editingId, setEditingId] = useState(null)
  const editing = schedules.find(s => s.id === editingId)
  // The Computer events row stays in the list while its tick is off, until
  // it is deleted (runAtStartup is its only setting for now).
  const [eventsShown, setEventsShown] = useState(!!runAtStartup)
  const showEvents = eventsShown || runAtStartup

  // The computer's networks (App.GetNetworkInfo): read when an editor that
  // offers them opens.
  const [netInfo, setNetInfo] = useState(null)
  const wantsNets = editingId === 'server' || !!editing
  useEffect(() => {
    if (!wantsNets || !getNetworkInfo) return
    let live = true
    Promise.resolve(getNetworkInfo()).then(i => { if (live) setNetInfo(i || null) }).catch(() => { if (live) setNetInfo(null) })
    return () => { live = false }
  }, [wantsNets, getNetworkInfo])
  const netsUnsupported = !!netInfo && !netInfo.supported

  // The server trigger's "Right now" panel, worked out again a moment after
  // a change (it tries the server).
  const [status, setStatus] = useState(null)
  // The server trigger's "at most every" unit as chosen (null: from its hours).
  const [serverInDays, setServerInDays] = useState(null)
  const statusKey = editingId === 'server' && serverTrigger ? JSON.stringify(serverTrigger) : ''
  useEffect(() => {
    if (!statusKey || !getTriggerStatus) { setStatus(null); return }
    let live = true
    setStatus(null)
    const timer = setTimeout(() => {
      Promise.resolve(getTriggerStatus(JSON.parse(statusKey)))
        .then(s => { if (live) setStatus(s || null) })
        .catch(() => { if (live) setStatus(null) })
    }, 600)
    return () => { live = false; clearTimeout(timer) }
  }, [statusKey, getTriggerStatus])

  const update = (id, patch) => setSchedules(schedules.map(s => s.id === id ? { ...s, ...patch } : s))
  const updateServer = patch => setServerTrigger({ ...(serverTrigger || newServerTrigger()), ...patch })
  const add = (unit) => {
    const s = newSchedule(unit, nextScheduleId(schedules))
    setSchedules([...schedules, s])
    setEditingId(s.id)
  }
  const addServer = () => {
    if (!serverTrigger) setServerTrigger(newServerTrigger())
    setEditingId('server')
  }
  const addEvents = () => {
    setRunAtStartup(true)
    setEventsShown(true)
    setEditingId('events')
  }
  const remove = (id) => {
    setSchedules(schedules.filter(s => s.id !== id))
    if (editingId === id) setEditingId(null)
  }
  const removeServer = () => {
    setServerTrigger(null)
    if (editingId === 'server') setEditingId(null)
  }
  const removeEvents = () => {
    setRunAtStartup(false)
    setEventsShown(false)
    if (editingId === 'events') setEditingId(null)
  }
  // The editor's Runs list: a trigger changed into another kind. A kind the
  // set already has is offered greyed.
  const changeKind = (from, to) => {
    if (to === 'times') add('days')
    else if (to === 'server') addServer()
    else if (to === 'events') addEvents()
    if (from === 'server') setServerTrigger(null)
    else if (from === 'events') { setRunAtStartup(false); setEventsShown(false) }
    else if (from) setSchedules(prev => prev.filter(s => s.id !== from))
  }
  const fmt = d => d.toLocaleString(undefined, { weekday: 'short', day: 'numeric', month: 'short', hour: '2-digit', minute: '2-digit' })
  const setNext = nextSetRun(schedules)
  const enabledCount = schedules.filter(s => s.enabled).length
  const serverOn = !!serverTrigger && serverTrigger.enabled
  const anyOn = enabledCount > 0 || serverOn || runAtStartup

  // The Runs list at the top of each editor.
  const runsSelect = (kind, from) => (
    <div className="sched-row">
      <span className="sched-key">{t('trigRuns')}</span>
      <select id="trig-runs" className="sched-field" value={kind} aria-label={t('trigRuns')}
        onChange={e => e.target.value !== kind && changeKind(from, e.target.value)}>
        <option value="times">{t('trigKindTimes')}</option>
        <option value="server" disabled={!!serverTrigger && kind !== 'server'}>{t('trigKindServer')}</option>
        <option value="events" disabled={showEvents && kind !== 'events'}>{t('trigKindEvents')}</option>
      </select>
    </div>
  )
  const hoursBox = (id, value, max, onChange, label) => (
    <input id={id} type="number" min="1" max={max} value={value} className="sched-field narrow" aria-label={label}
      onChange={e => onChange(Math.max(0, Math.min(max, parseInt(e.target.value, 10) || 0)))}
      onBlur={e => onChange(Math.min(max, Math.max(1, parseInt(e.target.value, 10) || 1)))} />
  )

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
        <strong>{t('trigEditTitle')}</strong>
        {runsSelect('times', s.id)}
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
                ? { ...fresh, enabled: s.enabled, start: s.start, at: fresh.at && s.at ? s.at : fresh.at, weekdays: keepDays,
                    ifUnreachable: s.ifUnreachable, waitHours: s.waitHours, networks: s.networks, notMetered: s.notMetered, onlyNets: s.onlyNets }
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

        {/* What a run does when the server cannot be reached at its time,
            and the networks it may run on (gui/server_trigger.go). */}
        <div className="trig-box" id="sched-server">
          <div className="trig-box-title" id="sched-unreach-label">{t('trigIfUnreachable')}</div>
          <div role="radiogroup" aria-labelledby="sched-unreach-label">
            <div className="sched-row">
              <label className="sched-radio">
                <input type="radio" name="sched-unreach" id="sched-unreach-fail" checked={s.ifUnreachable !== 'wait'}
                  onChange={() => update(s.id, { ifUnreachable: '', waitHours: 0 })} /> {t('trigFail')}
              </label>
            </div>
            <div className="sched-row">
              <label className="sched-radio">
                <input type="radio" name="sched-unreach" id="sched-unreach-wait" checked={s.ifUnreachable === 'wait'}
                  onChange={() => update(s.id, { ifUnreachable: 'wait', waitHours: s.waitHours > 0 ? s.waitHours : 12 })} /> {t('trigWait')}
              </label>
              {hoursBox('sched-wait-hours', s.waitHours > 0 ? s.waitHours : 12, MAX_WAIT_HOURS, v => update(s.id, { waitHours: v }), t('schedUnit_hours'))}
              <span>{t('schedUnit_hours')}</span>
            </div>
          </div>
          <div className="sched-hint">{t('trigWaitHint')}</div>
          <div className="sched-row">
            <label className="sched-startup">
              <input type="checkbox" id="sched-only-nets" disabled={netsUnsupported} checked={(s.networks || []).length > 0 || !!s.onlyNets}
                onChange={e => update(s.id, e.target.checked ? { onlyNets: true } : { onlyNets: false, networks: [] })} />
              <span>{t('trigOnlyNetworks')}</span>
            </label>
          </div>
          {((s.networks || []).length > 0 || s.onlyNets) && (
            <>
              <NetworkList t={t} idPrefix="sched-net" info={netInfo} chosen={s.networks || []} disabled={netsUnsupported}
                onChange={list => update(s.id, { networks: list })} />
              {(s.networks || []).length === 0 && <div className="sched-hint">{t('trigNetNoneTicked')}</div>}
            </>
          )}
          <div className="sched-row">
            <label className="sched-startup">
              <input type="checkbox" id="sched-not-metered" disabled={netsUnsupported} checked={!!s.notMetered}
                onChange={e => update(s.id, { notMetered: e.target.checked })} />
              <span>{t('trigNotMetered')}</span>
            </label>
          </div>
          {netsUnsupported && <div className="sched-hint">{t('trigNetUnsupported')}</div>}
          {wakeToRun && <div className="sched-hint">{t('trigNetWokenHint')}</div>}
        </div>

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

  // "When the server can be reached".
  const serverEditor = (st) => {
    const h = st.everyHours > 0 ? st.everyHours : 24
    // Shown in days from 2 days up, so the usual 24 stays "24 hours".
    const inDays = serverInDays ?? (h >= 48 && h % 24 === 0)
    const onlyThese = (st.networks || []).length > 0 || !!st.onlyNets
    const conns = st.connections || []
    const checkText = c => {
      const p = { ...(c.params || {}) }
      if (typeof p.time === 'string') {
        const d = new Date(p.time)
        if (!Number.isNaN(d.getTime())) p.time = fmt(d)
      }
      return t(c.key, p)
    }
    return (
      <div className="schedule-editor" id="server-editor">
        <strong>{t('trigEditTitle')}</strong>
        {runsSelect('server', 'server')}
        <div className="sched-row">
          <span className="sched-key">{t('trigAtMost')}</span>
          {hoursBox('server-every', inDays ? h / 24 : h, inDays ? MAX_EVERY_HOURS / 24 : MAX_EVERY_HOURS,
            v => updateServer({ everyHours: inDays ? v * 24 : v }), t('trigAtMost'))}
          <select id="server-every-unit" className="sched-field" value={inDays ? 'days' : 'hours'} aria-label={t('schedUnitLabel')}
            onChange={e => {
              const days = e.target.value === 'days'
              setServerInDays(days)
              updateServer({ everyHours: days ? Math.max(1, Math.round(h / 24)) * 24 : h })
            }}>
            <option value="hours">{t('schedUnit_hours')}</option>
            <option value="days">{t('schedUnit_days')}</option>
          </select>
        </div>
        <div className="sched-hint">{t('trigAtMostHint')}</div>

        <div className="sched-row" role="radiogroup" aria-labelledby="server-nets-label">
          <span className="sched-key" id="server-nets-label">{t('trigNetworks')}</span>
          <label className="sched-radio">
            <input type="radio" name="server-nets" id="server-nets-any" disabled={netsUnsupported} checked={!onlyThese}
              onChange={() => updateServer({ networks: [], onlyNets: false })} /> {t('trigNetAny')}
          </label>
          <label className="sched-radio">
            <input type="radio" name="server-nets" id="server-nets-only" disabled={netsUnsupported} checked={onlyThese}
              onChange={() => updateServer({ onlyNets: true })} /> {t('trigNetOnly')}
          </label>
        </div>
        {onlyThese && (
          <>
            <NetworkList t={t} idPrefix="server-net" info={netInfo} chosen={st.networks || []} disabled={netsUnsupported}
              onChange={list => updateServer({ networks: list })} />
            {(st.networks || []).length === 0 && <div className="sched-hint">{t('trigNetNoneTicked')}</div>}
          </>
        )}
        <div className="sched-hint">{netsUnsupported ? t('trigNetUnsupported') : t('trigNetHint')}</div>

        <div className="sched-row">
          <span className="sched-key">{t('trigConnection')}</span>
          <div className="sched-days" role="group" aria-label={t('trigConnection')}>
            {CONNECTIONS.map(k => {
              // No kind picked means any: all show as picked.
              const on = conns.length === 0 || conns.includes(k)
              return (
                <button key={k} type="button" id={`server-conn-${k}`} aria-pressed={on} disabled={netsUnsupported}
                  className={on ? 'sched-chip selected' : 'sched-chip'}
                  onClick={() => {
                    const cur = conns.length === 0 ? [...CONNECTIONS] : conns
                    const next = on ? cur.filter(x => x !== k) : [...cur, k]
                    if (next.length === 0) return // at least one kind
                    updateServer({ connections: next.length === CONNECTIONS.length ? [] : CONNECTIONS.filter(x => next.includes(x)) })
                  }}>
                  {connLabel(t, k)}
                </button>
              )
            })}
          </div>
        </div>
        <div className="sched-row">
          <span className="sched-key"></span>
          <label className="sched-startup">
            <input type="checkbox" id="server-not-metered" disabled={netsUnsupported} checked={!!st.notMetered}
              onChange={e => updateServer({ notMetered: e.target.checked })} />
            <span>{t('trigNotMetered')}</span>
          </label>
        </div>

        <div className="sched-next" id="server-now">
          <strong>{t('trigRightNow')}</strong>
          {!status
            ? <div className="sched-hint">{t('trigChecking')}</div>
            : (
              <>
                <ul className="trig-checks">
                  {status.checks.map((c, i) => (
                    <li key={i} className={c.ok ? 'ok' : 'no'}><span aria-hidden="true">{c.ok ? '✓' : '✗'}</span> {checkText(c)}</li>
                  ))}
                </ul>
                <div className={status.verdict === 'now' ? 'trig-verdict ok' : 'trig-verdict'} id="server-verdict">
                  {status.verdict === 'now' ? t('trigVerdictNow')
                    : status.verdict === 'notYet' ? t('trigVerdictNotYet', { time: status.nextAt ? fmt(new Date(status.nextAt)) : '' })
                    : t('trigVerdictNo')}
                </div>
              </>
            )}
        </div>
        <div className="sched-done">
          <button type="button" className="btn btn-secondary" id="server-done" onClick={() => setEditingId(null)}>{t('schedDone')}</button>
        </div>
      </div>
    )
  }

  // Computer events: for now, when the computer starts.
  const eventsEditor = () => (
    <div className="schedule-editor" id="events-editor">
      <strong>{t('trigEditTitle')}</strong>
      {runsSelect('events', 'events')}
      <div className="sched-row">
        <label className="sched-startup">
          <input type="checkbox" id="sched-startup" checked={runAtStartup} onChange={e => setRunAtStartup(e.target.checked)} />
          <span>{t('trigEventsStarts')}</span>
        </label>
      </div>
      <div className="sched-hint">{t('trigEventsHint')}</div>
      <div className="sched-done">
        <button type="button" className="btn btn-secondary" id="events-done" onClick={() => setEditingId(null)}>{t('schedDone')}</button>
      </div>
    </div>
  )

  const row = ({ id, key, on, onToggle, kind, what, off, extra, onEdit, onDelete, fixed, editingRow }) => (
    <li key={key} id={id} className={editingRow ? 'schedule-row editing' : 'schedule-row'}>
      <input type="checkbox" id={`${id}-on`} checked={on} disabled={!!fixed} aria-label={t('schedOnLabel')}
        onChange={e => onToggle && onToggle(e.target.checked)} />
      <span className={on ? 'sched-text trig-text' : 'sched-text trig-text off'}>
        <span className="trig-kind">{kind}</span>
        <span>
          {what}
          {off && <span className="sched-note"> ({t('schedOff')})</span>}
          {extra}
        </span>
      </span>
      {!fixed && (
        <span className="sched-buttons">
          <button type="button" className="btn btn-secondary" onClick={onEdit}>{t('edit')}</button>
          <button type="button" className="btn btn-secondary" onClick={onDelete}>{t('delete')}</button>
        </span>
      )}
    </li>
  )

  return (
    <div>
      <div className="form-group">
        <label id="trig-list-label">{t('trigListLabel')}</label>
        <ul className="sched-list" id="schedule-list" aria-labelledby="trig-list-label">
          {/* Every set can be run with Run Now: always on, not stored. */}
          {row({ id: 'trig-hand', key: 'hand', on: true, fixed: true, kind: t('trigKindHand'), what: t('trigHandWhat') })}
          {schedules.map(s => row({
            id: `sched-${s.id}`, key: s.id, on: s.enabled, onToggle: v => update(s.id, { enabled: v }),
            kind: t('trigKindTimes'), what: describe(t, language, s) + timesExtra(t, s), off: !s.enabled,
            extra: scheduleProblem(s) && <span className="sched-note problem"> ({t(scheduleProblem(s), { n: MIN_REPEAT_MINUTES })})</span>,
            onEdit: () => setEditingId(s.id === editingId ? null : s.id), onDelete: () => remove(s.id), editingRow: s.id === editingId,
          }))}
          {serverTrigger && row({
            id: 'trig-server', key: 'server', on: !!serverTrigger.enabled, onToggle: v => updateServer({ enabled: v }),
            kind: t('trigKindServer'), what: serverSummary(t, language, serverTrigger), off: !serverTrigger.enabled,
            onEdit: () => setEditingId(editingId === 'server' ? null : 'server'), onDelete: removeServer, editingRow: editingId === 'server',
          })}
          {showEvents && row({
            id: 'trig-events', key: 'events', on: !!runAtStartup, onToggle: v => setRunAtStartup(v),
            kind: t('trigKindEvents'), what: t('trigEventsStarts'), off: !runAtStartup,
            onEdit: () => setEditingId(editingId === 'events' ? null : 'events'), onDelete: removeEvents, editingRow: editingId === 'events',
          })}
        </ul>
        <div className="sched-add-row" id="trig-add">
          <span>{t('trigAddLabel')}</span>
          <button type="button" className="btn btn-secondary" id="sched-add" onClick={() => add('days')}>{t('trigKindTimes')}</button>
          <button type="button" className="btn btn-secondary" id="trig-add-server" disabled={!!serverTrigger} onClick={addServer}>{t('trigKindServer')}</button>
          <button type="button" className="btn btn-secondary" id="trig-add-events" disabled={showEvents} onClick={addEvents}>{t('trigKindEvents')}</button>
        </div>
        {editing && editor(editing)}
        {editingId === 'server' && serverTrigger && serverEditor(serverTrigger)}
        {editingId === 'events' && eventsEditor()}
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
        {!anyOn
          ? <>{t('trigOnlyHand')}</>
          : <>
              {setNext && <>{t('schedSetNext', { time: fmt(setNext.time) })} ({describe(t, language, setNext.schedule)})</>}
              {setNext && enabledCount > 1 && <><br />{tp(t, language, 'schedCountOn', enabledCount)}</>}
              {serverOn && <>{setNext && <br />}<span id="schedule-summary-server">{t('trigKindServer')}: {serverSummary(t, language, serverTrigger)}</span></>}
              {runAtStartup && <>{(setNext || serverOn) && <br />}{t('andAtStartup')}</>}
              {setNext && wakeToRun && <><br />{t('andWakes')}</>}
              {batteryLine && <><br /><span id="schedule-summary-battery">{batteryLine}</span></>}
              {setNext && <><br />{t('schedCatchUp')}</>}
            </>}
      </div>
    </div>
  )
}
