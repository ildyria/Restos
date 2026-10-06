export default defineNuxtConfig({
  compatibilityDate: '2026-01-01',
  ssr: false,
  devtools: { enabled: false },

  modules: ['@nuxt/ui', '@nuxt/fonts'],
  css: ['~/assets/css/main.css'],

  ui: {
    theme: { colors: ['primary', 'secondary', 'info', 'success', 'warning', 'error', 'neutral'] }
  },

  fonts: {
    families: [
      { name: 'EB Garamond', provider: 'google', weights: [400], styles: ['normal', 'italic'] },
      { name: 'Fraunces', provider: 'google', weights: [400, 500, 600], styles: ['normal', 'italic'] },
      { name: 'Inter', provider: 'google', weights: [400, 500, 600] }
    ]
  },

  colorMode: { preference: 'system', fallback: 'light' },

  app: {
    head: {
      htmlAttrs: { lang: 'en' },
      title: 'À\u00a0table\u202f!',
      meta: [{ name: 'viewport', content: 'width=device-width, initial-scale=1' }],
      link: [{ rel: 'icon', href: '/favicon.svg', type: 'image/svg+xml' }]
    }
  },

  // In dev the browser only talks to Nuxt; /api and /media go to the Go server.
  // changeOrigin stays off so the Host header matches the browser's Origin (the API checks both).
  nitro: {
    devProxy: {
      '/api': { target: 'http://localhost:8080/api', changeOrigin: false },
      '/media': { target: 'http://localhost:8080/media', changeOrigin: false }
    }
  }
})
