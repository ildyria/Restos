import type { User } from '~~/shared/types'

export interface ApiErrorShape { status: number, message: string }

/** Pulls a readable message out of a $fetch error (our API replies {"error": "..."}). */
export function errorMessage(e: unknown): string {
  const err = e as { data?: { error?: string }, statusMessage?: string, message?: string }
  return err?.data?.error || err?.statusMessage || err?.message || 'Something went wrong'
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

  async function logout() {
    await api('/api/auth/logout', { method: 'POST' })
    user.value = null
  }

  return { user, checked, refresh, login, logout }
}
