import { useEffect, useState } from 'react'
import { api, type APIKey, type APIKeyCreated } from '../api'
import './ApiKeys.css'

function PlusIcon() {
  return (
    <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2.5" strokeLinecap="round">
      <line x1="12" y1="5" x2="12" y2="19"/><line x1="5" y1="12" x2="19" y2="12"/>
    </svg>
  )
}

function TrashIcon() {
  return (
    <svg width="13" height="13" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round">
      <polyline points="3 6 5 6 21 6"/>
      <path d="M19 6l-1 14a2 2 0 0 1-2 2H8a2 2 0 0 1-2-2L5 6"/>
      <path d="M10 11v6"/><path d="M14 11v6"/>
      <path d="M9 6V4a1 1 0 0 1 1-1h4a1 1 0 0 1 1 1v2"/>
    </svg>
  )
}

function XIcon() {
  return (
    <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round">
      <line x1="18" y1="6" x2="6" y2="18"/><line x1="6" y1="6" x2="18" y2="18"/>
    </svg>
  )
}

function AlertIcon() {
  return (
    <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round" style={{ flexShrink: 0 }}>
      <circle cx="12" cy="12" r="10"/>
      <line x1="12" y1="8" x2="12" y2="12"/>
      <line x1="12" y1="16" x2="12.01" y2="16"/>
    </svg>
  )
}

function KeyIcon() {
  return (
    <svg width="48" height="48" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="1.25" strokeLinecap="round" strokeLinejoin="round">
      <path d="M21 2l-2 2m-7.61 7.61a5.5 5.5 0 1 1-7.778 7.778 5.5 5.5 0 0 1 7.777-7.777zm0 0L15.5 7.5m0 0l3 3L22 7l-3-3m-3.5 3.5L19 4"/>
    </svg>
  )
}

