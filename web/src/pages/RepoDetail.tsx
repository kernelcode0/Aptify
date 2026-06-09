import { useEffect, useRef, useState } from 'react'
import { useParams, useNavigate, Link } from 'react-router-dom'
import { api, type Package, type Repo, type SetupInfo } from '../api'
import './RepoDetail.css'

type Tab = 'packages' | 'setup'

function ChevronIcon() {
  return (
    <svg width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2.5" strokeLinecap="round" strokeLinejoin="round">
      <polyline points="9 18 15 12 9 6"/>
    </svg>
  )
}

function EditIcon() {
  return (
    <svg width="13" height="13" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round">
      <path d="M11 4H4a2 2 0 0 0-2 2v14a2 2 0 0 0 2 2h14a2 2 0 0 0 2-2v-7"/>
      <path d="M18.5 2.5a2.121 2.121 0 0 1 3 3L12 15l-4 1 1-4 9.5-9.5z"/>
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

function UploadIcon() {
  return (
    <svg width="32" height="32" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="1.5" strokeLinecap="round" strokeLinejoin="round">
      <polyline points="16 16 12 12 8 16"/>
      <line x1="12" y1="12" x2="12" y2="21"/>
      <path d="M20.39 18.39A5 5 0 0 0 18 9h-1.26A8 8 0 1 0 3 16.3"/>
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

function EmptyPackagesIcon() {
  return (
    <svg width="48" height="48" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="1.25" strokeLinecap="round" strokeLinejoin="round">
      <rect x="2" y="7" width="20" height="14" rx="2" ry="2"/>
      <path d="M16 7V5a2 2 0 0 0-2-2h-4a2 2 0 0 0-2 2v2"/>
      <line x1="12" y1="12" x2="12" y2="16"/>
      <line x1="10" y1="14" x2="14" y2="14"/>
    </svg>
  )
}

export default function RepoDetail() {
  const { id } = useParams<{ id: string }>()
  const navigate = useNavigate()
  const [repo, setRepo] = useState<Repo | null>(null)
  const [packages, setPackages] = useState<Package[]>([])
  const [totalPkgs, setTotalPkgs] = useState(0)
  const [page, setPage] = useState(1)
  const [setup, setSetup] = useState<SetupInfo | null>(null)
  const [tab, setTab] = useState<Tab>('packages')
  const [editing, setEditing] = useState(false)
  const [editForm, setEditForm] = useState({ name: '', codename: '' })
  const [saving, setSaving] = useState(false)
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState('')
  const [uploading, setUploading] = useState(false)
  const [dragOver, setDragOver] = useState(false)
  const inputRef = useRef<HTMLInputElement>(null)

  const loadRepo = async () => {
    if (!id) return
    try {
      const [repos, pkgsData] = await Promise.all([
        api.listRepos(),
        api.listPackages(id, page, 50),
      ])
      const found = repos.find(r => r.id === id)
      if (!found) { navigate('/'); return }
      setRepo(found)
      setPackages(pkgsData.packages)
      setTotalPkgs(pkgsData.total)
    } catch (e: unknown) {
      setError(e instanceof Error ? e.message : 'error')
    } finally {
      setLoading(false)
    }
  }

  const loadSetup = async () => {
    if (!id || setup) return
    try {
      const s = await api.getSetup(id)
      setSetup(s)
    } catch {}
  }

  useEffect(() => { loadRepo() }, [id])

  useEffect(() => {
    if (tab === 'setup') loadSetup()
  }, [tab])

  const handleUpload = async (file: File) => {
    if (!id || !file.name.endsWith('.deb')) {
      setError('Only .deb files are accepted')
      return
    }
    setUploading(true)
    setError('')
    try {
      const pkg = await api.uploadPackage(id, file)
      setPackages(p => [pkg, ...p].slice(0, 50))
      setTotalPkgs(t => t + 1)
      if (tab === 'setup') setSetup(null)
    } catch (e: unknown) {
      setError(e instanceof Error ? e.message : 'upload failed')
    } finally {
      setUploading(false)
    }
  }

  const handleDrop = (e: React.DragEvent) => {
    e.preventDefault()
    setDragOver(false)
    const file = e.dataTransfer.files[0]
    if (file) handleUpload(file)
  }

  const handleDelete = async (pkg: Package) => {
    if (!id) return
    if (!confirm(`Remove ${pkg.package} ${pkg.version}?`)) return
    try {
      await api.deletePackage(id, pkg.id)
      setPackages(p => p.filter(x => x.id !== pkg.id))
      setTotalPkgs(t => t - 1)
      if (tab === 'setup') setSetup(null)
    } catch (e: unknown) {
      setError(e instanceof Error ? e.message : 'delete failed')
    }
  }

  const handleDeleteRepo = async () => {
    if (!id || !repo) return
    if (!confirm(`Permanently delete repository "${repo.name}"? This cannot be undone.`)) return
    try {
      await api.deleteRepo(id)
      navigate('/')
    } catch (e: unknown) {
      setError(e instanceof Error ? e.message : 'delete failed')
    }
  }

  const handleEditSave = async (e: React.FormEvent) => {
    e.preventDefault()
    if (!id) return
    setSaving(true)
    setError('')
    try {
      const updated = await api.updateRepo(id, editForm.name, editForm.codename)
      setRepo(updated)
      setEditing(false)
      setSetup(null)
    } catch (err: unknown) {
      setError(err instanceof Error ? err.message : 'failed to save')
    } finally {
      setSaving(false)
    }
  }

  const openEdit = () => {
    if (!repo) return
    setEditForm({ name: repo.name, codename: repo.codename })
    setEditing(true)
  }

  if (loading) {
    return (
      <div style={{ display: 'flex', justifyContent: 'center', padding: '72px 0' }}>
        <div className="spinner" />
      </div>
    )
  }

  return (
    <div>
      <div className="detail-header">
        <div className="detail-header-left">
          <div className="breadcrumb">
            <Link to="/">Repositories</Link>
            <ChevronIcon />
            <span>{repo?.slug}</span>
          </div>
          <h1 className="detail-title">{repo?.name}</h1>
          <div className="detail-meta">
            <span className="tag">{repo?.codename}</span>
            <span className="meta-dot" />
            <span className="muted-text">{totalPkgs} package{totalPkgs !== 1 ? 's' : ''}</span>
          </div>
        </div>
        <div className="detail-actions">
          <button className="ghost" onClick={openEdit}>
            <EditIcon />
            Edit
          </button>
          <button className="danger" onClick={handleDeleteRepo}>
            <TrashIcon />
            Delete
          </button>
        </div>
      </div>

      {editing && (
        <div className="overlay" onClick={e => { if (e.target === e.currentTarget) setEditing(false) }}>
          <div className="modal">
            <div className="modal-header">
              <h2>Edit Repository</h2>
              <button className="modal-close" onClick={() => setEditing(false)} aria-label="Close">
                <XIcon />
              </button>
            </div>
            <form onSubmit={handleEditSave} className="form-stack">
              <div className="field">
                <label>Name</label>
                <input value={editForm.name} onChange={e => setEditForm(f => ({ ...f, name: e.target.value }))} required autoFocus />
              </div>
              <div className="field">
                <label>Codename</label>
                <input value={editForm.codename} onChange={e => setEditForm(f => ({ ...f, codename: e.target.value }))} required />
              </div>
              <div className="form-actions">
                <button type="submit" className="primary" disabled={saving}>
                  {saving
                    ? <><div className="spinner" style={{ width: 14, height: 14, borderWidth: '2px' }} />Saving…</>
                    : 'Save Changes'
                  }
                </button>
                <button type="button" className="ghost" onClick={() => setEditing(false)}>Cancel</button>
              </div>
            </form>
          </div>
        </div>
      )}

      {error && (
        <div className="alert-error">
          <AlertIcon />
          {error}
        </div>
      )}

      <div className="tabs">
        <button className={tab === 'packages' ? 'tab active' : 'tab'} onClick={() => setTab('packages')}>
          Packages
          {totalPkgs > 0 && <span className="tab-badge">{totalPkgs}</span>}
        </button>
        <button className={tab === 'setup' ? 'tab active' : 'tab'} onClick={() => setTab('setup')}>
          Setup Instructions
        </button>
      </div>

      {tab === 'packages' && (
        <div>
          <div
            className={`upload-zone${dragOver ? ' drag-over' : ''}${uploading ? ' uploading' : ''}`}
            onDragOver={e => { e.preventDefault(); setDragOver(true) }}
            onDragLeave={() => setDragOver(false)}
            onDrop={handleDrop}
            onClick={() => inputRef.current?.click()}
          >
            <input ref={inputRef} type="file" accept=".deb" style={{ display: 'none' }}
              onChange={e => { const f = e.target.files?.[0]; if (f) handleUpload(f) }} />
            {uploading ? (
              <div className="upload-uploading">
                <div className="spinner" />
                <div>
                  <div className="upload-title">Uploading and indexing…</div>
                  <div className="upload-sub">This may take a moment</div>
                </div>
              </div>
            ) : (
              <div className="upload-idle">
                <span className="upload-icon"><UploadIcon /></span>
                <div>
                  <div className="upload-title">Drop a <code>.deb</code> file here</div>
                  <div className="upload-sub">or click to browse your files</div>
                </div>
              </div>
            )}
          </div>

          {packages.length === 0 ? (
            <div className="empty-state">
              <div className="empty-icon"><EmptyPackagesIcon /></div>
              <h3>No packages yet</h3>
              <p>Upload a .deb file to add your first package.</p>
            </div>
          ) : (
            <div className="card" style={{ padding: 0, overflow: 'hidden' }}>
              <table>
                <thead>
                  <tr>
                    <th>Package</th>
                    <th>Version</th>
                    <th>Arch</th>
                    <th>Size</th>
                    <th>Uploaded</th>
                    <th style={{ width: 60 }}></th>
                  </tr>
                </thead>
                <tbody>
                  {packages.map(p => (
                    <tr key={p.id}>
                      <td><span className="pkg-name">{p.package}</span></td>
                      <td><code className="pkg-version">{p.version}</code></td>
                      <td><span className="tag">{p.arch}</span></td>
                      <td className="muted-text">{formatBytes(p.size)}</td>
                      <td className="muted-text">{formatDate(p.uploaded_at)}</td>
                      <td>
                        <button className="danger icon-btn" onClick={() => handleDelete(p)} title="Remove package">
                          <TrashIcon />
                        </button>
                      </td>
                    </tr>
                  ))}
                </tbody>
              </table>

              {totalPkgs > 50 && (
                <div className="pagination">
                  <button className="ghost" disabled={page === 1} onClick={() => setPage(p => p - 1)}>← Previous</button>
                  <span className="muted-text">Page {page} of {Math.ceil(totalPkgs / 50)}</span>
                  <button className="ghost" disabled={page >= Math.ceil(totalPkgs / 50)} onClick={() => setPage(p => p + 1)}>Next →</button>
                </div>
              )}
            </div>
          )}
        </div>
      )}

      {tab === 'setup' && setup && (
        <SetupInstructions setup={setup} slug={repo?.slug ?? ''} packages={packages} />
      )}
      {tab === 'setup' && !setup && (
        <div style={{ display: 'flex', justifyContent: 'center', padding: '48px 0' }}>
          <div className="spinner" />
        </div>
      )}
    </div>
  )
}

function SetupInstructions({ setup, slug, packages }: { setup: SetupInfo; slug: string; packages: Package[] }) {
  const examplePkg = packages.length > 0 ? packages[0].package : '<package-name>'
  return (
    <div className="setup">
      <p className="setup-intro">Add this repository to your system and install packages.</p>

      <SetupStep n={1} title="Import the signing key">
        <CopyBlock code={setup.addKey} />
      </SetupStep>

      <SetupStep n={2} title="Add the repository source">
        <CopyBlock code={setup.addSource} />
      </SetupStep>

      <SetupStep n={3} title="Update package index">
        <CopyBlock code={setup.update} />
      </SetupStep>

      <SetupStep n={4} title="Install a package">
        <CopyBlock code={`sudo apt install ${examplePkg}`} />
      </SetupStep>

      <div className="setup-details card">
        <div className="setup-details-title">Repository details</div>
        <table>
          <tbody>
            <tr><td className="info-label">Repository URL</td><td><code>{setup.repoURL}</code></td></tr>
            <tr><td className="info-label">Codename</td><td><code>{setup.codename}</code></td></tr>
            <tr><td className="info-label">Component</td><td><code>{setup.component}</code></td></tr>
            <tr><td className="info-label">Signing key</td><td><a href={setup.keyURL}>{setup.keyURL}</a></td></tr>
          </tbody>
        </table>
      </div>

      <div className="setup-sources card">
        <div className="setup-sources-label">/etc/apt/sources.list.d/{slug}.list</div>
        <pre>{setup.addSource.split('"')[1]}</pre>
      </div>
    </div>
  )
}

function SetupStep({ n, title, children }: { n: number; title: string; children: React.ReactNode }) {
  return (
    <div className="setup-step">
      <div className="step-header">
        <span className="step-num">{n}</span>
        <span className="step-title">{title}</span>
      </div>
      {children}
    </div>
  )
}

function CopyBlock({ code }: { code: string }) {
  const [copied, setCopied] = useState(false)
  const copy = () => {
    navigator.clipboard.writeText(code).then(() => {
      setCopied(true)
      setTimeout(() => setCopied(false), 1500)
    })
  }
  return (
    <div className="copy-block">
      <pre>{code}</pre>
      <button className="copy-btn" onClick={copy}>{copied ? '✓ Copied' : 'Copy'}</button>
    </div>
  )
}

function formatBytes(b: number) {
  if (b < 1024) return b + ' B'
  if (b < 1024 * 1024) return (b / 1024).toFixed(1) + ' KB'
  return (b / 1024 / 1024).toFixed(1) + ' MB'
}

function formatDate(s: string) {
  return new Date(s).toLocaleDateString(undefined, { month: 'short', day: 'numeric', year: 'numeric' })
}
