/** Admin pages anyone may open: signing in and signing up. */
export const PUBLIC_ADMIN_PATHS = ['/admin/login', '/admin/register']

/** Whether a route needs a signed-in user: everything under /admin except sign-in and sign-up. */
export function requiresLogin(path: string) {
  const p = path.replace(/\/+$/, '') || '/'
  return (p === '/admin' || p.startsWith('/admin/')) && !PUBLIC_ADMIN_PATHS.includes(p)
}