export default function ApiKeys() {
  const [keys, setKeys] = useState<APIKey[]>([])
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState('')
  const [showNew, setShowNew] = useState(false)
  const [newName, setNewName] = useState('')
  const [creating, setCreating] = useState(false)
  const [newKey, setNewKey] = useState<APIKeyCreated | null>(null)
  const [copied, setCopied] = useState(false)

  const load = () => {
    api.listAPIKeys()
      .then(k => setKeys(k ?? []))
      .catch(e => setError(e.message))
      .finally(() => setLoading(false))
  }

  useEffect(load, [])

  const handleCreate = async (e: React.FormEvent) => {
    e.preventDefault()
    if (!newName.trim()) return
    setCreating(true)
    setError('')
    try {
      const created = await api.createAPIKey(newName.trim())
      setNewKey(created)
      setShowNew(false)
      setNewName('')
      setKeys(prev => [...prev, created])
    } catch (e: unknown) {
      setError(e instanceof Error ? e.message : 'failed to create key')
    } finally {
      setCreating(false)
    }
  }

  const handleDelete = async (key: APIKey) => {
    if (!confirm(`Delete API key "${key.name}"? This cannot be undone.`)) return
    try {
      await api.deleteAPIKey(key.id)
      setKeys(prev => prev.filter(k => k.id !== key.id))
    } catch (e: unknown) {
      setError(e instanceof Error ? e.message : 'failed to delete key')
    }
  }

  const copyKey = () => {
    if (!newKey) return
    const handleSuccess = () => { setCopied(true); setTimeout(() => setCopied(false), 2000) }
    if (navigator.clipboard && window.isSecureContext) {
      navigator.clipboard.writeText(newKey.key).then(handleSuccess)
    } else {
      const ta = document.createElement('textarea')
      ta.value = newKey.key
      ta.style.position = 'fixed'; ta.style.left = '-9999px'
      document.body.appendChild(ta); ta.select()
      document.execCommand('copy')
      ta.remove()
      handleSuccess()
    }
  }

  return (
    <div>
      <div className="page-header">
        <div>
          <div className="page-kicker">Authentication</div>
          <h1 className="page-title">API Keys</h1>
          <p className="page-sub">{keys.length} key{keys.length !== 1 ? 's' : ''}</p>
        </div>
        <div className="page-actions">
          <button className="primary" onClick={() => setShowNew(true)}>
            <PlusIcon />
            New API Key
          </button>
        </div>
      </div>

      {error && (
        <div className="alert-error">
          <AlertIcon />
          {error}
        </div>
      )}

      {/* New key reveal modal */}
      {newKey && (
        <div className="overlay" onClick={e => { if (e.target === e.currentTarget) { setNewKey(null) } }}>
          <div className="modal">
            <div className="modal-header">
              <h2>API Key Created</h2>
              <button className="modal-close" onClick={() => setNewKey(null)} aria-label="Close"><XIcon /></button>
            </div>
            <div className="ak-reveal">
              <div className="ak-reveal-warning">
                <AlertIcon />
                This key will not be shown again. Copy it now.
              </div>
              <div className="ak-reveal-key">
                <code>{newKey.key}</code>
                <button className="copy-btn" onClick={copyKey}>{copied ? '✓ Copied' : 'Copy'}</button>
              </div>
              <p className="ak-reveal-meta">
                Name: <strong>{newKey.name}</strong> &nbsp;·&nbsp; Prefix: <code>{newKey.prefix}</code>
              </p>
            </div>
            <div className="form-actions">
              <button className="primary" onClick={() => setNewKey(null)}>Done</button>
            </div>
          </div>
        </div>
      )}

      {/* Create key modal */}
      {showNew && (
        <div className="overlay" onClick={e => { if (e.target === e.currentTarget) setShowNew(false) }}>
          <div className="modal">
            <div className="modal-header">
              <h2>New API Key</h2>
              <button className="modal-close" onClick={() => setShowNew(false)} aria-label="Close"><XIcon /></button>
            </div>
            <form onSubmit={handleCreate} className="form-stack">
              <div className="field">
                <label>Name</label>
                <input
                  value={newName}
                  onChange={e => setNewName(e.target.value)}
                  placeholder="my-ci-pipeline"
                  required
                  autoFocus
                />
                <span className="field-hint">A label to identify this key (e.g. the system it's used on)</span>
              </div>
              <div className="form-actions">
                <button type="submit" className="primary" disabled={creating}>
                  {creating
                    ? <><div className="spinner" style={{ width: 14, height: 14, borderWidth: '2px' }} />Creating…</>
                    : 'Create Key'
                  }
                </button>
                <button type="button" className="ghost" onClick={() => setShowNew(false)}>Cancel</button>
              </div>
            </form>
          </div>
        </div>
      )}

      {loading ? (
        <div className="loading-center"><div className="spinner" /></div>
      ) : keys.length === 0 ? (
        <div className="empty-state">
          <div className="empty-icon"><KeyIcon /></div>
          <h3>No API keys yet</h3>
          <p>Create a key to let CI pipelines and scripts authenticate without a password.</p>
        </div>
      ) : (
        <div className="package-table card">
          <table>
            <thead>
              <tr>
                <th>Name</th>
                <th>Prefix</th>
                <th>Created</th>
                <th>Last used</th>
                <th style={{ width: 60 }}></th>
              </tr>
            </thead>
            <tbody>
              {keys.map(k => (
                <tr key={k.id}>
                  <td><span className="pkg-name">{k.name}</span></td>
                  <td><code className="pkg-version">{k.prefix}…</code></td>
                  <td className="muted-text">{formatDate(k.created_at)}</td>
                  <td className="muted-text">{k.last_used ? formatDate(k.last_used) : '—'}</td>
                  <td>
                    <button className="danger icon-btn" onClick={() => handleDelete(k)} title="Delete key">
                      <TrashIcon />
                    </button>
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      )}
    </div>
  )
}

function formatDate(s: string) {
  return new Date(s).toLocaleDateString(undefined, { month: 'short', day: 'numeric', year: 'numeric' })
}
