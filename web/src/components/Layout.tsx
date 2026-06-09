import { Outlet, Link, useLocation, useNavigate } from 'react-router-dom'
import { useEffect, useState } from 'react'
import { api, setToken } from '../api'
import './Layout.css'

function PackageIcon({ size = 18 }: { size?: number }) {
  return (
    <svg width={size} height={size} viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round">
      <path d="M21 16V8a2 2 0 0 0-1-1.73l-7-4a2 2 0 0 0-2 0l-7 4A2 2 0 0 0 3 8v8a2 2 0 0 0 1 1.73l7 4a2 2 0 0 0 2 0l7-4A2 2 0 0 0 21 16z"/>
      <polyline points="3.27 6.96 12 12.01 20.73 6.96"/>
      <line x1="12" y1="22.08" x2="12" y2="12"/>
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
          <span className="nav-brand-icon"><PackageIcon /></span>
          <span className="nav-brand-name">Aptify</span>
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
