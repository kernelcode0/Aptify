import { useState } from 'react'
import { useNavigate, useLocation } from 'react-router-dom'
import { setToken, api } from '../api'

export default function Login() {
  const [username, setUsername] = useState('')
  const [password, setPassword] = useState('')
  const [error, setError] = useState('')
  const [loading, setLoading] = useState(false)
  const navigate = useNavigate()
  const location = useLocation()

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault()
    setLoading(true)
    setError('')
    try {
      const res = await api.login(username, password)
      setToken(res.token)
      
      const from = location.state?.from?.pathname || "/"
      navigate(from, { replace: true })
    } catch (e: unknown) {
      setToken('')
      setError('Invalid username or password.')
    } finally {
      setLoading(false)
    }
  }

  return (
    <div style={{ display: 'flex', justifyContent: 'center', alignItems: 'center', height: '100vh', backgroundColor: 'var(--bg)' }}>
      <div className="card" style={{ maxWidth: 400, width: '100%' }}>
        <h2 style={{ marginTop: 0, marginBottom: 8 }}>Admin Login</h2>
        <p className="muted-text" style={{ marginBottom: 24, fontSize: 14 }}>
          This APT repository is protected. Please sign in.
        </p>
        
        {error && <div className="alert-error" style={{ marginBottom: 16 }}>{error}</div>}
        
        <form onSubmit={handleSubmit}>
          <div className="field">
            <label>Username</label>
            <input 
              type="text" 
              value={username} 
              onChange={e => setUsername(e.target.value)} 
              placeholder="Username" 
              autoFocus
              required 
            />
          </div>
          <div className="field" style={{ marginTop: 12 }}>
            <label>Password</label>
            <input 
              type="password" 
              value={password} 
              onChange={e => setPassword(e.target.value)} 
              placeholder="Password" 
              required 
            />
          </div>
          <button type="submit" className="primary" style={{ width: '100%', marginTop: 8 }} disabled={loading}>
            {loading ? 'Authenticating...' : 'Login'}
          </button>
        </form>
      </div>
    </div>
  )
}
