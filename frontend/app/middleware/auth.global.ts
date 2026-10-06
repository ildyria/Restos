import { requiresLogin } from '~/utils/auth'

// Runs on every navigation, so a new admin page is protected without opting in.
// This only keeps the UI tidy; the API refuses admin requests without a session regardless.
export default defineNuxtRouteMiddleware(async (to) => {
  if (!requiresLogin(to.path)) return
  const { user, checked, refresh } = useAuth()
  if (!checked.value) await refresh()
  if (!user.value) {
    return navigateTo({ path: '/admin/login', query: { next: to.fullPath } })
  }
})
