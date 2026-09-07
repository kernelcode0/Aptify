import { useState } from 'react'
import { Link, useOutletContext } from 'react-router-dom'
import { api, type CurrentUser, type Role } from '../api'
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

function XIcon() {
  return (
    <svg width="15" height="15" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round">
      <line x1="18" y1="6" x2="6" y2="18"/>
      <line x1="6" y1="6" x2="18" y2="18"/>
    </svg>
  )
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
  const [twoFactorEnabled, setTwoFactorEnabled] = useState(user?.two_factor_enabled ?? false)

  // 2FA Modals state
  const [modalType, setModalType] = useState<'setup' | 'disable' | 'regen' | null>(null)
  const [setupData, setSetupData] = useState<{ secret: string; otpauth_url: string; qr_code: string; recovery_codes: string[] } | null>(null)
  const [verifyCode, setVerifyCode] = useState('')
  const [passwordInput, setPasswordInput] = useState('')
  const [disableCode, setDisableCode] = useState('')
  const [newRecoveryCodes, setNewRecoveryCodes] = useState<string[] | null>(null)
  const [loading, setLoading] = useState(false)
  const [modalError, setModalError] = useState('')
  const [codesCopied, setCodesCopied] = useState(false)

  if (!user) return null

  const role = roleContent[user.role]
  const initials = user.username.slice(0, 2).toUpperCase()

  const copyAccountId = async () => {
    await navigator.clipboard.writeText(user.user_id)
    setCopied(true)
    window.setTimeout(() => setCopied(false), 1800)
  }

  const handleStartSetup = async () => {
    setLoading(true)
    setModalError('')
    try {
      const data = await api.setup2FA()
      setSetupData(data)
      setVerifyCode('')
      setModalType('setup')
    } catch (err: unknown) {
      alert(err instanceof Error ? err.message : 'Failed to start 2FA setup')
    } finally {
      setLoading(false)
    }
  }

  const handleConfirmEnable = async (e: React.FormEvent) => {
    e.preventDefault()
    setLoading(true)
    setModalError('')
    try {
      await api.enable2FA(verifyCode)
      setTwoFactorEnabled(true)
      user.two_factor_enabled = true
      setModalType(null)
    } catch (err: unknown) {
      setModalError(err instanceof Error ? err.message : 'Invalid verification code')
    } finally {
      setLoading(false)
    }
  }

  const handleConfirmDisable = async (e: React.FormEvent) => {
    e.preventDefault()
    setLoading(true)
    setModalError('')
    try {
      await api.disable2FA(passwordInput, disableCode)
      setTwoFactorEnabled(false)
      user.two_factor_enabled = false
      setModalType(null)
      setPasswordInput('')
      setDisableCode('')
    } catch (err: unknown) {
      setModalError(err instanceof Error ? err.message : 'Failed to disable 2FA')
    } finally {
      setLoading(false)
    }
  }

  const handleConfirmRegen = async (e: React.FormEvent) => {
    e.preventDefault()
    setLoading(true)
    setModalError('')
    try {
      const res = await api.regenerateRecoveryCodes(passwordInput)
      setNewRecoveryCodes(res.recovery_codes)
      setPasswordInput('')
    } catch (err: unknown) {
      setModalError(err instanceof Error ? err.message : 'Failed to regenerate codes')
    } finally {
      setLoading(false)
    }
  }

  const copyCodes = async (codes: string[]) => {
    await navigator.clipboard.writeText(codes.join('\n'))
    setCodesCopied(true)
    window.setTimeout(() => setCodesCopied(false), 1800)
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
            <div><dt>Two-Factor Auth</dt><dd>{twoFactorEnabled ? <span style={{ color: '#5eead4', display: 'flex', alignItems: 'center', gap: 5 }}><Icon name="check" size={13}/> Enabled</span> : <span style={{ color: '#94a3b8' }}>Disabled</span>}</dd></div>
          </dl>
        </section>

        <div className="profile-main">
          {/* Two-Factor Authentication Card */}
          <section className="access-card" aria-labelledby="twofa-title">
            <div className="section-heading">
              <span className="section-icon" style={{ color: twoFactorEnabled ? '#5eead4' : '#fbbf24', background: twoFactorEnabled ? 'rgba(45,212,191,.09)' : 'rgba(251,191,36,.09)', borderColor: twoFactorEnabled ? 'rgba(45,212,191,.2)' : 'rgba(251,191,36,.2)' }}>
                <Icon name="shield" size={19} />
              </span>
              <div>
                <span className="section-eyebrow">Security</span>
                <h2 id="twofa-title">Two-factor authentication (2FA)</h2>
              </div>
            </div>
            <p className="access-summary">
              Protect your Aptify account with an extra layer of security using a time-based one-time password (TOTP) from Google Authenticator, 1Password, Bitwarden, or Apple Passwords.
            </p>

            <div style={{ display: 'flex', gap: '12px', alignItems: 'center', flexWrap: 'wrap' }}>
              {!twoFactorEnabled ? (
                <button
                  type="button"
                  className="primary"
                  onClick={handleStartSetup}
                  disabled={loading}
                  style={{ display: 'inline-flex', alignItems: 'center', gap: '7px', padding: '9px 16px' }}
                >
                  <Icon name="shield" size={15} />
                  <span>Enable two-factor authentication</span>
                </button>
              ) : (
                <>
                  <button
                    type="button"
                    className="ghost"
                    onClick={() => { setModalType('regen'); setNewRecoveryCodes(null); setPasswordInput(''); setModalError('') }}
                    style={{ padding: '8px 14px' }}
                  >
                    View / Regenerate recovery codes
                  </button>
                  <button
                    type="button"
                    className="danger"
                    onClick={() => { setModalType('disable'); setPasswordInput(''); setDisableCode(''); setModalError('') }}
                    style={{ padding: '8px 14px' }}
                  >
                    Disable 2FA
                  </button>
                </>
              )}
            </div>
          </section>

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

      {/* 2FA Setup Modal */}
      {modalType === 'setup' && setupData && (
        <div className="overlay" onClick={e => { if (e.target === e.currentTarget) setModalType(null) }}>
          <div className="modal" style={{ maxWidth: '520px', maxHeight: '90vh', overflowY: 'auto' }}>
            <div className="modal-header">
              <h2>Set up Two-Factor Authentication</h2>
              <button className="modal-close" onClick={() => setModalType(null)} aria-label="Close"><XIcon /></button>
            </div>
            <div className="modal-body" style={{ display: 'flex', flexDirection: 'column', gap: '16px' }}>
              {modalError && <div className="login-error" role="alert"><p>{modalError}</p></div>}
              <div>
                <strong>1. Scan this QR code</strong>
                <p style={{ color: 'var(--muted)', fontSize: '13px', margin: '4px 0 10px' }}>
                  Open your authenticator app (Google Authenticator, 1Password, Bitwarden, etc.) and scan this code:
                </p>
                <div style={{ textAlign: 'center', background: '#fff', padding: '12px', borderRadius: '8px', width: 'fit-content', margin: '0 auto' }}>
                  <img src={setupData.qr_code} alt="2FA QR Code" width="180" height="180" style={{ display: 'block' }} />
                </div>
                <p style={{ fontSize: '12px', color: 'var(--muted)', marginTop: '8px', textAlign: 'center' }}>
                  Or enter key manually: <code style={{ color: '#5eead4', letterSpacing: '1px' }}>{setupData.secret}</code>
                </p>
              </div>

              <div style={{ borderTop: '1px solid var(--border)', paddingTop: '12px' }}>
                <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
                  <strong>2. Save emergency recovery codes</strong>
                  <button type="button" className="ghost" onClick={() => copyCodes(setupData.recovery_codes)} style={{ fontSize: '12px', padding: '4px 8px' }}>
                    {codesCopied ? 'Copied!' : 'Copy all'}
                  </button>
                </div>
                <p style={{ color: 'var(--muted)', fontSize: '12px', margin: '4px 0 8px' }}>
                  If you lose your authenticator device, each recovery code can be used once to sign in.
                </p>
                <div style={{ display: 'grid', gridTemplateColumns: 'repeat(2, 1fr)', gap: '6px', background: 'rgba(0,0,0,0.25)', padding: '10px', borderRadius: '6px' }}>
                  {setupData.recovery_codes.map(c => <code key={c} style={{ fontSize: '12px', fontFamily: 'monospace' }}>{c}</code>)}
                </div>
              </div>

              <form onSubmit={handleConfirmEnable} style={{ borderTop: '1px solid var(--border)', paddingTop: '12px' }}>
                <label style={{ display: 'block', marginBottom: '8px', fontSize: '13px', fontWeight: 600 }}>
                  3. Enter 6-digit confirmation code from your app
                </label>
                <input
                  type="text"
                  placeholder="123456"
                  value={verifyCode}
                  onChange={e => setVerifyCode(e.target.value)}
                  maxLength={8}
                  autoFocus
                  required
                  style={{ width: '100%', marginBottom: '16px' }}
                />
                <div style={{ display: 'flex', justifyContent: 'flex-end', gap: '10px' }}>
                  <button type="button" className="ghost" onClick={() => setModalType(null)}>Cancel</button>
                  <button type="submit" className="primary" disabled={loading}>
                    {loading ? 'Verifying…' : 'Activate 2FA'}
                  </button>
                </div>
              </form>
            </div>
          </div>
        </div>
      )}

      {/* 2FA Disable Modal */}
      {modalType === 'disable' && (
        <div className="overlay" onClick={e => { if (e.target === e.currentTarget) setModalType(null) }}>
          <div className="modal" style={{ maxWidth: '440px' }}>
            <div className="modal-header">
              <h2>Disable Two-Factor Authentication</h2>
              <button className="modal-close" onClick={() => setModalType(null)} aria-label="Close"><XIcon /></button>
            </div>
            <form onSubmit={handleConfirmDisable} className="modal-body" style={{ display: 'flex', flexDirection: 'column', gap: '14px' }}>
              {modalError && <div className="login-error" role="alert"><p>{modalError}</p></div>}
              <p style={{ color: 'var(--muted)', fontSize: '13px' }}>
                Disabling 2FA will lower your account security. Please verify your password and 2FA code to confirm.
              </p>
              <div>
                <label style={{ display: 'block', fontSize: '12px', marginBottom: '4px' }}>Your account password</label>
                <input
                  type="password"
                  value={passwordInput}
                  onChange={e => setPasswordInput(e.target.value)}
                  placeholder="Enter your current password"
                  required
                  style={{ width: '100%' }}
                />
              </div>
              <div>
                <label style={{ display: 'block', fontSize: '12px', marginBottom: '4px' }}>2FA code or recovery code</label>
                <input
                  type="text"
                  value={disableCode}
                  onChange={e => setDisableCode(e.target.value)}
                  placeholder="123456 or recovery code"
                  required
                  style={{ width: '100%' }}
                />
              </div>
              <div style={{ display: 'flex', justifyContent: 'flex-end', gap: '10px', marginTop: '10px' }}>
                <button type="button" className="ghost" onClick={() => setModalType(null)}>Cancel</button>
                <button type="submit" className="danger" disabled={loading}>
                  {loading ? 'Disabling…' : 'Disable 2FA'}
                </button>
              </div>
            </form>
          </div>
        </div>
      )}

      {/* Regenerate Recovery Codes Modal */}
      {modalType === 'regen' && (
        <div className="overlay" onClick={e => { if (e.target === e.currentTarget) setModalType(null) }}>
          <div className="modal" style={{ maxWidth: '480px', maxHeight: '90vh', overflowY: 'auto' }}>
            <div className="modal-header">
              <h2>{newRecoveryCodes ? 'Your new recovery codes' : 'Regenerate recovery codes'}</h2>
              <button className="modal-close" onClick={() => setModalType(null)} aria-label="Close"><XIcon /></button>
            </div>
            <div className="modal-body">
              {modalError && <div className="login-error" role="alert" style={{ marginBottom: '12px' }}><p>{modalError}</p></div>}
              {!newRecoveryCodes ? (
                <form onSubmit={handleConfirmRegen} style={{ display: 'flex', flexDirection: 'column', gap: '14px' }}>
                  <p style={{ color: 'var(--muted)', fontSize: '13px' }}>
                    Generating new recovery codes will invalidate all existing codes. Please confirm your password to continue.
                  </p>
                  <div>
                    <label style={{ display: 'block', fontSize: '12px', marginBottom: '4px' }}>Account password</label>
                    <input
                      type="password"
                      value={passwordInput}
                      onChange={e => setPasswordInput(e.target.value)}
                      placeholder="Enter password"
                      required
                      style={{ width: '100%' }}
                    />
                  </div>
                  <div style={{ display: 'flex', justifyContent: 'flex-end', gap: '10px', marginTop: '10px' }}>
                    <button type="button" className="ghost" onClick={() => setModalType(null)}>Cancel</button>
                    <button type="submit" className="primary" disabled={loading}>
                      {loading ? 'Regenerating…' : 'Regenerate codes'}
                    </button>
                  </div>
                </form>
              ) : (
                <div style={{ display: 'flex', flexDirection: 'column', gap: '14px' }}>
                  <p style={{ color: 'var(--muted)', fontSize: '13px' }}>
                    Save these 10 single-use codes now. Old recovery codes are now invalid.
                  </p>
                  <div style={{ display: 'grid', gridTemplateColumns: 'repeat(2, 1fr)', gap: '6px', background: 'rgba(0,0,0,0.25)', padding: '12px', borderRadius: '6px' }}>
                    {newRecoveryCodes.map(c => <code key={c} style={{ fontSize: '12px', fontFamily: 'monospace' }}>{c}</code>)}
                  </div>
                  <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginTop: '6px' }}>
                    <button type="button" className="ghost" onClick={() => copyCodes(newRecoveryCodes)}>
                      {codesCopied ? 'Copied!' : 'Copy all codes'}
                    </button>
                    <button type="button" className="primary" onClick={() => setModalType(null)}>
                      Done
                    </button>
                  </div>
                </div>
              )}
            </div>
          </div>
        </div>
      )}
    </div>
  )
}

function ProfileLink({ to, icon, title, description }: { to: string; icon: IconName; title: string; description: string }) {
  return <Link to={to} className="profile-link"><span className="profile-link-icon"><Icon name={icon} /></span><span><strong>{title}</strong><small>{description}</small></span><i><Icon name="arrow" size={14} /></i></Link>
}
