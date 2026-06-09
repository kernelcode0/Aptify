import { Outlet, Link, useLocation } from 'react-router-dom'
import { useEffect, useState } from 'react'
import { api } from '../api'
import './Layout.css'

export default function Layout() {
  const loc = useLocation()
  const [checking, setChecking] = useState(true)

  useEffect(() => {
    api.checkAuth()
      .catch(() => {}) // 401 interceptor handles redirect
      .finally(() => setChecking(false))
  }, [])

  if (checking) {
    return <div style={{ display: 'flex', justifyContent: 'center', alignItems: 'center', height: '100vh', backgroundColor: 'var(--bg)' }}><div className="spinner" /></div>
  }

  return (
    <div className="layout">
      <nav className="nav">
        <Link to="/" className="nav-brand">
          <span className="nav-logo">📦</span>
          <span>apt-repository</span>
        </Link>
        <div className="nav-links">
          <Link to="/" className={loc.pathname === '/' ? 'active' : ''}>Repositories</Link>
        </div>
      </nav>
      <main className="main">
        <Outlet />
      </main>
    </div>
  )
}
