import { useEffect, useState } from 'react'
import { api, type AuditEntry } from '../api'
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

export default function AuditLog() {
  const [entries, setEntries] = useState<AuditEntry[]>([])
  const [total, setTotal] = useState(0)
  const [offset, setOffset] = useState(0)
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState('')
  const limit = 50

  useEffect(() => {
    setLoading(true)
    api.listAudit(offset, limit)
      .then(data => {
        setEntries(data.entries ?? [])
        setTotal(data.total)
      })
      .catch(e => setError(e.message))
      .finally(() => setLoading(false))
  }, [offset])

  return (
    <div>
      <div className="page-header">
        <div>
          <div className="page-kicker">Operations</div>
          <h1 className="page-title">Audit Log</h1>
          <p className="page-sub">{total} entr{total === 1 ? 'y' : 'ies'}</p>
        </div>
      </div>

      {error && <div className="alert-error"><AlertIcon />{error}</div>}

      {loading ? (
        <div className="loading-center"><div className="spinner" /></div>
      ) : (
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
                  <td><span className="pkg-name">{e.username}</span></td>
                  <td><span className="tag">{formatAction(e.action)}</span></td>
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
          <div className="pagination">
            <button className="ghost" disabled={offset === 0} onClick={() => setOffset(v => Math.max(0, v - limit))}>Previous</button>
            <span className="muted-text">{total === 0 ? 'Page 0 of 0' : `Page ${Math.floor(offset / limit) + 1} of ${Math.ceil(total / limit)}`}</span>
            <button className="ghost" disabled={offset + limit >= total} onClick={() => setOffset(v => v + limit)}>Next</button>
          </div>
        </div>
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

function formatDateTime(s: string) {
  return new Date(s).toLocaleString(undefined, {
    month: 'short',
    day: 'numeric',
    year: 'numeric',
    hour: '2-digit',
    minute: '2-digit',
  })
}
