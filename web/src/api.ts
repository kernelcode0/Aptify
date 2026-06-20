// getToken/setToken are preserved as no-ops for API key upload path compatibility.
// No token is stored in JavaScript — localStorage was removed to prevent XSS theft.
export function getToken(): string {
  return ''
}

export function setToken(_t: string) {
  // no-op
}

async function req<T>(method: string, path: string, body?: unknown): Promise<T> {
  const token = getToken()
  const headers: Record<string, string> = {}
  if (token) headers['Authorization'] = `Bearer ${token}`
  if (body) headers['Content-Type'] = 'application/json'

  const res = await fetch(path, {
    method,
    headers,
    credentials: 'same-origin', // Send HttpOnly session cookie
    body: body ? JSON.stringify(body) : undefined,
  })
  if (!res.ok) {
    if (res.status === 401 && window.location.pathname !== '/login') {
      window.location.href = '/login'
    }
    const err = await res.json().catch(() => ({ error: res.statusText }))
    if (res.status === 403) {
      throw new Error('You do not have permission to perform this action.')
    }
    throw new Error(err.error ?? res.statusText)
  }
  if (res.status === 204) return undefined as unknown as T
  return res.json()
}

export interface Repo {
  id: string
  slug: string
  name: string
  codename: string
  type: 'deb' | 'rpm'
  created_at: string
}

export interface Package {
  id: string
  repo_id: string
  filename: string
  package: string
  version: string
  release?: string
  arch: string
  size: number
  sha256: string
  uploaded_at: string
}

export interface SetupInfo {
  type: 'deb' | 'rpm'
  keyURL: string
  repoURL: string
  codename?: string
  component?: string
  addKey?: string
  addSource?: string
  install?: string
  repoFile?: string
  update: string
}

export interface PackageList {
  packages: Package[]
  total: number
}

export interface APIKey {
  id: string
  name: string
  prefix: string
  created_at: string
  last_used: string | null
}

export interface APIKeyCreated extends APIKey {
  key: string // only present on creation response
}

export interface RepoStatus {
  id: string
  indexing: boolean
  last_indexed: string | null
}

export type Role = 'admin' | 'member' | 'viewer'

export interface CurrentUser {
  valid: boolean
  user_id: string
  username: string
  role: Role
}

export interface User {
  id: string
  username: string
  role: Role
  created_at: string
}

export interface AuditEntry {
  id: string
  user_id: string
  username: string
  action: string
  resource: string
  detail: string
  created_at: string
}

export interface AuditList {
  total: number
  entries: AuditEntry[]
}

export const api = {
  checkAuth: () => req<CurrentUser>('GET', '/api/auth/check'),
  login: (username: string, password: string) => req<{ token: string }>('POST', '/api/auth/login', { username, password }),
  listRepos: () => req<Repo[]>('GET', '/api/repos'),
  createRepo: (slug: string, name: string, codename: string, type: Repo['type']) =>
    req<Repo>('POST', '/api/repos', { slug, name, codename, type }),
  updateRepo: (id: string, name: string, codename: string) =>
    req<Repo>('PUT', `/api/repos/${id}`, { name, codename }),
  deleteRepo: (id: string) => req<void>('DELETE', `/api/repos/${id}`),
  listPackages: (repoId: string, page: number = 1, limit: number = 50) =>
    req<PackageList>('GET', `/api/repos/${repoId}/packages?page=${page}&limit=${limit}`),
  deletePackage: (repoId: string, pkgId: string) =>
    req<void>('DELETE', `/api/repos/${repoId}/packages/${pkgId}`),
  getSetup: (repoId: string) => req<SetupInfo>('GET', `/api/repos/${repoId}/setup`),
  getRepoStatus: (repoId: string) => req<RepoStatus>('GET', `/api/repos/${repoId}/status`),
  uploadPackage: async (repoId: string, file: File) => {
    const token = getToken()
    const form = new FormData()
    form.append('file', file)
    const res = await fetch(`/api/repos/${repoId}/packages`, {
      method: 'POST',
      headers: token ? { Authorization: `Bearer ${token}` } : {},
      credentials: 'same-origin', // Send HttpOnly session cookie
      body: form,
    })
    if (!res.ok) {
      if (res.status === 401 && window.location.pathname !== '/login') {
        window.location.href = '/login'
      }
      const err = await res.json().catch(() => ({ error: res.statusText }))
      if (res.status === 403) {
        throw new Error('You do not have permission to perform this action.')
      }
      throw new Error(err.error ?? res.statusText)
    }
    return res.json() as Promise<Package>
  },
  listAPIKeys: () => req<APIKey[]>('GET', '/api/auth/keys'),
  createAPIKey: (name: string) => req<APIKeyCreated>('POST', '/api/auth/keys', { name }),
  deleteAPIKey: (id: string) => req<void>('DELETE', `/api/auth/keys/${id}`),
  listUsers: () => req<User[]>('GET', '/api/users'),
  createUser: (username: string, password: string, role: Role) =>
    req<User>('POST', '/api/users', { username, password, role }),
  updateUser: (id: string, data: { role?: Role; password?: string }) =>
    req<User>('PUT', `/api/users/${id}`, data),
  deleteUser: (id: string) => req<void>('DELETE', `/api/users/${id}`),
  listAudit: (offset: number = 0, limit: number = 50) =>
    req<AuditList>('GET', `/api/audit?offset=${offset}&limit=${limit}`),
  clearAudit: () => req<void>('DELETE', '/api/audit'),
}
