import { defineConfig } from 'vite'
import react from '@vitejs/plugin-react'

export default defineConfig({
  plugins: [react()],
  server: {
    port: 5173,
    host: true,
    proxy: {
      '/api/v1': {
        target: 'http://127.0.0.1:8080',
        changeOrigin: true,
      },
    },
    // also forward the bare-path OAuth callback the browser lands on
    // (Google redirects to :8080 directly so this is only needed for the authorize link)
  },
})
