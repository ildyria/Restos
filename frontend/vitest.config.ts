import { fileURLToPath } from 'node:url'
import { defineConfig } from 'vitest/config'

// Plain unit tests: no Nuxt runtime. Nuxt auto-imports used by a file under test are stubbed in the test.
export default defineConfig({
  resolve: {
    alias: {
      '~~': fileURLToPath(new URL('./', import.meta.url)),
      '~': fileURLToPath(new URL('./app', import.meta.url))
    }
  },
  test: { include: ['test/**/*.test.ts'] }
})
