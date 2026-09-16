import { useState, useEffect } from 'react'
import { useNavigate, useLocation, useSearchParams } from 'react-router-dom'
import { api } from '../api'
import './Login.css'

type IconName = 'user' | 'lock' | 'eye' | 'eyeOff' | 'arrow' | 'alert' | 'shield' | 'package' | 'check' | 'capsLock' | 'x'

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
    capsLock: <><path d="M12 3.5 5.5 10h3.5v6.5h6V10h3.5L12 3.5Z" fill="currentColor" fillOpacity="0.25"/><line x1="5.5" y1="19.5" x2="18.5" y2="19.5"/></>,
    x: <><path d="M18 6 6 18M6 6l12 12"/></>,
  }
  return <svg width={size} height={size} viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round" aria-hidden="true">{paths[name]}</svg>
}

function AptifyMark() {
  return <svg className="login-logo-mark" viewBox="0 0 64 64" role="img" aria-label="Aptify"><rect width="64" height="64" rx="14" fill="#0f1115"/><rect x="9" y="9" width="46" height="46" rx="10" fill="#191d25" stroke="#303744" strokeWidth="2"/><path d="M18 24h28v20H18z" fill="#202631" stroke="#5eead4" strokeWidth="3" strokeLinejoin="round"/><path d="M16 18h32v9H16z" fill="#14b8a6" stroke="#5eead4" strokeWidth="3" strokeLinejoin="round"/><path d="M27 34h10" stroke="#f59e0b" strokeWidth="4" strokeLinecap="round"/><path d="M23 18l4-6h10l4 6" fill="none" stroke="#f59e0b" strokeWidth="3" strokeLinecap="round" strokeLinejoin="round"/></svg>
}

function GitHubIcon({ size = 13 }: { size?: number }) {
  return (
    <svg width={size} height={size} viewBox="0 0 24 24" fill="currentColor" aria-hidden="true">
      <path d="M12 2a10 10 0 0 0-3.16 19.49c.5.09.68-.22.68-.48v-1.87c-2.78.6-3.37-1.18-3.37-1.18-.45-1.16-1.11-1.47-1.11-1.47-.91-.62.07-.61.07-.61 1 .07 1.53 1.03 1.53 1.03.9 1.53 2.35 1.09 2.92.83.09-.65.35-1.09.64-1.34-2.22-.25-4.56-1.11-4.56-4.94 0-1.09.39-1.98 1.03-2.68-.1-.25-.45-1.27.1-2.64 0 0 .84-.27 2.75 1.02A9.56 9.56 0 0 1 12 6.82c.85 0 1.71.12 2.51.34 1.91-1.29 2.75-1.02 2.75-1.02.55 1.37.2 2.39.1 2.64.64.7 1.03 1.59 1.03 2.68 0 3.84-2.34 4.68-4.57 4.93.36.31.68.92.68 1.85v2.77c0 .27.18.58.69.48A10 10 0 0 0 12 2Z"/>
    </svg>
  )
}

