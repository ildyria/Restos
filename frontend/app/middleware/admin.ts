export default defineNuxtRouteMiddleware(async (to) => {
  const { user, checked, refresh } = useAuth()
  if (!checked.value) await refresh()
  if (!user.value) {
    return navigateTo({ path: '/admin/login', query: { next: to.fullPath } })
  }
})
