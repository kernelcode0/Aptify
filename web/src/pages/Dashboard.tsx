import { useEffect, useState } from 'react'
import { Link, useNavigate } from 'react-router-dom'
import { api, getToken, type Repo } from '../api'
import './Dashboard.css'

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

  return (
    <div>
      <div className="page-header">
        <div>
          <h1>Repositories</h1>
          <p className="page-sub">Manage your APT package repositories</p>
        </div>
        <div style={{ display: 'flex', gap: '8px' }}>
          <button className="ghost" onClick={() => {
            const t = getToken();
            window.location.href = `/api/system/gpg-key` + (t ? `?token=${t}` : '');
          }}>Export GPG Key</button>
          <button className="primary" onClick={() => setShowNew(true)}>+ New Repository</button>
        </div>
      </div>

      {error && <div className="alert-error">{error}</div>}

      {showNew && (
        <div className="card new-repo-card">
          <h2>Create Repository</h2>
          <form onSubmit={handleCreate} className="new-repo-form">
            <div className="field">
              <label>Name</label>
              <input value={form.name} onChange={e => setForm(f => ({ ...f, name: e.target.value }))}
                placeholder="Ubuntu Developer Toolkit" required />
            </div>
            <div className="field">
              <label>Slug <span className="muted">(URL-safe identifier)</span></label>
              <input value={form.slug} onChange={e => setForm(f => ({ ...f, slug: e.target.value.toLowerCase() }))}
                placeholder="ubuntu-toolkit" pattern="[a-z0-9][a-z0-9\-]{0,62}" required />
            </div>
            <div className="field">
              <label>Codename</label>
              <input value={form.codename} onChange={e => setForm(f => ({ ...f, codename: e.target.value }))}
                placeholder="stable" required />
            </div>
            <div className="form-actions">
              <button type="submit" className="primary" disabled={creating}>
                {creating ? 'Creating…' : 'Create'}
              </button>
              <button type="button" className="ghost" onClick={() => setShowNew(false)}>Cancel</button>
            </div>
          </form>
        </div>
      )}

      {loading ? (
        <div style={{ display: 'flex', justifyContent: 'center', padding: '48px' }}>
          <div className="spinner" />
        </div>
      ) : repos.length === 0 ? (
        <div className="empty-state">
          <h3>No repositories yet</h3>
          <p>Create your first repository to get started.</p>
        </div>
      ) : (
        <div className="repo-grid">
          {repos.map(r => (
            <Link key={r.id} to={`/repos/${r.id}`} className="repo-card card">
              <div className="repo-card-top">
                <span className="repo-icon">📦</span>
                <span className="tag">{r.codename}</span>
              </div>
              <div className="repo-name">{r.name}</div>
              <div className="repo-slug muted">{r.slug}</div>
            </Link>
          ))}
        </div>
      )}
    </div>
  )
}
