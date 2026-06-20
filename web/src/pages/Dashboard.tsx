import { useEffect, useMemo, useState } from 'react'
import { Link, useNavigate, useOutletContext } from 'react-router-dom'
import { api, getToken, type CurrentUser, type Repo } from '../api'
import './Dashboard.css'

type RepoFilter = 'all' | Repo['type']
type RepoSort = 'newest' | 'oldest' | 'name'

function Icon({ name, size = 16 }: { name: 'plus' | 'key' | 'x' | 'repo' | 'search' | 'arrow' | 'alert' | 'spark'; size?: number }) {
  const paths = {
    plus: <><path d="M12 5v14M5 12h14" /></>,
    key: <><circle cx="7.5" cy="15.5" r="4.5" /><path d="m10.7 12.3 8.8-8.8M15 8l2 2M17.5 5.5l2 2" /></>,
    x: <><path d="m18 6-12 12M6 6l12 12" /></>,
    repo: <><path d="M3 8h18v13H3zM1 3h22v5H1zM10 13h4" /></>,
    search: <><circle cx="11" cy="11" r="7" /><path d="m20 20-4-4" /></>,
    arrow: <><path d="M5 12h14M14 7l5 5-5 5" /></>,
    alert: <><circle cx="12" cy="12" r="10" /><path d="M12 8v4M12 16h.01" /></>,
    spark: <><path d="m12 3 1.4 4.1L17.5 8.5l-4.1 1.4L12 14l-1.4-4.1-4.1-1.4 4.1-1.4L12 3ZM18.5 15l.7 2.3 2.3.7-2.3.7-.7 2.3-.7-2.3-2.3-.7 2.3-.7.7-2.3Z" /></>,
  }
  return <svg width={size} height={size} viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round" aria-hidden="true">{paths[name]}</svg>
}

