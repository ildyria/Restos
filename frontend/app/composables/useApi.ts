import type { User } from '~~/shared/types'

export interface ApiErrorShape { status: number, message: string }

/** Pulls a readable message out of a $fetch error (our API replies {"error": "..."}). */
export function errorMessage(e: unknown): string {
  const err = e as { data?: { error?: string }, statusCode?: number, statusMessage?: string, message?: string }
  if (err?.data?.error) return err.data.error
  // Our API answers oversized uploads with a 400; a 413 comes from a proxy in front of it.
  if (err?.statusCode === 413) return 'The file is too large for the server to accept (rejected before reaching the site; check the reverse proxy\'s upload size limit).'
  return err?.statusMessage || err?.message || 'Something went wrong'
}

/** Mutating call helper: same-origin cookies plus the CSRF header the API requires. */
export function api<T>(path: string, opts: { method?: 'GET' | 'POST' | 'PUT' | 'DELETE', body?: unknown } = {}) {
  return $fetch<T>(path, {
    method: opts.method ?? 'GET',
    body: opts.body as Record<string, unknown> | FormData | undefined,
    headers: { 'X-Requested-With': 'restos' }
  })
}

export function mediaUrl(key: string | null | undefined, kind: 'thumb' | 'full' = 'thumb') {
  return key ? `/media/${kind}/${key}.jpg` : undefined
}

export function useAuth() {
  const user = useState<User | null>('auth-user', () => null)
  const checked = useState('auth-checked', () => false)

  async function refresh() {
    try {
      user.value = await api<User>('/api/auth/me')
    } catch {
      user.value = null
    }
    checked.value = true
    return user.value
  }

  async function login(email: string, password: string) {
    user.value = await api<User>('/api/auth/login', { method: 'POST', body: { email, password } })
  }

  async function register(input: { name: string, email: string, password: string }) {
    user.value = await api<User>('/api/auth/register', { method: 'POST', body: input })
    checked.value = true
  }

  async function logout() {
    await api('/api/auth/logout', { method: 'POST' })
    user.value = null
  }

  return { user, checked, refresh, login, register, logout }
}

/** Whether public sign-up is open (RESTOS_REGISTRATION on the server). */
export function useAuthConfig() {
  return useFetch<{ registration: boolean }>('/api/auth/config', { key: 'auth-config', default: () => ({ registration: false }) })
}
