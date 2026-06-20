import { useEffect, useMemo, useState } from 'react'
import { useOutletContext } from 'react-router-dom'
import { api, type APIKey, type APIKeyCreated, type CurrentUser } from '../api'
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

function ShieldIcon() {
  return <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round"><path d="M12 22s8-4 8-10V5l-8-3-8 3v7c0 6 8 10 8 10Z"/><path d="m9 12 2 2 4-4"/></svg>
}

export default function ApiKeys() {
  const { user } = useOutletContext<{ user: CurrentUser | null }>()
  const [keys, setKeys] = useState<APIKey[]>([])
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState('')
  const [showNew, setShowNew] = useState(false)
  const [newName, setNewName] = useState('')
  const [creating, setCreating] = useState(false)
  const [newKey, setNewKey] = useState<APIKeyCreated | null>(null)
  const [copied, setCopied] = useState(false)
  const [query, setQuery] = useState('')

  if (user?.role === 'viewer') {
    return (
      <div>
        <div className="page-header">
          <div>
            <div className="page-kicker">Authentication</div>
            <h1 className="page-title">API Keys</h1>
          </div>
        </div>
        <div className="empty-state">
          <h3>Access Denied</h3>
          <p>You don't have permission to manage API keys. Contact an administrator if you need access.</p>
        </div>
      </div>
    )
  }

  const load = () => {
    api.listAPIKeys()
      .then(k => setKeys(k ?? []))
      .catch(e => setError(e.message))
      .finally(() => setLoading(false))
  }

  useEffect(load, [])

  const visibleKeys = useMemo(() => {
    const term = query.trim().toLowerCase()
    return term ? keys.filter(key => key.name.toLowerCase().includes(term) || key.prefix.toLowerCase().includes(term)) : keys
  }, [keys, query])

  const usedKeys = keys.filter(key => key.last_used).length

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
    <div className="api-keys-page">
      <div className="page-header">
        <div>
          <div className="page-kicker">Authentication</div>
          <h1 className="page-title">API Keys</h1>
          <p className="page-sub">Secure credentials for CI pipelines, deployment tools, and automation.</p>
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

      {!loading && keys.length > 0 && <section className="key-overview" aria-label="API key overview">
        <div className="key-stat key-stat-primary"><span>Total keys</span><strong>{keys.length}</strong><small>Issued credentials</small></div>
        <div className="key-stat"><i className="used-dot" /><div><span>Used keys</span><strong>{usedKeys}</strong><small>Authenticated at least once</small></div></div>
        <div className="key-stat"><i className="unused-dot" /><div><span>Never used</span><strong>{keys.length - usedKeys}</strong><small>Awaiting first use</small></div></div>
      </section>}

      {/* New key reveal modal */}
      {newKey && (
        <div className="overlay" onClick={e => { if (e.target === e.currentTarget) { setNewKey(null) } }}>
          <div className="modal key-modal" role="dialog" aria-modal="true" aria-labelledby="key-created-title">
            <div className="modal-header">
              <div><div className="modal-kicker">Credential ready</div><h2 id="key-created-title">API key created</h2></div>
              <button className="modal-close" onClick={() => setNewKey(null)} aria-label="Close"><XIcon /></button>
            </div>
            <div className="ak-reveal">
              <div className="ak-reveal-warning">
                <span><AlertIcon /></span><div><strong>Copy this key now</strong><small>For your security, the secret will not be shown again.</small></div>
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
          <div className="modal key-modal" role="dialog" aria-modal="true" aria-labelledby="new-key-title">
            <div className="modal-header">
              <div><div className="modal-kicker">New credential</div><h2 id="new-key-title">Create API key</h2></div>
              <button className="modal-close" onClick={() => setShowNew(false)} aria-label="Close"><XIcon /></button>
            </div>
            <form onSubmit={handleCreate} className="form-stack">
              <div className="field">
                <label>Key name</label>
                <input
                  value={newName}
                  onChange={e => setNewName(e.target.value)}
                  placeholder="my-ci-pipeline"
                  required
                  autoFocus
                />
                <span className="field-hint">Use a descriptive label for the service or machine using this key.</span>
              </div>
              <div className="key-security-note"><ShieldIcon /><span>Store the generated secret in a secure credential manager.</span></div>
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
        <div className="empty-state api-empty">
          <div className="key-empty-visual"><span><KeyIcon /></span></div>
          <div className="empty-eyebrow">Secure automation starts here</div><h2>No API keys yet</h2>
          <p>Create a key to let CI pipelines and scripts authenticate without sharing your password.</p>
          <button className="primary" onClick={() => setShowNew(true)}><PlusIcon />Create API key</button>
        </div>
      ) : (
        <section className="keys-collection">
          <div className="keys-collection-header"><div><h2>Issued credentials</h2><p>{visibleKeys.length === keys.length ? `${keys.length} key${keys.length === 1 ? '' : 's'}` : `${visibleKeys.length} of ${keys.length} keys`}</p></div>
            <label className="key-search"><span className="search-symbol" aria-hidden="true" /><span className="sr-only">Search API keys</span><input value={query} onChange={e => setQuery(e.target.value)} placeholder="Search keys…" /></label>
          </div>
        <div className="package-table card keys-table">
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
              {visibleKeys.map(k => (
                <tr key={k.id}>
                  <td><div className="key-identity"><span><svg width="15" height="15" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round"><circle cx="7.5" cy="15.5" r="4.5"/><path d="m10.7 12.3 8.8-8.8M15 8l2 2M17.5 5.5l2 2"/></svg></span><div><strong>{k.name}</strong><small>{k.last_used ? 'Active credential' : 'Not used yet'}</small></div></div></td>
                  <td><code className="key-prefix">{k.prefix}…</code></td>
                  <td className="muted-text">{formatDate(k.created_at)}</td>
                  <td>{k.last_used ? <span className="key-used"><i />{formatDate(k.last_used)}</span> : <span className="key-never">Never</span>}</td>
                  <td>
                    <button className="danger icon-btn" onClick={() => handleDelete(k)} title="Delete key">
                      <TrashIcon />
                    </button>
                  </td>
                </tr>
              ))}
              {visibleKeys.length === 0 && <tr><td colSpan={5}><div className="keys-no-results"><strong>No keys found</strong><span>Try a different name or prefix.</span><button className="ghost" onClick={() => setQuery('')}>Clear search</button></div></td></tr>}
            </tbody>
          </table>
        </div>
        </section>
      )}
    </div>
  )
}

function formatDate(s: string) {
  return new Date(s).toLocaleDateString(undefined, { month: 'short', day: 'numeric', year: 'numeric' })
}
