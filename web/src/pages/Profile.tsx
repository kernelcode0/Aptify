import { useOutletContext } from 'react-router-dom'
import { type CurrentUser } from '../api'
import './Profile.css'

function UserIcon() {
  return (
    <svg width="48" height="48" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="1.25" strokeLinecap="round" strokeLinejoin="round">
      <path d="M20 21v-2a4 4 0 0 0-4-4H8a4 4 0 0 0-4 4v2"/>
      <circle cx="12" cy="7" r="4"/>
    </svg>
  )
}

export default function Profile() {
  const { user } = useOutletContext<{ user: CurrentUser | null }>()

  const getRoleDisplay = (role: string): string => {
    const roleMap: Record<string, string> = {
      admin: 'Administrator',
      member: 'Member',
      viewer: 'Viewer'
    }
    return roleMap[role] || role
  }

  const getRoleDescription = (role: string): string => {
    const descMap: Record<string, string> = {
      admin: 'Full access to all features including user management and audit logs',
      member: 'Can manage repositories and packages, create API keys',
      viewer: 'Read-only access to repositories and packages'
    }
    return descMap[role] || ''
  }

  return (
    <div>
      <div className="page-header">
        <div>
          <div className="page-kicker">Account</div>
          <h1 className="page-title">Profile</h1>
        </div>
      </div>

      {user && (
        <div className="profile-card card">
          <div className="profile-content">
            <div className="profile-icon">
              <UserIcon />
            </div>
            <div className="profile-info">
              <div className="profile-field">
                <label>Username</label>
                <p className="profile-value">{user.username}</p>
              </div>
              <div className="profile-field">
                <label>Role</label>
                <p className="profile-value">
                  <span className={`role-badge role-${user.role}`}>
                    {getRoleDisplay(user.role)}
                  </span>
                </p>
              </div>
              <div className="profile-field">
                <label>Permissions</label>
                <p className="profile-description">{getRoleDescription(user.role)}</p>
              </div>
            </div>
          </div>
        </div>
      )}
    </div>
  )
}
