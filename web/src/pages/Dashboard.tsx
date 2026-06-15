import { useEffect, useState } from 'react'
import { Link, useNavigate } from 'react-router-dom'
import { api, getToken, type Repo } from '../api'
import './Dashboard.css'

function PlusIcon() {
  return (
    <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2.5" strokeLinecap="round">
      <line x1="12" y1="5" x2="12" y2="19"/><line x1="5" y1="12" x2="19" y2="12"/>
    </svg>
  )
}

function KeyIcon() {
  return (
    <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round">
      <path d="M21 2l-2 2m-7.61 7.61a5.5 5.5 0 1 1-7.778 7.778 5.5 5.5 0 0 1 7.777-7.777zm0 0L15.5 7.5m0 0l3 3L22 7l-3-3m-3.5 3.5L19 4"/>
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

function RepoIcon() {
  return (
    <svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="1.75" strokeLinecap="round" strokeLinejoin="round">
      <polyline points="21 8 21 21 3 21 3 8"/>
      <rect x="1" y="3" width="22" height="5"/>
      <line x1="10" y1="12" x2="14" y2="12"/>
    </svg>
  )
}

function EmptyBoxIcon() {
  return (
    <svg width="48" height="48" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="1.25" strokeLinecap="round" strokeLinejoin="round">
      <polyline points="21 8 21 21 3 21 3 8"/>
      <rect x="1" y="3" width="22" height="5"/>
      <line x1="10" y1="12" x2="14" y2="12"/>
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

export default function Dashboard() {
  const [repos, setRepos] = useState<Repo[]>([])
  const [loading, setLoading] = useState(true)
  const [showNew, setShowNew] = useState(false)
  const [form, setForm] = useState({ slug: '', name: '', codename: 'stable' })
  const [creating, setCreating] = useState(false)
  const [error, setError] = useState('')
  const navigate = useNavigate()

  const load = () => {
    api.listRepos()
      .then(setRepos)
      .catch(e => setError(e.message))
      .finally(() => setLoading(false))
  }

  useEffect(load, [])

  const handleCreate = async (e: React.FormEvent) => {
    e.preventDefault()
    setCreating(true)
    setError('')
    try {
      const repo = await api.createRepo(form.slug, form.name, form.codename)
      setShowNew(false)
      setForm({ slug: '', name: '', codename: 'stable' })
      navigate(`/repos/${repo.id}`)
    } catch (e: unknown) {
      setError(e instanceof Error ? e.message : 'unknown error')
    } finally {
      setCreating(false)
    }
  }

  const handleExportKey = async () => {
    try {
      const res = await fetch('/api/system/gpg-key', {
        headers: { Authorization: `Bearer ${getToken() || ''}` },
      })
      if (!res.ok) throw new Error('Failed to export GPG key')
      const blob = await res.blob()
      const url = window.URL.createObjectURL(blob)
      const a = document.createElement('a')
      a.href = url
      a.download = 'signing-key.asc'
      document.body.appendChild(a)
      a.click()
      window.URL.revokeObjectURL(url)
      document.body.removeChild(a)
    } catch (err: unknown) {
      setError(err instanceof Error ? err.message : 'Export failed')
    }
  }

  return (
    <div>
      <div className="page-header">
        <div>
          <div className="page-kicker">Control plane</div>
          <h1 className="page-title">Repositories</h1>
          <p className="page-sub">{repos.length} active repositor{repos.length === 1 ? 'y' : 'ies'}</p>
        </div>
        <div className="page-actions">
          <button className="ghost" onClick={handleExportKey}>
            <KeyIcon />
            Export GPG Key
          </button>
          <button className="primary" onClick={() => setShowNew(true)}>
            <PlusIcon />
            New Repository
          </button>
        </div>
      </div>

      {error && (
        <div className="alert-error">
          <AlertIcon />
          {error}
        </div>
      )}

      {showNew && (
        <div className="overlay" onClick={e => { if (e.target === e.currentTarget) setShowNew(false) }}>
          <div className="modal">
            <div className="modal-header">
              <h2>Create Repository</h2>
              <button className="modal-close" onClick={() => setShowNew(false)} aria-label="Close">
                <XIcon />
              </button>
            </div>
            <form onSubmit={handleCreate} className="form-stack">
              <div className="field">
                <label>Name</label>
                <input
                  value={form.name}
                  onChange={e => setForm(f => ({ ...f, name: e.target.value }))}
                  placeholder="Ubuntu Developer Toolkit"
                  required
                  autoFocus
                />
              </div>
              <div className="field">
                <label>Slug</label>
                <input
                  value={form.slug}
                  onChange={e => setForm(f => ({ ...f, slug: e.target.value.toLowerCase() }))}
                  placeholder="ubuntu-toolkit"
                  pattern="[a-z0-9][a-z0-9\-]{0,62}"
                  required
                />
                <span className="field-hint">Lowercase letters, numbers, and hyphens only</span>
              </div>
              <div className="field">
                <label>Codename</label>
                <input
                  value={form.codename}
                  onChange={e => setForm(f => ({ ...f, codename: e.target.value }))}
                  placeholder="stable"
                  required
                />
              </div>
              <div className="form-actions">
                <button type="submit" className="primary" disabled={creating}>
                  {creating
                    ? <><div className="spinner" style={{ width: 14, height: 14, borderWidth: '2px' }} />Creating…</>
                    : 'Create Repository'
                  }
                </button>
                <button type="button" className="ghost" onClick={() => setShowNew(false)}>Cancel</button>
              </div>
            </form>
          </div>
        </div>
      )}

      {loading ? (
        <div className="loading-center">
          <div className="spinner" />
        </div>
      ) : repos.length === 0 ? (
        <div className="empty-state">
          <div className="empty-icon"><EmptyBoxIcon /></div>
          <h3>No repositories yet</h3>
          <p>Create your first repository to get started.</p>
        </div>
      ) : (
        <>
        <div className="dashboard-strip">
          <div>
            <span className="strip-label">Repositories</span>
            <strong>{repos.length}</strong>
          </div>
          <div>
            <span className="strip-label">Default suite</span>
            <strong>{repos.find(r => r.codename === 'stable') ? 'stable' : repos[0]?.codename ?? 'none'}</strong>
          </div>
          <div>
            <span className="strip-label">Signing key</span>
            <strong>available</strong>
          </div>
        </div>
        <div className="repo-grid">
          {repos.map(r => (
            <Link key={r.id} to={`/repos/${r.id}`} className="repo-card card">
              <div className="repo-card-header">
                <span className="repo-card-icon"><RepoIcon /></span>
                <span className="tag">{r.codename}</span>
              </div>
              <div className="repo-card-body">
                <div className="repo-name">{r.name}</div>
                <div className="repo-slug">/{r.slug}</div>
              </div>
              <div className="repo-card-footer">
                <span className="repo-date">{formatDate(r.created_at)}</span>
              </div>
            </Link>
          ))}
        </div>
        </>
      )}
    </div>
  )
}

function formatDate(s: string) {
  return new Date(s).toLocaleDateString(undefined, { month: 'short', day: 'numeric', year: 'numeric' })
}
