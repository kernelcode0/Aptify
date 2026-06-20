import { useEffect, useState } from 'react'
import { useOutletContext } from 'react-router-dom'
import { api, type AuditEntry, type CurrentUser } from '../api'
import './AuditLog.css'

function AlertIcon() {
  return (
    <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round" style={{ flexShrink: 0 }}>
      <circle cx="12" cy="12" r="10"/>
      <line x1="12" y1="8" x2="12" y2="12"/>
      <line x1="12" y1="16" x2="12.01" y2="16"/>
    </svg>
  )
}

function DownloadIcon() {
  return (
    <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round">
      <path d="M12 3v12m0 0 5-5m-5 5-5-5"/><path d="M5 21h14"/>
    </svg>
  )
}

function TrashIcon() {
  return <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round"><path d="M3 6h18M8 6V4h8v2M19 6l-1 14H6L5 6M10 11v5M14 11v5"/></svg>
}

function XIcon() {
  return <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round"><path d="m18 6-12 12M6 6l12 12"/></svg>
}

export default function AuditLog() {
  const { user } = useOutletContext<{ user: CurrentUser | null }>()
  const [entries, setEntries] = useState<AuditEntry[]>([])
  const [total, setTotal] = useState(0)
  const [offset, setOffset] = useState(0)
  const [limit, setLimit] = useState(50)
  const [loading, setLoading] = useState(true)
  const [exporting, setExporting] = useState(false)
  const [clearing, setClearing] = useState(false)
  const [showClear, setShowClear] = useState(false)
  const [error, setError] = useState('')

  useEffect(() => {
    setLoading(true)
    api.listAudit(offset, limit)
      .then(data => {
        setEntries(data.entries ?? [])
        setTotal(data.total)
      })
      .catch(e => setError(e.message))
      .finally(() => setLoading(false))
  }, [offset, limit])

  const handleLimitChange = (value: number) => {
    setLimit(value)
    setOffset(0)
  }

  const exportAudit = async (): Promise<boolean> => {
    setExporting(true)
    setError('')
    try {
      const exportLimit = 200
      let exportOffset = 0
      let exportTotal = 0
      const allEntries: AuditEntry[] = []

      do {
        const data = await api.listAudit(exportOffset, exportLimit)
        const batch = data.entries ?? []
        exportTotal = data.total
        allEntries.push(...batch)
        exportOffset += batch.length
        if (batch.length === 0) break
      } while (allEntries.length < exportTotal)

      const rows = [
        ['ID', 'User', 'Action', 'Resource', 'Detail', 'Timestamp'],
        ...allEntries.map(entry => [entry.id, entry.username, entry.action, entry.resource, entry.detail || '', entry.created_at]),
      ]
      const csv = rows.map(row => row.map(toCsvCell).join(',')).join('\r\n')
      const blob = new Blob(['\uFEFF', csv], { type: 'text/csv;charset=utf-8' })
      const url = URL.createObjectURL(blob)
      const link = document.createElement('a')
      link.href = url
      link.download = `aptify-audit-log-${new Date().toISOString().slice(0, 10)}.csv`
      document.body.appendChild(link)
      link.click()
      link.remove()
      URL.revokeObjectURL(url)
      return true
    } catch (e: unknown) {
      setError(e instanceof Error ? e.message : 'Failed to export audit log')
      return false
    } finally {
      setExporting(false)
    }
  }

  const handleClear = async (exportFirst: boolean) => {
    setClearing(true)
    setError('')
    try {
      if (exportFirst && !(await exportAudit())) return
      await api.clearAudit()
      setEntries([])
      setTotal(0)
      setOffset(0)
      setShowClear(false)
    } catch (e: unknown) {
      setError(e instanceof Error ? e.message : 'Failed to clear audit log')
    } finally {
      setClearing(false)
    }
  }

  return (
    <div className="audit-page">
      <div className="page-header">
        <div>
          <div className="page-kicker">Operations</div>
          <h1 className="page-title">Audit Log</h1>
          <p className="page-sub">Trace security events and administrative activity across your workspace.</p>
        </div>
        <div className="page-actions">
          <button className="ghost" onClick={exportAudit} disabled={exporting || clearing || total === 0}>
            {exporting ? <><span className="spinner audit-export-spinner" />Exporting…</> : <><DownloadIcon />Export CSV</>}
          </button>
          {user?.role === 'admin' && <button className="danger clear-audit-trigger" onClick={() => setShowClear(true)} disabled={clearing || total === 0}><TrashIcon />Clear logs</button>}
        </div>
      </div>

      {error && <div className="alert-error"><AlertIcon />{error}</div>}

      {showClear && <div className="overlay" onClick={e => { if (e.target === e.currentTarget && !clearing) setShowClear(false) }}>
        <div className="modal clear-audit-modal" role="alertdialog" aria-modal="true" aria-labelledby="clear-audit-title" aria-describedby="clear-audit-description">
          <div className="modal-header"><div><div className="modal-kicker clear-kicker">Permanent action</div><h2 id="clear-audit-title">Clear all audit logs?</h2></div><button className="modal-close" onClick={() => setShowClear(false)} disabled={clearing} aria-label="Close"><XIcon /></button></div>
          <div className="clear-warning-icon"><TrashIcon /></div>
          <p id="clear-audit-description" className="clear-audit-copy">This will permanently delete all <strong>{total} audit event{total === 1 ? '' : 's'}</strong>. This history cannot be recovered after it is cleared.</p>
          <div className="export-recommendation"><DownloadIcon /><div><strong>Download a copy first?</strong><span>We recommend exporting the current logs for your security records.</span></div></div>
          <div className="clear-audit-actions">
            <button className="primary" onClick={() => handleClear(true)} disabled={clearing}><DownloadIcon />{clearing && exporting ? 'Exporting…' : clearing ? 'Clearing…' : 'Export & clear'}</button>
            <button className="danger danger-bordered" onClick={() => handleClear(false)} disabled={clearing}><TrashIcon />Clear without export</button>
            <button className="ghost" onClick={() => setShowClear(false)} disabled={clearing}>Cancel</button>
          </div>
        </div>
      </div>}

      {!loading && <section className="audit-overview" aria-label="Audit overview">
        <div className="audit-stat audit-stat-primary"><span>Total events</span><strong>{total}</strong><small>Recorded activity</small></div>
        <div className="audit-stat"><span>Visible range</span><strong>{total === 0 ? '0' : `${offset + 1}–${Math.min(offset + entries.length, total)}`}</strong><small>of {total} events</small></div>
        <div className="audit-stat"><span>Page size</span><strong>{limit}</strong><small>Events per page</small></div>
      </section>}

      {loading ? (
        <div className="loading-center"><div className="spinner" /></div>
      ) : (
        <section className="audit-collection">
        <div className="audit-collection-heading"><div><h2>Event history</h2><p>Newest activity appears first</p></div><span className="audit-export-note">CSV export includes all {total} events</span></div>
        <div className="package-table card audit-table">
          <table>
            <thead>
              <tr>
                <th>User</th>
                <th>Action</th>
                <th>Resource</th>
                <th>Detail</th>
                <th>Time</th>
              </tr>
            </thead>
            <tbody>
              {entries.map(e => (
                <tr key={e.id}>
                  <td><div className="audit-user"><span>{e.username.slice(0, 2).toUpperCase()}</span><strong>{e.username}</strong></div></td>
                  <td><span className={`audit-action action-${actionTone(e.action)}`}>{formatAction(e.action)}</span></td>
                  <td><code className="pkg-version">{e.resource}</code></td>
                  <td><Detail value={e.detail} /></td>
                  <td className="muted-text">{formatDateTime(e.created_at)}</td>
                </tr>
              ))}
              {entries.length === 0 && (
                <tr>
                  <td colSpan={5} className="muted-text">No audit entries yet.</td>
                </tr>
              )}
            </tbody>
          </table>
          <div className="pagination audit-pagination">
            <label className="page-size">
              <span>Rows per page</span>
              <select value={limit} onChange={e => handleLimitChange(Number(e.target.value))}>
                {[20, 30, 50, 100].map(size => <option key={size} value={size}>{size}</option>)}
              </select>
            </label>
            <div className="page-status">
              <span className="muted-text">{total === 0 ? '0 entries' : `${offset + 1}–${Math.min(offset + entries.length, total)} of ${total}`}</span>
              <span className="page-number">{total === 0 ? 'Page 0 of 0' : `Page ${Math.floor(offset / limit) + 1} of ${Math.ceil(total / limit)}`}</span>
            </div>
            <div className="page-buttons">
              <button className="ghost" disabled={offset === 0} onClick={() => setOffset(v => Math.max(0, v - limit))}>Previous</button>
              <button className="ghost" disabled={offset + limit >= total} onClick={() => setOffset(v => v + limit)}>Next</button>
            </div>
          </div>
        </div>
        </section>
      )}
    </div>
  )
}

