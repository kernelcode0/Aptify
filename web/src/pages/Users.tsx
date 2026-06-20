import { useEffect, useMemo, useState } from 'react'
import { useOutletContext } from 'react-router-dom'
import { api, type CurrentUser, type Role, type User } from '../api'
import './Users.css'

function PlusIcon() {
  return (
    <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2.5" strokeLinecap="round">
      <line x1="12" y1="5" x2="12" y2="19"/><line x1="5" y1="12" x2="19" y2="12"/>
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

export default function Users() {
  const { user: current } = useOutletContext<{ user: CurrentUser | null }>()
  const [users, setUsers] = useState<User[]>([])
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState('')
  const [showNew, setShowNew] = useState(false)
  const [creating, setCreating] = useState(false)
  const [createError, setCreateError] = useState('')
  const [newUser, setNewUser] = useState({ username: '', password: '', role: 'viewer' as Role })
  const [editing, setEditing] = useState<string | null>(null)
  const [editRole, setEditRole] = useState<Role>('viewer')
  const [editPassword, setEditPassword] = useState('')
  const [query, setQuery] = useState('')
  const [roleFilter, setRoleFilter] = useState<'all' | Role>('all')

  const load = () => {
    api.listUsers()
      .then(u => setUsers(u ?? []))
      .catch(e => setError(e.message))
      .finally(() => setLoading(false))
  }

  useEffect(load, [])

  const visibleUsers = useMemo(() => {
    const term = query.trim().toLowerCase()
    return users.filter(u => (roleFilter === 'all' || u.role === roleFilter) && (!term || u.username.toLowerCase().includes(term)))
  }, [users, query, roleFilter])

  const roleCount = (role: Role) => users.filter(user => user.role === role).length

  const handleCreate = async (e: React.FormEvent) => {
    e.preventDefault()
    setCreating(true)
    setCreateError('')
    try {
      const created = await api.createUser(newUser.username.trim(), newUser.password, newUser.role)
      setUsers(prev => [...prev, created])
      setNewUser({ username: '', password: '', role: 'viewer' })
      setShowNew(false)
    } catch (e: unknown) {
      setCreateError(e instanceof Error ? e.message : 'failed to create user')
    } finally {
      setCreating(false)
    }
  }

  const openEdit = (u: User) => {
    setEditing(u.id)
    setEditRole(u.role)
    setEditPassword('')
  }

  const saveEdit = async (u: User) => {
    setError('')
    try {
      const data: { role?: Role; password?: string } = {}
      if (editRole !== u.role) data.role = editRole
      if (editPassword) data.password = editPassword
      const updated = await api.updateUser(u.id, data)
      setUsers(prev => prev.map(x => x.id === u.id ? updated : x))
      setEditing(null)
    } catch (e: unknown) {
      setError(e instanceof Error ? e.message : 'failed to update user')
    }
  }

  const handleDelete = async (u: User) => {
    if (!confirm(`Delete user "${u.username}"? This cannot be undone.`)) return
    setError('')
    try {
      await api.deleteUser(u.id)
      setUsers(prev => prev.filter(x => x.id !== u.id))
    } catch (e: unknown) {
      setError(e instanceof Error ? e.message : 'failed to delete user')
    }
  }

  return (
    <div className="users-page">
      <div className="page-header">
        <div>
          <div className="page-kicker">Access control</div>
          <h1 className="page-title">Users</h1>
          <p className="page-sub">Manage who can access Aptify and what they can do.</p>
        </div>
        <div className="page-actions">
          <button className="primary" onClick={() => { setCreateError(''); setShowNew(true) }}>
            <PlusIcon />
            Add User
          </button>
        </div>
      </div>

      {error && <div className="alert-error"><AlertIcon />{error}</div>}

      {!loading && <div className="user-metrics" aria-label="User overview">
        <div className="user-metric user-metric-total"><span>Workspace accounts</span><strong>{users.length}</strong><small>{users.length === 1 ? 'Active identity' : 'Active identities'}</small></div>
        <div className="user-metric"><i className="metric-admin" /><span>Administrators</span><strong>{roleCount('admin')}</strong></div>
        <div className="user-metric"><i className="metric-member" /><span>Members</span><strong>{roleCount('member')}</strong></div>
        <div className="user-metric"><i className="metric-viewer" /><span>Viewers</span><strong>{roleCount('viewer')}</strong></div>
      </div>}

      {showNew && (
        <div className="overlay" onClick={e => { if (e.target === e.currentTarget) setShowNew(false) }}>
          <div className="modal">
            <div className="modal-header">
              <h2>Add User</h2>
              <button className="modal-close" onClick={() => setShowNew(false)} aria-label="Close"><XIcon /></button>
            </div>
            <form onSubmit={handleCreate} className="form-stack">
              {createError && <div className="alert-error" role="alert"><AlertIcon />{createError}</div>}
              <div className="field">
                <label>Username</label>
                <input value={newUser.username} onChange={e => { setCreateError(''); setNewUser(f => ({ ...f, username: e.target.value })) }} minLength={3} maxLength={32} pattern="[A-Za-z0-9_.-]{3,32}" aria-invalid={createError.toLowerCase().includes('username')} required autoFocus />
                <span className="field-hint">3-32 characters. Letters, numbers, dots, hyphens, and underscores are allowed.</span>
              </div>
              <div className="field">
                <label>Password</label>
                <input type="password" value={newUser.password} onChange={e => setNewUser(f => ({ ...f, password: e.target.value }))} minLength={8} required />
              </div>
              <div className="field">
                <label>Role</label>
                <select value={newUser.role} onChange={e => setNewUser(f => ({ ...f, role: e.target.value as Role }))}>
                  <option value="admin">Admin</option>
                  <option value="member">Member</option>
                  <option value="viewer">Viewer</option>
                </select>
              </div>
              <div className="form-actions">
                <button type="submit" className="primary" disabled={creating}>
                  {creating ? <><div className="spinner" style={{ width: 14, height: 14, borderWidth: '2px' }} />Creating…</> : 'Create User'}
                </button>
                <button type="button" className="ghost" onClick={() => setShowNew(false)}>Cancel</button>
              </div>
            </form>
          </div>
        </div>
      )}

      {loading ? (
        <div className="loading-center"><div className="spinner" /></div>
      ) : (
        <section className="users-collection">
          <div className="users-collection-header">
            <div><h2>Workspace users</h2><p>{visibleUsers.length === users.length ? `${users.length} account${users.length === 1 ? '' : 's'}` : `${visibleUsers.length} of ${users.length} accounts`}</p></div>
            <div className="users-toolbar">
              <label className="user-search"><span className="search-symbol" aria-hidden="true" /><span className="sr-only">Search users</span><input value={query} onChange={e => setQuery(e.target.value)} placeholder="Search users…" /></label>
              <label><span className="sr-only">Filter by role</span><select value={roleFilter} onChange={e => setRoleFilter(e.target.value as 'all' | Role)}><option value="all">All roles</option><option value="admin">Administrators</option><option value="member">Members</option><option value="viewer">Viewers</option></select></label>
            </div>
          </div>
        <div className="package-table card users-table">
          <table>
            <thead>
              <tr>
                <th>Username</th>
                <th>Role</th>
                <th>Created</th>
                <th style={{ width: 210 }}>Actions</th>
              </tr>
            </thead>
            <tbody>
              {visibleUsers.map(u => {
                const isSelf = current?.user_id === u.id
                return (
                  <tr key={u.id}>
                    <td><div className="user-identity"><span className={`user-avatar avatar-${u.role}`}>{u.username.slice(0, 2).toUpperCase()}</span><span><strong>{u.username}</strong><small>{isSelf ? 'Your account' : 'Workspace user'}</small></span>{isSelf && <span className="self-label">You</span>}</div></td>
                    <td>
                      {editing === u.id ? (
                        <select value={editRole} disabled={isSelf} onChange={e => setEditRole(e.target.value as Role)}>
                          <option value="admin">Admin</option>
                          <option value="member">Member</option>
                          <option value="viewer">Viewer</option>
                        </select>
                      ) : (
                        <RoleBadge role={u.role} />
                      )}
                    </td>
                    <td className="muted-text">{formatDate(u.created_at)}</td>
                    <td>
                      {editing === u.id ? (
                        <div className="user-edit-actions">
                          <input type="password" placeholder="New password" value={editPassword} onChange={e => setEditPassword(e.target.value)} />
                          <button className="primary" onClick={() => saveEdit(u)}>Save</button>
                          <button className="ghost" onClick={() => setEditing(null)}>Cancel</button>
                        </div>
                      ) : (
                        <div className="user-actions">
                          <button className="ghost icon-btn" onClick={() => openEdit(u)} title="Edit user"><EditIcon /></button>
                          {!isSelf && <button className="danger icon-btn" onClick={() => handleDelete(u)} title="Delete user"><TrashIcon /></button>}
                        </div>
                      )}
                    </td>
                  </tr>
                )
              })}
              {visibleUsers.length === 0 && <tr><td colSpan={4}><div className="users-no-results"><strong>No users found</strong><span>Try a different name or role.</span><button className="ghost" onClick={() => { setQuery(''); setRoleFilter('all') }}>Clear filters</button></div></td></tr>}
            </tbody>
          </table>
        </div>
        </section>
      )}
    </div>
  )
}

function RoleBadge({ role }: { role: Role }) {
  return <span className={`role-badge role-${role}`}>{role}</span>
}

function formatDate(s: string) {
  return new Date(s).toLocaleDateString(undefined, { month: 'short', day: 'numeric', year: 'numeric' })
}
