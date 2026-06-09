const TOKEN_KEY = 'apt_admin_token'

export function getToken(): string {
  return localStorage.getItem(TOKEN_KEY) ?? ''
}

export function setToken(t: string) {
  localStorage.setItem(TOKEN_KEY, t)
}

async function req<T>(method: string, path: string, body?: unknown): Promise<T> {
  const token = getToken()
  const headers: Record<string, string> = {}
  if (token) headers['Authorization'] = `Bearer ${token}`
  if (body) headers['Content-Type'] = 'application/json'

  const res = await fetch(path, {
    method,
    headers,
    body: body ? JSON.stringify(body) : undefined,
  })
  if (!res.ok) {
    if (res.status === 401 && window.location.pathname !== '/login') {
      setToken('')
      window.location.href = '/login'
    }
    const err = await res.json().catch(() => ({ error: res.statusText }))
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
  created_at: string
}

export interface Package {
  id: string
  repo_id: string
  filename: string
  package: string
  version: string
  arch: string
  size: number
  sha256: string
  uploaded_at: string
}

export interface SetupInfo {
  keyURL: string
  repoURL: string
  codename: string
  component: string
  addKey: string
  addSource: string
  update: string
}

export interface PackageList {
  packages: Package[]
  total: number
}

export const api = {
  checkAuth: () => req<{ authenticated: boolean }>('GET', '/api/auth/check'),
  login: (username: string, password: string) => req<{ token: string }>('POST', '/api/auth/login', { username, password }),
  listRepos: () => req<Repo[]>('GET', '/api/repos'),
  createRepo: (slug: string, name: string, codename: string) =>
    req<Repo>('POST', '/api/repos', { slug, name, codename }),
  updateRepo: (id: string, name: string, codename: string) =>
    req<Repo>('PUT', `/api/repos/${id}`, { name, codename }),
  deleteRepo: (id: string) => req<void>('DELETE', `/api/repos/${id}`),
  listPackages: (repoId: string, page: number = 1, limit: number = 50) => 
    req<PackageList>('GET', `/api/repos/${repoId}/packages?page=${page}&limit=${limit}`),
  deletePackage: (repoId: string, pkgId: string) =>
    req<void>('DELETE', `/api/repos/${repoId}/packages/${pkgId}`),
  getSetup: (repoId: string) => req<SetupInfo>('GET', `/api/repos/${repoId}/setup`),
  uploadPackage: async (repoId: string, file: File) => {
    const token = getToken()
    const form = new FormData()
    form.append('file', file)
    const res = await fetch(`/api/repos/${repoId}/packages`, {
      method: 'POST',
      headers: token ? { Authorization: `Bearer ${token}` } : {},
      body: form,
    })
    if (!res.ok) {
      if (res.status === 401 && window.location.pathname !== '/login') {
        setToken('')
        window.location.href = '/login'
      }
      const err = await res.json().catch(() => ({ error: res.statusText }))
      throw new Error(err.error ?? res.statusText)
    }
    return res.json() as Promise<Package>
  },
}
