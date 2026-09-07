import { useState } from 'react'
import { useNavigate, useLocation } from 'react-router-dom'
import { api } from '../api'
import './Login.css'

type IconName = 'user' | 'lock' | 'eye' | 'eyeOff' | 'arrow' | 'alert' | 'shield' | 'package' | 'check'

function Icon({ name, size = 16 }: { name: IconName; size?: number }) {
  const paths: Record<IconName, React.ReactNode> = {
    user: <><circle cx="12" cy="8" r="4"/><path d="M4 21v-2a6 6 0 0 1 6-6h4a6 6 0 0 1 6 6v2"/></>,
    lock: <><rect x="4" y="10" width="16" height="11" rx="2"/><path d="M8 10V7a4 4 0 0 1 8 0v3"/></>,
    eye: <><path d="M2 12s3.5-6 10-6 10 6 10 6-3.5 6-10 6S2 12 2 12Z"/><circle cx="12" cy="12" r="2.5"/></>,
    eyeOff: <><path d="m3 3 18 18M10.6 6.2A10.5 10.5 0 0 1 12 6c6.5 0 10 6 10 6a17 17 0 0 1-2.1 2.7M6.5 6.5C3.6 8.2 2 12 2 12s3.5 6 10 6c1.4 0 2.7-.3 3.8-.7"/></>,
    arrow: <><path d="M5 12h14M14 7l5 5-5 5"/></>,
    alert: <><circle cx="12" cy="12" r="10"/><path d="M12 8v4M12 16h.01"/></>,
    shield: <><path d="M12 22s8-4 8-10V5l-8-3-8 3v7c0 6 8 10 8 10Z"/><path d="m9 12 2 2 4-4"/></>,
    package: <><path d="M3 8h18v13H3zM1 3h22v5H1zM10 13h4"/></>,
    check: <path d="m5 12 4 4L19 6"/>,
  }
  return <svg width={size} height={size} viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round" aria-hidden="true">{paths[name]}</svg>
}

function AptifyMark() {
  return <svg className="login-logo-mark" viewBox="0 0 64 64" role="img" aria-label="Aptify"><rect width="64" height="64" rx="14" fill="#0f1115"/><rect x="9" y="9" width="46" height="46" rx="10" fill="#191d25" stroke="#303744" strokeWidth="2"/><path d="M18 24h28v20H18z" fill="#202631" stroke="#5eead4" strokeWidth="3" strokeLinejoin="round"/><path d="M16 18h32v9H16z" fill="#14b8a6" stroke="#5eead4" strokeWidth="3" strokeLinejoin="round"/><path d="M27 34h10" stroke="#f59e0b" strokeWidth="4" strokeLinecap="round"/><path d="M23 18l4-6h10l4 6" fill="none" stroke="#f59e0b" strokeWidth="3" strokeLinecap="round" strokeLinejoin="round"/></svg>
}

