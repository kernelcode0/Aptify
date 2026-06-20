import { useState } from 'react'
import { Link, useOutletContext } from 'react-router-dom'
import { type CurrentUser, type Role } from '../api'
import './Profile.css'

type IconName = 'user' | 'shield' | 'check' | 'copy' | 'arrow' | 'key' | 'team' | 'audit' | 'repo' | 'spark'

function Icon({ name, size = 16 }: { name: IconName; size?: number }) {
  const paths: Record<IconName, React.ReactNode> = {
    user: <><circle cx="12" cy="8" r="4" /><path d="M4 21v-2a6 6 0 0 1 6-6h4a6 6 0 0 1 6 6v2" /></>,
    shield: <><path d="M12 22s8-4 8-10V5l-8-3-8 3v7c0 6 8 10 8 10Z" /><path d="m9 12 2 2 4-4" /></>,
    check: <path d="m5 12 4 4L19 6" />,
    copy: <><rect x="9" y="9" width="11" height="11" rx="2" /><path d="M15 9V6a2 2 0 0 0-2-2H6a2 2 0 0 0-2 2v7a2 2 0 0 0 2 2h3" /></>,
    arrow: <><path d="M5 12h14M14 7l5 5-5 5" /></>,
    key: <><circle cx="7.5" cy="15.5" r="4.5" /><path d="m10.7 12.3 8.8-8.8M15 8l2 2M17.5 5.5l2 2" /></>,
    team: <><path d="M16 21v-2a4 4 0 0 0-4-4H6a4 4 0 0 0-4 4v2" /><circle cx="9" cy="7" r="4" /><path d="M22 21v-2a4 4 0 0 0-3-3.87M16 3.13a4 4 0 0 1 0 7.75" /></>,
    audit: <><path d="M14 2H6a2 2 0 0 0-2 2v16a2 2 0 0 0 2 2h12a2 2 0 0 0 2-2V8Z" /><path d="M14 2v6h6M8 13h8M8 17h5" /></>,
    repo: <><path d="M3 8h18v13H3zM1 3h22v5H1zM10 13h4" /></>,
    spark: <><path d="m12 3 1.3 4.2 4.2 1.3-4.2 1.3L12 14l-1.3-4.2-4.2-1.3 4.2-1.3L12 3Z" /></>,
  }
  return <svg width={size} height={size} viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round" aria-hidden="true">{paths[name]}</svg>
}

const roleContent: Record<Role, { name: string; summary: string; permissions: string[] }> = {
  admin: {
    name: 'Administrator',
    summary: 'Full workspace access with control over repositories, users, and security activity.',
    permissions: ['Create and manage repositories', 'Upload and remove packages', 'Manage API keys', 'Manage users and audit logs'],
  },
  member: {
    name: 'Member',
    summary: 'Operational access for managing repositories, packages, and personal API credentials.',
    permissions: ['View all repositories', 'Upload and remove packages', 'Create and manage API keys'],
  },
  viewer: {
    name: 'Viewer',
    summary: 'Read-only workspace access for browsing repositories and published packages.',
    permissions: ['View all repositories', 'Browse package metadata', 'Access repository setup instructions'],
  },
}

export default function Profile() {
  const { user } = useOutletContext<{ user: CurrentUser | null }>()
  const [copied, setCopied] = useState(false)

  if (!user) return null

  const role = roleContent[user.role]
  const initials = user.username.slice(0, 2).toUpperCase()

  const copyAccountId = async () => {
    await navigator.clipboard.writeText(user.user_id)
    setCopied(true)
    window.setTimeout(() => setCopied(false), 1800)
  }

  return (
    <div className="profile-page">
      <header className="profile-header">
        <div>
          <div className="page-kicker"><Icon name="spark" size={13} /> Account</div>
          <h1 className="page-title">Your profile</h1>
          <p className="page-sub">Your identity and access across this Aptify workspace.</p>
        </div>
        <div className="session-state"><span />Authenticated session</div>
      </header>

      <div className="profile-layout">
        <section className="identity-card" aria-labelledby="identity-title">
          <div className="identity-accent" />
          <div className="identity-avatar" aria-hidden="true"><span>{initials}</span><i><Icon name="check" size={11} /></i></div>
          <div className="identity-copy">
            <span className="identity-label">Signed in as</span>
            <h2 id="identity-title">{user.username}</h2>
            <span className={`role-badge role-${user.role}`}><Icon name="shield" size={12} />{role.name}</span>
          </div>

          <dl className="account-details">
            <div><dt>Username</dt><dd>{user.username}</dd></div>
            <div><dt>Account ID</dt><dd><code title={user.user_id}>{user.user_id}</code><button onClick={copyAccountId} className={copied ? 'copied' : ''} aria-label="Copy account ID"><Icon name={copied ? 'check' : 'copy'} size={14} /><span>{copied ? 'Copied' : 'Copy'}</span></button></dd></div>
            <div><dt>Account status</dt><dd><span className="status-dot" />Active</dd></div>
          </dl>
        </section>

        <div className="profile-main">
          <section className="access-card" aria-labelledby="access-title">
            <div className="section-heading">
              <span className={`section-icon role-icon-${user.role}`}><Icon name="shield" size={19} /></span>
              <div><span className="section-eyebrow">Role & access</span><h2 id="access-title">{role.name} permissions</h2></div>
            </div>
            <p className="access-summary">{role.summary}</p>
            <div className="permission-grid">
              {role.permissions.map(permission => <div className="permission-item" key={permission}><span><Icon name="check" size={12} /></span>{permission}</div>)}
            </div>
          </section>

          <section className="profile-links" aria-labelledby="shortcuts-title">
            <div className="profile-links-heading"><div><span className="section-eyebrow">Workspace</span><h2 id="shortcuts-title">Quick access</h2></div><span>Based on your role</span></div>
            <div className="link-grid">
              <ProfileLink to="/" icon="repo" title="Repositories" description="Browse package sources" />
              {user.role !== 'viewer' && <ProfileLink to="/api-keys" icon="key" title="API keys" description="Manage credentials" />}
              {user.role === 'admin' && <ProfileLink to="/users" icon="team" title="Users" description="Manage workspace access" />}
              {user.role === 'admin' && <ProfileLink to="/audit" icon="audit" title="Audit log" description="Review security activity" />}
            </div>
          </section>
        </div>
      </div>
    </div>
  )
}

function ProfileLink({ to, icon, title, description }: { to: string; icon: IconName; title: string; description: string }) {
  return <Link to={to} className="profile-link"><span className="profile-link-icon"><Icon name={icon} /></span><span><strong>{title}</strong><small>{description}</small></span><i><Icon name="arrow" size={14} /></i></Link>
}