export default function Login() {
  const [username, setUsername] = useState('')
  const [password, setPassword] = useState('')
  const [showPassword, setShowPassword] = useState(false)
  const [capsLock, setCapsLock] = useState(false)
  const [serverHealthy, setServerHealthy] = useState<boolean | null>(null)
  const [error, setError] = useState('')
  const [loading, setLoading] = useState(false)
  const [searchParams] = useSearchParams()
  const [step, setStep] = useState<'credentials' | '2fa'>(
    searchParams.get('step') === '2fa' ? '2fa' : 'credentials'
  )
  const [preAuthToken, setPreAuthToken] = useState('')
  const [twoFactorCode, setTwoFactorCode] = useState('')
  const [useRecovery, setUseRecovery] = useState(false)
  const [rememberDevice, setRememberDevice] = useState(true)
  const [activeStepTooltip, setActiveStepTooltip] = useState<'source' | 'sign' | 'publish' | null>(null)
  const navigate = useNavigate()
  const location = useLocation()

  useEffect(() => {
    fetch('/health')
      .then(res => setServerHealthy(res.ok))
      .catch(() => setServerHealthy(false))
  }, [])

  useEffect(() => {
    const handleKeyDown = (e: KeyboardEvent) => {
      if (e.key === 'Escape' && step === '2fa') {
        setStep('credentials')
        setError('')
        setTwoFactorCode('')
      }
    }
    window.addEventListener('keydown', handleKeyDown)
    return () => window.removeEventListener('keydown', handleKeyDown)
  }, [step])

  const checkCapsLock = (e: React.KeyboardEvent<HTMLInputElement>) => {
    setCapsLock(e.getModifierState('CapsLock'))
  }

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

  const verify2FACode = async (codeToVerify: string) => {
    setLoading(true)
    setError('')
    try {
      await api.verify2FA(preAuthToken, codeToVerify, rememberDevice)
      const from = (location.state as { from?: { pathname?: string } })?.from?.pathname || '/'
      navigate(from, { replace: true })
    } catch (err: unknown) {
      setError(err instanceof Error ? err.message : 'Invalid verification code')
    } finally {
      setLoading(false)
    }
  }

  const handle2FASubmit = (e: React.FormEvent) => {
    e.preventDefault()
    verify2FACode(twoFactorCode)
  }

  const handle2FACodeChange = (val: string) => {
    if (useRecovery) {
      setTwoFactorCode(val)
      return
    }
    const cleaned = val.replace(/[^0-9]/g, '').slice(0, 6)
    setTwoFactorCode(cleaned)
  }

  return <main className="login-root">
    <div className="login-atmosphere" aria-hidden="true"><span/><span/><span/></div>

    <section className="login-story" aria-label="About Aptify">
      <div className="login-brand">
        <AptifyMark />
        <div>
          <div className="login-brand-name">
            <strong>Aptify</strong>
            {serverHealthy !== null && (
              <span className={`brand-status-badge ${serverHealthy ? 'status-online' : 'status-offline'}`} title={serverHealthy ? 'Daemon online' : 'Daemon unreachable'}>
                <i />
                <span>{serverHealthy ? 'Online' : 'Offline'}</span>
              </span>
            )}
          </div>
          <span>Package control plane</span>
        </div>
      </div>

      <div className="login-story-copy">
        <div className="story-kicker"><span />Self-hosted by design</div>
        <h1>Private packages.<br/><em>Beautifully controlled.</em></h1>
        <p>One secure home for publishing, signing, and distributing the software your teams depend on.</p>
      </div>

      <div className="package-flow">
        <div className="flow-glow" />
        <div
          className={`flow-step flow-source ${activeStepTooltip === 'source' ? 'is-hovered' : ''}`}
          onMouseEnter={() => setActiveStepTooltip('source')}
          onMouseLeave={() => setActiveStepTooltip(null)}
          tabIndex={0}
          role="button"
          aria-label="Package source guarantee"
        >
          <i><Icon name="package" size={18}/></i>
          <div className="flow-step-text">
            <strong>Package</strong>
            <small>nano_9.0.deb</small>
          </div>
          {activeStepTooltip === 'source' && (
            <div className="flow-tooltip" role="tooltip">
              <span className="tooltip-title">Ingestion Guarantee</span>
              <span className="tooltip-desc">Format validation & SHA256 checksumming</span>
            </div>
          )}
        </div>
        <div className="flow-connector connector-1">
          <div className="connector-track">
            <div className="connector-beam" />
          </div>
        </div>
        <div
          className={`flow-step flow-sign ${activeStepTooltip === 'sign' ? 'is-hovered' : ''}`}
          onMouseEnter={() => setActiveStepTooltip('sign')}
          onMouseLeave={() => setActiveStepTooltip(null)}
          tabIndex={0}
          role="button"
          aria-label="Cryptographic signature guarantee"
        >
          <i><Icon name="shield" size={18}/></i>
          <div className="flow-step-text">
            <strong>Sign</strong>
            <small>GPG verified</small>
          </div>
          {activeStepTooltip === 'sign' && (
            <div className="flow-tooltip" role="tooltip">
              <span className="tooltip-title">Chain of Trust</span>
              <span className="tooltip-desc">InRelease & Release.gpg signing</span>
            </div>
          )}
        </div>
        <div className="flow-connector connector-2">
          <div className="connector-track">
            <div className="connector-beam" />
          </div>
        </div>
        <div
          className={`flow-step flow-publish ${activeStepTooltip === 'publish' ? 'is-hovered' : ''}`}
          onMouseEnter={() => setActiveStepTooltip('publish')}
          onMouseLeave={() => setActiveStepTooltip(null)}
          tabIndex={0}
          role="button"
          aria-label="Distribution guarantee"
        >
          <i><Icon name="check" size={18}/></i>
          <div className="flow-step-text">
            <strong>Publish</strong>
            <small>Ready to install</small>
          </div>
          {activeStepTooltip === 'publish' && (
            <div className="flow-tooltip" role="tooltip">
              <span className="tooltip-title">Atomic Distribution</span>
              <span className="tooltip-desc">Zero-downtime pool index updates</span>
            </div>
          )}
        </div>
      </div>

      <div className="story-proof">
        <div><strong>DEB + RPM</strong><span>One control plane</span></div>
        <div><strong>GPG signed</strong><span>Trusted releases</span></div>
        <div><strong>Self-hosted</strong><span>Your infrastructure</span></div>
      </div>
    </section>

    <section className="login-access">
      <div className="access-topline">
        <span><i/>Secure administrator access</span>
        <a
          href="https://github.com/kernelcode0"
          target="_blank"
          rel="noopener noreferrer"
          className="author-pill"
          title="kernelcode0 on GitHub"
        >
          <GitHubIcon size={12} />
          <span>@kernelcode0</span>
        </a>
      </div>
      <div className="login-card">
        {step === 'credentials' ? (
          <>
            <div className="login-heading">
              <div className="login-heading-icon"><Icon name="lock" size={22}/></div>
              <div className="login-kicker"><span className="login-kicker-dot"/>Secure Workspace</div>
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
                <div className="login-input">
                  <Icon name="lock"/>
                  <input
                    id="password"
                    type={showPassword ? 'text' : 'password'}
                    value={password}
                    onChange={e => setPassword(e.target.value)}
                    onKeyDown={checkCapsLock}
                    onKeyUp={checkCapsLock}
                    onBlur={() => setCapsLock(false)}
                    placeholder="Enter your password"
                    autoComplete="current-password"
                    required
                  />
                  <button type="button" onClick={() => setShowPassword(show => !show)} aria-label={showPassword ? 'Hide password' : 'Show password'} tabIndex={-1}><Icon name={showPassword ? 'eyeOff' : 'eye'}/></button>
                </div>
                {capsLock && (
                  <div className="caps-lock-badge" role="status" aria-live="polite">
                    <span className="caps-lock-icon-wrap">
                      <Icon name="capsLock" size={11}/>
                    </span>
                    <span>Caps Lock is on</span>
                  </div>
                )}
              </label>
              <button type="submit" className="login-submit" disabled={loading}>
                {loading ? <><span className="spinner login-spinner"/>Verifying credentials…</> : <><span>Continue to workspace</span><i><Icon name="arrow" size={15}/></i></>}
              </button>
            </form>
          </>
        ) : (
          <>
            <div className="login-heading">
              <div className="login-heading-icon"><Icon name="shield" size={22}/></div>
              <div className="login-kicker"><span className="login-kicker-dot"/>Two-Factor Challenge</div>
              <h2>Two-factor authentication</h2>
              <p>{useRecovery ? 'Enter an emergency single-use recovery code.' : 'Enter the 6-digit code from your authenticator app.'}</p>
            </div>

            {error && <div className="login-error" role="alert"><Icon name="alert"/><span><strong>Verification failed</strong>{error}</span></div>}

            <form onSubmit={handle2FASubmit} className="login-form">
              <label className="login-field" htmlFor="twofactor">
                <span>{useRecovery ? 'Recovery code' : '6-digit authentication code'}</span>
                <div className="login-input login-input-2fa">
                  <input
                    id="twofactor"
                    type="text"
                    inputMode={useRecovery ? 'text' : 'numeric'}
                    pattern={useRecovery ? undefined : '[0-9]*'}
                    className={useRecovery ? 'input-recovery' : 'input-totp'}
                    value={twoFactorCode}
                    onChange={e => handle2FACodeChange(e.target.value)}
                    placeholder={useRecovery ? 'xxxx-xxxx' : '000000'}
                    autoFocus
                    required
                    maxLength={useRecovery ? 16 : 6}
                    autoComplete="one-time-code"
                    spellCheck={false}
                  />
                  {twoFactorCode.length > 0 && (
                    <button
                      type="button"
                      className="btn-clear-totp"
                      onClick={() => { setTwoFactorCode(''); setError('') }}
                      aria-label="Clear code"
                      tabIndex={-1}
                    >
                      <Icon name="x" size={13} />
                    </button>
                  )}
                  {!useRecovery && (
                    <div className="totp-slots" aria-hidden="true">
                      {[0, 1, 2, 3, 4, 5].map(idx => (
                        <span
                          key={idx}
                          className={`totp-slot ${twoFactorCode.length > idx ? 'filled' : ''} ${twoFactorCode.length === idx ? 'active' : ''}`}
                        />
                      ))}
                    </div>
                  )}
                </div>
                {!useRecovery && twoFactorCode.length === 6 && (
                  <div className="totp-indicator-row" aria-hidden="true">
                    <span className="totp-ready-hint">Ready — Press Enter ↵</span>
                  </div>
                )}
              </label>

              <label className="login-checkbox-field">
                <input
                  type="checkbox"
                  checked={rememberDevice}
                  onChange={e => setRememberDevice(e.target.checked)}
                />
                <span className="custom-checkbox">
                  {rememberDevice && <Icon name="check" size={11} />}
                </span>
                <span>Don't ask again on this device for 30 days</span>
              </label>

              <button
                type="submit"
                className={`login-submit ${twoFactorCode.length === 6 && !loading ? 'btn-ready' : ''}`}
                disabled={loading}
              >
                {loading ? (
                  <><span className="spinner login-spinner"/>Verifying code…</>
                ) : twoFactorCode.length === 6 && !useRecovery ? (
                  <><span>Verify and sign in</span><span className="key-return-hint">↵</span></>
                ) : (
                  <><span>Verify and sign in</span><i><Icon name="arrow" size={15}/></i></>
                )}
              </button>
              <div className="twofactor-links">
                <button
                  type="button"
                  className="btn-link-accent"
                  onClick={() => { setUseRecovery(prev => !prev); setError(''); setTwoFactorCode('') }}
                >
                  {useRecovery ? 'Use authenticator app code' : 'Use emergency recovery code'}
                </button>
                <button
                  type="button"
                  className="btn-link-muted"
                  onClick={() => { setStep('credentials'); setError(''); setTwoFactorCode('') }}
                  title="Press Escape to go back"
                >
                  Back to login <kbd className="btn-kbd-hint">Esc</kbd>
                </button>
              </div>
            </form>
          </>
        )}

        <div className="login-security"><Icon name="shield" size={14}/><span>Your credentials are sent securely to your Aptify server.</span></div>
      </div>
      <div className="login-foot">
        <span>Aptify control plane</span>
        <i/>
        <a
          href="https://github.com/kernelcode0"
          target="_blank"
          rel="noopener noreferrer"
          className="foot-author-link"
        >
          <GitHubIcon size={12} />
          <span>Engineered by <strong>@kernelcode0</strong></span>
        </a>
        <i/>
        <span>Private by default</span>
      </div>
    </section>
  </main>
}