export default function Dashboard() {
  const { user } = useOutletContext<{ user: CurrentUser | null }>()
  const [repos, setRepos] = useState<Repo[]>([])
  const [loading, setLoading] = useState(true)
  const [showNew, setShowNew] = useState(false)
  const [form, setForm] = useState<{ slug: string; name: string; codename: string; type: Repo['type'] }>({ slug: '', name: '', codename: 'stable', type: 'deb' })
  const [creating, setCreating] = useState(false)
  const [error, setError] = useState('')
  const [createError, setCreateError] = useState('')
  const [query, setQuery] = useState('')
  const [filter, setFilter] = useState<RepoFilter>('all')
  const [sort, setSort] = useState<RepoSort>('newest')
  const navigate = useNavigate()

  const load = () => {
    setLoading(true)
    api.listRepos().then(setRepos).catch(e => setError(e.message)).finally(() => setLoading(false))
  }
  useEffect(load, [])

  const visibleRepos = useMemo(() => {
    const term = query.trim().toLowerCase()
    return repos
      .filter(repo => filter === 'all' || repo.type === filter)
      .filter(repo => !term || [repo.name, repo.slug, repo.codename, repo.type].some(value => value.toLowerCase().includes(term)))
      .sort((a, b) => sort === 'name'
        ? a.name.localeCompare(b.name)
        : sort === 'oldest'
          ? +new Date(a.created_at) - +new Date(b.created_at)
          : +new Date(b.created_at) - +new Date(a.created_at))
  }, [repos, query, filter, sort])

  const debCount = repos.filter(repo => repo.type === 'deb').length
  const rpmCount = repos.length - debCount

  const handleCreate = async (e: React.FormEvent) => {
    e.preventDefault()
    setCreating(true)
    setCreateError('')
    try {
      const repo = await api.createRepo(form.slug, form.name, form.codename, form.type)
      setShowNew(false)
      setForm({ slug: '', name: '', codename: 'stable', type: 'deb' })
      navigate(`/repos/${repo.id}`)
    } catch (e: unknown) {
      setCreateError(e instanceof Error ? e.message : 'Unknown error')
    } finally {
      setCreating(false)
    }
  }

  const handleExportKey = async () => {
    try {
      const res = await fetch('/api/system/gpg-key', { headers: { Authorization: `Bearer ${getToken() || ''}` } })
      if (!res.ok) throw new Error('Failed to export GPG key')
      const url = window.URL.createObjectURL(await res.blob())
      const a = document.createElement('a')
      a.href = url
      a.download = 'signing-key.asc'
      document.body.appendChild(a)
      a.click()
      window.URL.revokeObjectURL(url)
      a.remove()
    } catch (err: unknown) {
      setError(err instanceof Error ? err.message : 'Export failed')
    }
  }

  return (
    <div className="dashboard">
      <header className="dashboard-hero">
        <div className="dashboard-heading">
          <div className="page-kicker"><Icon name="spark" size={13} /> Control plane</div>
          <h1 className="page-title">Repositories</h1>
          <p className="page-sub">Manage package sources, distribution suites, and signing from one place.</p>
        </div>
        {user?.role === 'admin' && <button className="primary hero-primary" onClick={() => { setCreateError(''); setShowNew(true) }}><Icon name="plus" />New repository</button>}
      </header>

      {error && <div className="alert-error" role="alert"><Icon name="alert" /> <span>{error}</span><button onClick={() => setError('')} aria-label="Dismiss error"><Icon name="x" size={14} /></button></div>}

      {showNew && (
        <div className="overlay" onClick={e => { if (e.target === e.currentTarget) setShowNew(false) }}>
          <div className="modal" role="dialog" aria-modal="true" aria-labelledby="create-repo-title">
            <div className="modal-header"><div><div className="modal-kicker">New package source</div><h2 id="create-repo-title">Create repository</h2></div><button className="modal-close" onClick={() => setShowNew(false)} aria-label="Close"><Icon name="x" /></button></div>
            <form onSubmit={handleCreate} className="form-stack">
              {createError && <div className="alert-error" role="alert"><Icon name="alert" /> <span>{createError}</span></div>}
              <div className="field"><label htmlFor="repo-type">Package format</label><div className="type-picker">
                {(['deb', 'rpm'] as const).map(type => <button key={type} type="button" className={form.type === type ? 'selected' : ''} onClick={() => setForm(f => ({ ...f, type }))}><strong>{type.toUpperCase()}</strong><span>{type === 'deb' ? 'Debian / Ubuntu' : 'Red Hat / Fedora'}</span></button>)}
              </div></div>
              <div className="field"><label htmlFor="repo-name">Repository name</label><input id="repo-name" value={form.name} onChange={e => setForm(f => ({ ...f, name: e.target.value }))} placeholder={form.type === 'rpm' ? 'Fedora Developer Toolkit' : 'Ubuntu Developer Toolkit'} required autoFocus /></div>
              <div className="field"><label htmlFor="repo-slug">URL slug</label><div className="input-prefix"><span>/</span><input id="repo-slug" value={form.slug} onChange={e => setForm(f => ({ ...f, slug: e.target.value.toLowerCase() }))} placeholder="developer-toolkit" pattern="[a-z0-9][a-z0-9\-]{0,62}" required /></div><span className="field-hint">Lowercase letters, numbers, and hyphens only.</span></div>
              {form.type === 'deb' && <div className="field"><label htmlFor="repo-codename">Distribution suite</label><input id="repo-codename" value={form.codename} onChange={e => setForm(f => ({ ...f, codename: e.target.value }))} placeholder="stable" required /><span className="field-hint">For example: stable, bookworm, or noble.</span></div>}
              <div className="form-actions"><button type="button" className="ghost" onClick={() => setShowNew(false)}>Cancel</button><button type="submit" className="primary" disabled={creating}>{creating ? <><span className="spinner spinner-small" />Creating…</> : <>Create repository<Icon name="arrow" size={14} /></>}</button></div>
            </form>
          </div>
        </div>
      )}

      {loading ? <DashboardSkeleton /> : repos.length === 0 ? (
        <section className="empty-state dashboard-empty">
          <div className="empty-visual"><span className="empty-orbit" /><span className="empty-repo"><Icon name="repo" size={34} /></span></div>
          <div className="empty-eyebrow">Your package hub starts here</div><h2>Create your first repository</h2><p>Host and distribute DEB or RPM packages securely from your own infrastructure.</p>
          {user?.role === 'admin' && <button className="primary" onClick={() => setShowNew(true)}><Icon name="plus" />Create repository</button>}
        </section>
      ) : <>
        <section className="metrics" aria-label="Repository overview">
          <div className="metric metric-featured"><span className="metric-label">Total repositories</span><strong>{repos.length}</strong><span className="metric-note">Package sources online</span></div>
          <div className="metric"><span className="metric-dot metric-dot-deb" /><div><span className="metric-label">DEB repositories</span><strong>{debCount}</strong></div></div>
          <div className="metric"><span className="metric-dot metric-dot-rpm" /><div><span className="metric-label">RPM repositories</span><strong>{rpmCount}</strong></div></div>
        </section>

        <section className="collection">
          <div className="collection-heading"><div><h2>All repositories</h2><p>{visibleRepos.length === repos.length ? `${repos.length} package source${repos.length === 1 ? '' : 's'}` : `${visibleRepos.length} of ${repos.length} repositories`}</p></div><button className="ghost key-action" onClick={handleExportKey}><Icon name="key" />Export signing key</button></div>
          <div className="repo-toolbar">
            <label className="search-box"><Icon name="search" /><span className="sr-only">Search repositories</span><input value={query} onChange={e => setQuery(e.target.value)} placeholder="Search by name, slug, or suite…" />{query && <button onClick={() => setQuery('')} aria-label="Clear search"><Icon name="x" size={14} /></button>}</label>
            <div className="filter-group" aria-label="Filter by format">{(['all', 'deb', 'rpm'] as RepoFilter[]).map(value => <button key={value} className={filter === value ? 'active' : ''} onClick={() => setFilter(value)}>{value === 'all' ? 'All' : value.toUpperCase()} {value === 'all' ? repos.length : value === 'deb' ? debCount : rpmCount}</button>)}</div>
            <label className="sort-control"><span className="sr-only">Sort repositories</span><select value={sort} onChange={e => setSort(e.target.value as RepoSort)}><option value="newest">Newest first</option><option value="oldest">Oldest first</option><option value="name">Name A–Z</option></select></label>
          </div>

          {visibleRepos.length === 0 ? <div className="no-results"><Icon name="search" size={26} /><h3>No repositories found</h3><p>Try a different search or package format.</p><button className="ghost" onClick={() => { setQuery(''); setFilter('all') }}>Clear filters</button></div> :
          <div className="repo-grid">{visibleRepos.map(repo => <Link key={repo.id} to={`/repos/${repo.id}`} className="repo-card">
            <div className="repo-card-top"><span className={`format-mark format-${repo.type}`}><Icon name="repo" size={20} /></span><span className={`format-badge format-badge-${repo.type}`}>{repo.type.toUpperCase()}</span></div>
            <div className="repo-card-body"><h3>{repo.name}</h3><span className="repo-path">/{repo.slug}</span></div>
            <dl className="repo-meta"><div><dt>{repo.type === 'deb' ? 'Suite' : 'Client'}</dt><dd>{repo.type === 'deb' ? repo.codename : 'yum / dnf'}</dd></div><div><dt>Created</dt><dd>{formatDate(repo.created_at)}</dd></div></dl>
            <div className="repo-card-footer"><span>Open repository</span><span className="card-arrow"><Icon name="arrow" size={14} /></span></div>
          </Link>)}</div>}
        </section>
      </>}
    </div>
  )
}

function DashboardSkeleton() {
  return <div className="dashboard-skeleton" aria-label="Loading repositories"><div className="skeleton metrics-skeleton" /><div className="skeleton toolbar-skeleton" /><div className="skeleton-grid"><div className="skeleton card-skeleton" /><div className="skeleton card-skeleton" /><div className="skeleton card-skeleton" /></div></div>
}

function formatDate(value: string) {
  return new Date(value).toLocaleDateString(undefined, { month: 'short', day: 'numeric', year: 'numeric' })
}
