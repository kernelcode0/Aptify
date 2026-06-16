import { Outlet, Link, useLocation, useNavigate } from 'react-router-dom'
import { useEffect, useState } from 'react'
import { api, setToken } from '../api'
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

export default function Layout() {
  const loc = useLocation()
  const navigate = useNavigate()
  const [checking, setChecking] = useState(true)

  useEffect(() => {
    api.checkAuth()
      .catch(() => {})
      .finally(() => setChecking(false))
  }, [])

  const handleLogout = () => {
    setToken('')
    navigate('/login')
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

        <div className="nav-right">
          <div className="nav-links">
            <Link to="/" className={loc.pathname === '/' ? 'active' : ''}>Repositories</Link>
          </div>
          <div className="nav-divider" />
          <button className="nav-logout" onClick={handleLogout}>
            <LogOutIcon />
            Sign out
          </button>
        </div>
      </nav>
      <main className="main">
        <Outlet />
      </main>
    </div>
  )
}