export default function Login() {
  const [username, setUsername] = useState('')
  const [password, setPassword] = useState('')
  const [showPassword, setShowPassword] = useState(false)
  const [error, setError] = useState('')
  const [loading, setLoading] = useState(false)
  const [step, setStep] = useState<'credentials' | '2fa'>('credentials')
  const [preAuthToken, setPreAuthToken] = useState('')
  const [twoFactorCode, setTwoFactorCode] = useState('')
  const [useRecovery, setUseRecovery] = useState(false)
  const navigate = useNavigate()
  const location = useLocation()

  const handleCredentialsSubmit = async (e: React.FormEvent) => {
    e.preventDefault()
    setLoading(true)
    setError('')
    try {
      const res = await api.login(username, password)
      if (res.status === '2fa_required' && res.pre_auth_token) {
        setPreAuthToken(res.pre_auth_token)
        setStep('2fa')
        return
      }
      const from = (location.state as { from?: { pathname?: string } })?.from?.pathname || '/'
      navigate(from, { replace: true })
    } catch {
      setError('The username or password you entered is incorrect.')
    } finally {
      setLoading(false)
    }
  }

  const handle2FASubmit = async (e: React.FormEvent) => {
    e.preventDefault()
    setLoading(true)
    setError('')
    try {
      await api.verify2FA(preAuthToken, twoFactorCode)
      const from = (location.state as { from?: { pathname?: string } })?.from?.pathname || '/'
      navigate(from, { replace: true })
    } catch (err: unknown) {
      setError(err instanceof Error ? err.message : 'Invalid verification code')
    } finally {
      setLoading(false)
    }
  }

  return <main className="login-root">
    <div className="login-atmosphere" aria-hidden="true"><span/><span/><span/></div>

    <section className="login-story" aria-label="About Aptify">
      <div className="login-brand">
        <AptifyMark />
        <div><strong>Aptify</strong><span>Package control plane</span></div>
      </div>

      <div className="login-story-copy">
        <div className="story-kicker"><span />Self-hosted by design</div>
        <h1>Private packages.<br/><em>Beautifully controlled.</em></h1>
        <p>One secure home for publishing, signing, and distributing the software your teams depend on.</p>
      </div>

      <div className="package-flow" aria-hidden="true">
        <div className="flow-glow" />
        <div className="flow-track"><span/><span/><span/><span/></div>
        <div className="flow-step flow-source"><i><Icon name="package" size={18}/></i><strong>Package</strong><small>nano_9.0.deb</small></div>
        <div className="flow-step flow-sign"><i><Icon name="shield" size={18}/></i><strong>Sign</strong><small>GPG verified</small></div>
        <div className="flow-step flow-publish"><i><Icon name="check" size={18}/></i><strong>Publish</strong><small>Ready to install</small></div>
      </div>

      <div className="story-proof">
        <div><strong>DEB + RPM</strong><span>One control plane</span></div>
        <div><strong>GPG signed</strong><span>Trusted releases</span></div>
        <div><strong>Self-hosted</strong><span>Your infrastructure</span></div>
      </div>
    </section>

    <section className="login-access">
      <div className="access-topline"><span><i/>Secure administrator access</span><small>APTIFY / AUTH</small></div>
      <div className="login-card">
        {step === 'credentials' ? (
          <>
            <div className="login-heading">
              <div className="login-heading-icon"><Icon name="lock" size={19}/></div>
              <span>Welcome back</span>
              <h2>Sign in to Aptify</h2>
              <p>Enter your workspace credentials to continue.</p>
            </div>

            {error && <div className="login-error" role="alert"><Icon name="alert"/><span><strong>Sign-in failed</strong>{error}</span></div>}

            <form onSubmit={handleCredentialsSubmit} className="login-form">
              <label className="login-field" htmlFor="username">
                <span>Username</span>
                <div className="login-input"><Icon name="user"/><input id="username" type="text" value={username} onChange={e => setUsername(e.target.value)} placeholder="Enter your username" autoComplete="username" autoCapitalize="none" spellCheck={false} autoFocus required /></div>
              </label>
              <label className="login-field" htmlFor="password">
                <span>Password</span>
                <div className="login-input"><Icon name="lock"/><input id="password" type={showPassword ? 'text' : 'password'} value={password} onChange={e => setPassword(e.target.value)} placeholder="Enter your password" autoComplete="current-password" required /><button type="button" onClick={() => setShowPassword(show => !show)} aria-label={showPassword ? 'Hide password' : 'Show password'}><Icon name={showPassword ? 'eyeOff' : 'eye'}/></button></div>
              </label>
              <button type="submit" className="login-submit" disabled={loading}>
                {loading ? <><span className="spinner login-spinner"/>Verifying credentials…</> : <><span>Continue to workspace</span><i><Icon name="arrow" size={15}/></i></>}
              </button>
            </form>
          </>
        ) : (
          <>
            <div className="login-heading">
              <div className="login-heading-icon"><Icon name="shield" size={19}/></div>
              <span>Two-factor challenge</span>
              <h2>Two-factor authentication</h2>
              <p>{useRecovery ? 'Enter an emergency single-use recovery code.' : 'Enter the 6-digit code from your authenticator app.'}</p>
            </div>

            {error && <div className="login-error" role="alert"><Icon name="alert"/><span><strong>Verification failed</strong>{error}</span></div>}

            <form onSubmit={handle2FASubmit} className="login-form">
              <label className="login-field" htmlFor="twofactor">
                <span>{useRecovery ? 'Recovery code' : '6-digit authentication code'}</span>
                <div className="login-input">
                  <Icon name="lock"/>
                  <input
                    id="twofactor"
                    type="text"
                    value={twoFactorCode}
                    onChange={e => setTwoFactorCode(e.target.value)}
                    placeholder={useRecovery ? 'xxxx-xxxx' : '123456'}
                    autoFocus
                    required
                    maxLength={useRecovery ? 16 : 8}
                    autoComplete="one-time-code"
                  />
                </div>
              </label>
              <button type="submit" className="login-submit" disabled={loading}>
                {loading ? <><span className="spinner login-spinner"/>Verifying code…</> : <><span>Verify and sign in</span><i><Icon name="arrow" size={15}/></i></>}
              </button>
              <div style={{ display: 'flex', justifyContent: 'space-between', marginTop: '0.75rem', fontSize: '0.85rem' }}>
                <button
                  type="button"
                  style={{ background: 'none', border: 'none', color: '#5eead4', cursor: 'pointer', padding: 0 }}
                  onClick={() => { setUseRecovery(prev => !prev); setError(''); setTwoFactorCode('') }}
                >
                  {useRecovery ? 'Use authenticator app code' : 'Use a recovery code'}
                </button>
                <button
                  type="button"
                  style={{ background: 'none', border: 'none', color: '#94a3b8', cursor: 'pointer', padding: 0 }}
                  onClick={() => { setStep('credentials'); setError(''); setTwoFactorCode('') }}
                >
                  Back to login
                </button>
              </div>
            </form>
          </>
        )}

        <div className="login-security"><Icon name="shield" size={14}/><span>Your credentials are sent securely to your Aptify server.</span></div>
      </div>
      <div className="login-foot"><span>Aptify control plane</span><i/> <span>Private by default</span></div>
    </section>
  </main>
}
