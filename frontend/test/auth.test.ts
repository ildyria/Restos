import { readdirSync, readFileSync, statSync } from 'node:fs'
import { join, relative } from 'node:path'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import { PUBLIC_ADMIN_PATHS, requiresLogin } from '~/utils/auth'

describe('requiresLogin', () => {
  it.each([
    '/admin', '/admin/', '/admin/ranking', '/admin/users', '/admin/about',
    '/admin/restaurants/new', '/admin/restaurants/12', '/admin/login/extra'
  ])('protects %s', path => expect(requiresLogin(path)).toBe(true))

  it.each([
    '/', '/about', '/r/brasserie-marcel', '/admin/login', '/admin/login/', '/admin/register',
    '/administrator', '/r/admin'
  ])('leaves %s public', path => expect(requiresLogin(path)).toBe(false))
})

// Every page file under pages/admin maps to a protected route, except sign-in and sign-up.
describe('admin pages', () => {
  const root = join(__dirname, '../app/pages')
  const files = (dir: string): string[] => readdirSync(dir).flatMap((f) => {
    const p = join(dir, f)
    return statSync(p).isDirectory() ? files(p) : p.endsWith('.vue') ? [p] : []
  })
  const routeOf = (file: string) => '/' + relative(root, file)
    .replace(/\.vue$/, '')
    .replace(/(^|\/)index$/, '')
    .replace(/\[[^\]]+\]/g, '1')
  const pages = files(join(root, 'admin')).map(f => ({ file: relative(root, f), route: routeOf(f) }))

  it('finds the admin pages', () => {
    expect(pages.map(p => p.route)).toEqual(expect.arrayContaining(['/admin', '/admin/login', '/admin/restaurants/1']))
  })

  it.each(pages)('$route ($file) needs a login unless it is sign-in/up', ({ route }) => {
    expect(requiresLogin(route)).toBe(!PUBLIC_ADMIN_PATHS.includes(route))
  })

  it.each(pages)('$file does not rely on a removed per-page middleware', ({ file }) => {
    expect(readFileSync(join(root, file), 'utf8')).not.toMatch(/middleware:\s*['"]admin['"]/)
  })
})

// The global middleware itself, with Nuxt's auto-imports stubbed.
describe('auth.global middleware', () => {
  const navigateTo = vi.fn((to: unknown) => ({ redirectedTo: to }))
  const auth = { user: { value: null as null | { id: number } }, checked: { value: false }, refresh: vi.fn() }
  let middleware: (to: { path: string, fullPath: string }) => Promise<unknown>

  beforeEach(async () => {
    vi.resetModules()
    navigateTo.mockClear()
    auth.refresh.mockReset()
    auth.user.value = null
    auth.checked.value = false
    vi.stubGlobal('defineNuxtRouteMiddleware', (fn: unknown) => fn)
    vi.stubGlobal('useAuth', () => auth)
    vi.stubGlobal('navigateTo', navigateTo)
    middleware = (await import('~/middleware/auth.global')).default as typeof middleware
  })

  const route = (fullPath: string) => ({ path: fullPath.split('?')[0]!, fullPath })

  it('sends a signed-out visitor to the login page, remembering where they were going', async () => {
    auth.refresh.mockImplementation(async () => { auth.checked.value = true })
    const res = await middleware(route('/admin/restaurants/3?tab=visits'))
    expect(auth.refresh).toHaveBeenCalledOnce()
    expect(navigateTo).toHaveBeenCalledWith({ path: '/admin/login', query: { next: '/admin/restaurants/3?tab=visits' } })
    expect(res).toEqual({ redirectedTo: expect.anything() })
  })

  it('lets a signed-in admin through', async () => {
    auth.refresh.mockImplementation(async () => { auth.checked.value = true; auth.user.value = { id: 1 } })
    expect(await middleware(route('/admin'))).toBeUndefined()
    expect(navigateTo).not.toHaveBeenCalled()
  })

  it('does not re-check the session once known', async () => {
    auth.checked.value = true
    auth.user.value = { id: 1 }
    await middleware(route('/admin/users'))
    expect(auth.refresh).not.toHaveBeenCalled()
  })

  it.each(['/', '/r/x', '/admin/login', '/admin/register'])('leaves %s alone without asking the server', async (path) => {
    expect(await middleware(route(path))).toBeUndefined()
    expect(auth.refresh).not.toHaveBeenCalled()
    expect(navigateTo).not.toHaveBeenCalled()
  })
})
