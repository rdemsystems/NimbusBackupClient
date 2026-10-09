import { useState, useEffect, useCallback } from 'react'
import { useTranslation } from '../i18n/i18nContext'

// Wails bindings are resolved by App.jsx on window.go; this component receives
// them as props so it stays testable/usable when the runtime is absent (the dev
// server, or a plain browser) — every backend call is optional here.
function resolve(name, prop) {
  if (prop) return prop
  if (window.go && window.go.main && window.go.main.App) return window.go.main.App[name]
  return undefined
}

/**
 * EncryptionKeyField edits the path to a PBS encryption key file.
 *
 * The GUI only ever sees the PATH (never key material) and only supports
 * passphrase-less key files, because it has no console to prompt on. The
 * fingerprint is surfaced on every change so the user can confirm the key that
 * will actually be used, and an unusable path is reported inline instead of
 * failing later during a backup or a restore.
 */
export default function EncryptionKeyField({ value, onChange, className = '' }) {
  const { t, language } = useTranslation()
  const [info, setInfo] = useState(null)
  const [busy, setBusy] = useState(false)
  // Safekeeping copy (QR code / paper key) and import from text.
  const [copyPass, setCopyPass] = useState('')
  const [shownCopy, setShownCopy] = useState(null)
  const [importOpen, setImportOpen] = useState(false)
  const [importText, setImportText] = useState('')
  const [importPass, setImportPass] = useState('')

  const inspect = useCallback(resolve('InspectEncryptionKeyFile'), [])
  const generate = useCallback(resolve('GenerateEncryptionKeyFile'), [])
  const openExisting = useCallback(resolve('OpenEncryptionKeyDialog'), [])
  const openNew = useCallback(resolve('OpenEncryptionKeySaveDialog'), [])
  const getCopy = useCallback(resolve('GetEncryptionKeyCopy'), [])
  const exportPaper = useCallback(resolve('ExportEncryptionPaperKey'), [])
  const importText_ = useCallback(resolve('ImportEncryptionKeyText'), [])

  // A shown copy belongs to the key it was made from.
  useEffect(() => { setShownCopy(null) }, [value, copyPass])

  // Re-inspect whenever the path changes. inspect() is synchronous in Go and
  // cheap (one stat + a small JSON read), and doing it here means the user
  // always sees the fingerprint of the key that will be used.
  useEffect(() => {
    if (!inspect) {
      setInfo(null)
      return
    }
    // Wails bindings return a Promise: wait for it, and drop the answer if
    // the path changed meanwhile (the user is typing).
    let stale = false
    Promise.resolve()
      .then(() => inspect(value || ''))
      .then((res) => { if (!stale) setInfo(res || null) })
      .catch(() => { if (!stale) setInfo(null) })
    return () => { stale = true }
  }, [value, inspect])

  const handleBrowse = async () => {
    if (!openExisting) return
    try {
      const p = await openExisting()
      if (p) onChange(p)
    } catch (err) {
      console.error('encryption key browse failed', err)
    }
  }

  const handleGenerate = async () => {
    if (!generate || !openNew) return
    setBusy(true)
    try {
      let path = await openNew()
      if (!path) return
      const msg = t('encryptionKeyConfirm').replace('{path}', path)
      if (!window.confirm(msg)) return
      const result = await generate(path)
      onChange(path)
      // Trust the backend's own view: it only returns a usable key file.
      if (result) setInfo(result)
      window.alert(t('encryptionKeyCreated').replace('{path}', path).replace('{fp}', result?.fingerprint || '?'))
    } catch (err) {
      window.alert(t('encryptionKeyGenerateFailed').replace('{err}', err))
    } finally {
      setBusy(false)
    }
  }

  const handleToggleCopy = async () => {
    if (shownCopy) {
      setShownCopy(null)
      return
    }
    if (!getCopy) return
    try {
      setShownCopy(await getCopy(value || '', copyPass))
    } catch (err) {
      window.alert(`❌ ${err}`)
    }
  }

  // Printable copy (key text + QR code), like `proxmox-backup-client key paperkey`.
  const handlePrint = async () => {
    if (!exportPaper) return
    try {
      const out = await exportPaper(value || '', copyPass, {
        lang: language,
        title: t('encPaperTitle'),
        notes: [t('encPaperNoteRestore'), t('encPaperNotePBS'), copyPass ? t('encPaperNoteProtected') : t('encPaperNoteClear')],
      })
      if (out) window.alert(`✅ ${t('encPaperExported')} ${out}`)
    } catch (err) {
      window.alert(`❌ ${err}`)
    }
  }

  const handleImportText = async () => {
    if (!importText_ || !importText.trim()) return
    setBusy(true)
    try {
      const result = await importText_(importText, importPass)
      if (result && result.path) {
        onChange(result.path)
        setInfo(result)
        setImportText('')
        setImportPass('')
        setImportOpen(false)
        window.alert(t('encryptionKeyCreated').replace('{path}', result.path).replace('{fp}', result.fingerprint || '?'))
      }
    } catch (err) {
      window.alert(`❌ ${err}`)
    } finally {
      setBusy(false)
    }
  }

  const reason = info && !info.usable ? info.reason : ''
  const keyReadable = info && info.exists && info.fingerprint

  return (
    <div className={`form-group ${className}`.trim()}>
      <label>{t('encryptionKey')}</label>
      <div style={{ display: 'flex', gap: '8px', alignItems: 'stretch' }}>
        <input
          type="text"
          value={value || ''}
          onChange={(e) => onChange(e.target.value)}
          placeholder={t('phEncryptionKey')}
          spellCheck={false}
        />
        <button type="button" className="btn btn-secondary" onClick={handleBrowse} disabled={!openExisting || busy}>
          {t('encryptionKeyBrowse')}
        </button>
        <button type="button" className="btn btn-secondary" onClick={handleGenerate} disabled={!generate || busy}>
          {t('encryptionKeyGenerate')}
        </button>
        <button
          type="button"
          className="btn btn-secondary"
          onClick={() => onChange('')}
          disabled={busy || !(value || '')}
        >
          {t('encryptionKeyClear')}
        </button>
      </div>

      <div style={{ marginTop: '6px', fontSize: '12px', lineHeight: '1.4' }}>
        <div style={{ color: '#666' }}>{t('encryptionKeyHint')}</div>
        {!(value || '') && <div style={{ color: '#a06000' }}>{t('encryptionKeyUnset')}</div>}
        {info && info.usable && info.fingerprint && (
          <div style={{ color: '#1a7f37', wordBreak: 'break-all' }}>
            {t('encryptionKeyOk').replace('{fp}', info.fingerprint)}
          </div>
        )}
        {reason && (
          <div style={{ color: '#c62828', wordBreak: 'break-word' }}>{reason}</div>
        )}
      </div>

      {keyReadable && (
        <div style={{ marginTop: '10px' }}>
          <input
            type="password"
            value={copyPass}
            onChange={(e) => setCopyPass(e.target.value)}
            placeholder={t('encCopyPassphrase')}
          />
          <div style={{ display: 'flex', gap: '8px', marginTop: '6px' }}>
            <button type="button" className="btn btn-secondary" onClick={handlePrint} disabled={!exportPaper || busy}>
              🖨️ {t('encPaperExport')}
            </button>
            <button type="button" className="btn btn-secondary" onClick={handleToggleCopy} disabled={!getCopy || busy}>
              {shownCopy ? `🙈 ${t('encHide')}` : `👁️ ${t('encShow')}`}
            </button>
          </div>
          {shownCopy && (
            <div style={{ marginTop: '8px' }}>
              <textarea readOnly value={shownCopy.key} rows="8" style={{ fontFamily: 'monospace', fontSize: '0.8em', width: '100%' }} onFocus={(e) => e.target.select()} />
              {shownCopy.qr_svg && (
                <div style={{ textAlign: 'center', marginTop: '8px' }}>
                  <img src={`data:image/svg+xml;charset=utf-8,${encodeURIComponent(shownCopy.qr_svg)}`} alt={t('encQRAlt')} style={{ width: '220px', height: '220px' }} />
                  <div style={{ fontSize: '0.85em', color: '#64748b' }}>{t('encQRHint')}</div>
                </div>
              )}
            </div>
          )}
        </div>
      )}

      <div style={{ marginTop: '10px' }}>
        <button type="button" className="btn btn-secondary" onClick={() => setImportOpen(!importOpen)} disabled={!importText_ || busy}>
          📥 {t('encImportTitle')}
        </button>
        {importOpen && (
          <div style={{ marginTop: '6px' }}>
            <textarea value={importText} onChange={(e) => setImportText(e.target.value)} placeholder={t('encImportPlaceholder')} rows="5" style={{ fontFamily: 'monospace', fontSize: '0.85em', width: '100%' }} />
            <input type="password" value={importPass} onChange={(e) => setImportPass(e.target.value)} placeholder={t('encImportPassphrase')} style={{ marginTop: '6px' }} />
            <div className="info-box" style={{ marginTop: '6px', backgroundColor: '#fff3cd', borderColor: '#ffeeba' }}>
              ⚠️ {t('encImportUnprotectedWarning')}
            </div>
            <button type="button" className="btn" onClick={handleImportText} disabled={!importText.trim() || busy} style={{ marginTop: '6px' }}>
              {t('encImport')}
            </button>
          </div>
        )}
      </div>
    </div>
  )
}