function Detail({ value }: { value: string }) {
  if (!value || value === '{}') return <span className="muted-text">-</span>
  try {
    const parsed = JSON.parse(value)
    if (typeof parsed === 'object' && parsed !== null && Object.keys(parsed).length > 0) {
      return (
        <div className="audit-detail-kv">
          {Object.entries(parsed).map(([k, v]) => (
            <div key={k} className="kv-row">
              <span className="kv-key">{k}:</span>
              <span className="kv-val">{typeof v === 'object' ? JSON.stringify(v) : String(v)}</span>
            </div>
          ))}
        </div>
      )
    }
    return <pre className="audit-detail">{JSON.stringify(parsed, null, 2)}</pre>
  } catch {
    return <span className="audit-raw">{value}</span>
  }
}

function formatAction(action: string) {
  return action.split('_').join(' ')
}

function actionTone(action: string) {
  if (action.includes('delete') || action.includes('remove')) return 'danger'
  if (action.includes('create') || action.includes('upload') || action.includes('add')) return 'create'
  if (action.includes('login')) return 'auth'
  if (action.includes('update') || action.includes('edit')) return 'update'
  return 'neutral'
}

function formatDateTime(s: string) {
  return new Date(s).toLocaleString(undefined, {
    month: 'short',
    day: 'numeric',
    year: 'numeric',
    hour: '2-digit',
    minute: '2-digit',
  })
}

function toCsvCell(value: string) {
  const safe = /^[=+\-@]/.test(value) ? `'${value}` : value
  return `"${safe.replace(/"/g, '""')}"`
}
