import { Outlet, Link, useLocation, useNavigate } from 'react-router-dom'
import { useEffect, useState } from 'react'
import { api, setToken, type CurrentUser } from '../api'
import './Layout.css'

function AptifyMark() {
  return (
    <svg className="nav-brand-svg" viewBox="0 0 64 64" role="img" aria-label="Aptify">
      <rect width="64" height="64" rx="14" fill="#0f1115" />
      <rect x="9" y="9" width="46" height="46" rx="10" fill="#191d25" stroke="#303744" strokeWidth="2" />
      <path d="M18 24h28v20H18z" fill="#202631" stroke="#5eead4" strokeWidth="3" strokeLinejoin="round" />
      <path d="M16 18h32v9H16z" fill="#14b8a6" stroke="#5eead4" strokeWidth="3" strokeLinejoin="round" />
      <path d="M27 34h10" stroke="#f59e0b" strokeWidth="4" strokeLinecap="round" />
      <path d="M23 18l4-6h10l4 6" fill="none" stroke="#f59e0b" strokeWidth="3" strokeLinecap="round" strokeLinejoin="round" />
    </svg>
  )
}

function LogOutIcon() {
  return (
    <svg width="13" height="13" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round">
      <path d="M9 21H5a2 2 0 0 1-2-2V5a2 2 0 0 1 2-2h4"/>
      <polyline points="16 17 21 12 16 7"/>
      <line x1="21" y1="12" x2="9" y2="12"/>
    </svg>
  )
}

function GitHubIcon() {
  return (
    <svg width="16" height="16" viewBox="0 0 24 24" fill="currentColor" aria-hidden="true">
      <path d="M12 2a10 10 0 0 0-3.16 19.49c.5.09.68-.22.68-.48v-1.87c-2.78.6-3.37-1.18-3.37-1.18-.45-1.16-1.11-1.47-1.11-1.47-.91-.62.07-.61.07-.61 1 .07 1.53 1.03 1.53 1.03.9 1.53 2.35 1.09 2.92.83.09-.65.35-1.09.64-1.34-2.22-.25-4.56-1.11-4.56-4.94 0-1.09.39-1.98 1.03-2.68-.1-.25-.45-1.27.1-2.64 0 0 .84-.27 2.75 1.02A9.56 9.56 0 0 1 12 6.82c.85 0 1.71.12 2.51.34 1.91-1.29 2.75-1.02 2.75-1.02.55 1.37.2 2.39.1 2.64.64.7 1.03 1.59 1.03 2.68 0 3.84-2.34 4.68-4.57 4.93.36.31.68.92.68 1.85v2.77c0 .27.18.58.69.48A10 10 0 0 0 12 2Z"/>
    </svg>
  )
}

const starPromptKey = 'aptify.github-star-prompt.dismissed'

export default function Layout() {
  const loc = useLocation()
  const navigate = useNavigate()
  const [checking, setChecking] = useState(true)
  const [user, setUser] = useState<CurrentUser | null>(null)
  const [menuOpen, setMenuOpen] = useState(false)
  const [showStarPrompt, setShowStarPrompt] = useState(() => {
    try {
      return localStorage.getItem(starPromptKey) !== 'true'
    } catch {
      return true
    }
  })

  useEffect(() => {
    api.checkAuth()
      .then(setUser)
      .catch(() => {})
      .finally(() => setChecking(false))
  }, [])

  useEffect(() => setMenuOpen(false), [loc.pathname])

  const handleLogout = () => {
    setToken('')
    navigate('/login')
  }

  const dismissStarPrompt = () => {
    setShowStarPrompt(false)
    try {
      localStorage.setItem(starPromptKey, 'true')
    } catch {
      // The preference is non-essential when browser storage is unavailable.
    }
  }

  if (checking) {
    return (
      <div className="full-center">
        <div className="spinner" />
      </div>
    )
  }

  return (
    <div className="layout">
      <nav className="nav">
        <Link to="/" className="nav-brand">
          <span className="nav-brand-icon"><AptifyMark /></span>
          <span className="nav-brand-text">
            <span className="nav-brand-name">Aptify</span>
            <span className="nav-brand-tagline">Self-hosted APT, simplified</span>
          </span>
        </Link>

        <button className="nav-toggle" onClick={() => setMenuOpen(open => !open)} aria-expanded={menuOpen} aria-controls="primary-navigation" aria-label="Toggle navigation">
          <span /><span /><span />
        </button>

        <div className={`nav-right ${menuOpen ? 'open' : ''}`} id="primary-navigation">
          <div className="nav-links">
            <Link to="/" className={loc.pathname === '/' ? 'active' : ''}>Repositories</Link>
            {user?.role !== 'viewer' && <Link to="/api-keys" className={loc.pathname === '/api-keys' ? 'active' : ''}>API Keys</Link>}
            {user?.role === 'admin' && <Link to="/users" className={loc.pathname === '/users' ? 'active' : ''}>Users</Link>}
            {user?.role === 'admin' && <Link to="/audit" className={loc.pathname === '/audit' ? 'active' : ''}>Audit Log</Link>}
          </div>
          <div className="nav-divider" />
          <Link to="/profile" className={`nav-link nav-profile ${loc.pathname === '/profile' ? 'active' : ''}`} title={user?.username}>
            <span className="nav-avatar">{user?.username?.slice(0, 1).toUpperCase()}</span>
            <span>{user?.username}</span>
          </Link>
          <button className="nav-logout" onClick={handleLogout}>
            <LogOutIcon />
            Sign out
          </button>
        </div>
      </nav>
      <main className="main">
        {showStarPrompt && (
          <aside className="star-prompt" aria-label="Support Aptify">
            <span className="star-prompt-icon"><GitHubIcon /></span>
            <span className="star-prompt-copy"><strong>Enjoying Aptify?</strong> Help more people discover the project by giving it a star on GitHub.</span>
            <a className="star-prompt-action" href="https://github.com/kernelcode0/Aptify" target="_blank" rel="noreferrer" onClick={dismissStarPrompt}><GitHubIcon />Star on GitHub</a>
            <button className="star-prompt-dismiss" onClick={dismissStarPrompt} aria-label="Dismiss GitHub star suggestion">×</button>
          </aside>
        )}
        <Outlet context={{ user }} />
      </main>
    </div>
  )
}
