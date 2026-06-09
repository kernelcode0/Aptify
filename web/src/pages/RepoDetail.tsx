import { useEffect, useRef, useState } from 'react'
import { useParams, useNavigate, Link } from 'react-router-dom'
import { api, type Package, type Repo, type SetupInfo } from '../api'
import './RepoDetail.css'

type Tab = 'packages' | 'setup'

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
      if (tab === 'setup') setSetup(null) // invalidate setup to refresh arches
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
      setSetup(null) // invalidate setup
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

  if (loading) return <div style={{ display: 'flex', justifyContent: 'center', padding: '64px' }}><div className="spinner" /></div>

  return (
    <div>
      <div className="detail-header">
        <div>
          <div className="breadcrumb"><Link to="/">Repositories</Link> / {repo?.slug}</div>
          <h1>{repo?.name}</h1>
          <div className="detail-meta">
            <span className="tag">{repo?.codename}</span>
            <span className="muted-text">{totalPkgs} package{totalPkgs !== 1 ? 's' : ''}</span>
          </div>
        </div>
        <div style={{ display: 'flex', gap: '8px' }}>
          <button className="ghost" onClick={openEdit}>Edit</button>
          <button className="danger" onClick={handleDeleteRepo}>Delete Repo</button>
        </div>
      </div>

      {editing && (
        <div className="card" style={{ marginBottom: 24 }}>
          <h2>Edit Repository</h2>
          <form onSubmit={handleEditSave} className="new-repo-form">
            <div className="field">
              <label>Name</label>
              <input value={editForm.name} onChange={e => setEditForm(f => ({ ...f, name: e.target.value }))} required />
            </div>
            <div className="field">
              <label>Codename</label>
              <input value={editForm.codename} onChange={e => setEditForm(f => ({ ...f, codename: e.target.value }))} required />
            </div>
            <div className="form-actions">
              <button type="submit" className="primary" disabled={saving}>{saving ? 'Saving...' : 'Save Changes'}</button>
              <button type="button" className="ghost" onClick={() => setEditing(false)}>Cancel</button>
            </div>
          </form>
        </div>
      )}

      {error && <div className="alert-error">{error}</div>}

      <div className="tabs">
        <button className={tab === 'packages' ? 'tab active' : 'tab'} onClick={() => setTab('packages')}>Packages</button>
        <button className={tab === 'setup' ? 'tab active' : 'tab'} onClick={() => setTab('setup')}>Setup Instructions</button>
      </div>

      {tab === 'packages' && (
        <div>
          <div
            className={`upload-zone ${dragOver ? 'drag-over' : ''} ${uploading ? 'uploading' : ''}`}
            onDragOver={e => { e.preventDefault(); setDragOver(true) }}
            onDragLeave={() => setDragOver(false)}
            onDrop={handleDrop}
            onClick={() => inputRef.current?.click()}
          >
            <input ref={inputRef} type="file" accept=".deb" style={{ display: 'none' }}
              onChange={e => { const f = e.target.files?.[0]; if (f) handleUpload(f) }} />
            {uploading
              ? <><div className="spinner" /><span>Uploading and indexing…</span></>
              : <><span className="upload-icon">⬆️</span><span>Drop a <code>.deb</code> file here, or click to browse</span></>
            }
          </div>

          {packages.length === 0 ? (
            <div className="empty-state">
              <h3>No packages yet</h3>
              <p>Upload a .deb file to get started.</p>
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
                    <th></th>
                  </tr>
                </thead>
                <tbody>
                  {packages.map(p => (
                    <tr key={p.id}>
                      <td><strong>{p.package}</strong></td>
                      <td><code>{p.version}</code></td>
                      <td><span className="tag">{p.arch}</span></td>
                      <td className="muted-text">{formatBytes(p.size)}</td>
                      <td className="muted-text">{formatDate(p.uploaded_at)}</td>
                      <td><button className="danger" style={{ fontSize: '11px', padding: '3px 10px' }} onClick={() => handleDelete(p)}>Remove</button></td>
                    </tr>
                  ))}
                </tbody>
              </table>
              
              {totalPkgs > 50 && (
                <div style={{ display: 'flex', justifyContent: 'space-between', padding: '16px', borderTop: '1px solid var(--border)' }}>
                  <button className="ghost" disabled={page === 1} onClick={() => setPage(p => p - 1)}>Previous</button>
                  <span className="muted-text" style={{ fontSize: 14 }}>Page {page} of {Math.ceil(totalPkgs / 50)}</span>
                  <button className="ghost" disabled={page >= Math.ceil(totalPkgs / 50)} onClick={() => setPage(p => p + 1)}>Next</button>
                </div>
              )}
            </div>
          )}
        </div>
      )}

      {tab === 'setup' && setup && (
        <SetupInstructions setup={setup} slug={repo?.slug ?? ''} packages={packages} />
      )}
    </div>
  )
}

function SetupInstructions({ setup, slug, packages }: { setup: SetupInfo; slug: string; packages: Package[] }) {
  const examplePkg = packages.length > 0 ? packages[0].package : '<package-name>'
  return (
    <div className="setup">
      <p className="setup-intro">Add this repository to your system and install packages:</p>

      <SetupStep n={1} title="Import the signing key">
        <CopyBlock code={setup.addKey} />
      </SetupStep>

      <SetupStep n={2} title="Add the repository source">
        <CopyBlock code={setup.addSource} />
      </SetupStep>

      <SetupStep n={3} title="Update apt">
        <CopyBlock code={setup.update} />
      </SetupStep>

      <SetupStep n={4} title="Install a package">
        <CopyBlock code={`sudo apt install ${examplePkg}`} />
      </SetupStep>

      <div className="setup-info card" style={{ marginTop: 24 }}>
        <table>
          <tbody>
            <tr><td className="info-label">Repository URL</td><td><code>{setup.repoURL}</code></td></tr>
            <tr><td className="info-label">Codename</td><td><code>{setup.codename}</code></td></tr>
            <tr><td className="info-label">Component</td><td><code>{setup.component}</code></td></tr>
            <tr><td className="info-label">Signing key</td><td><a href={setup.keyURL}>{setup.keyURL}</a></td></tr>
          </tbody>
        </table>
      </div>

      <div className="setup-sources card" style={{ marginTop: 16 }}>
        <p style={{ marginBottom: 8, fontSize: 12, color: 'var(--muted)', fontWeight: 600, textTransform: 'uppercase', letterSpacing: '0.05em' }}>
          /etc/apt/sources.list.d/{slug}.list
        </p>
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
