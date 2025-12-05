import { defineConfig } from 'vite'
import vue from '@vitejs/plugin-vue'

export default defineConfig({
  plugins: [vue()],
  server: {
    host: '0.0.0.0', // Важно для Docker: слушать все интерфейсы
    port: 5173,      // Явно указываем порт
    watch: {
      usePolling: true // Нужно для Windows/Docker, чтобы работало автообновление
    },
    proxy: {
      '/api': {
        // 'backend' — это имя сервиса из docker-compose.yml
        target: 'http://backend:8080', 
        changeOrigin: true,
        secure: false,
      }
    }
  }
})
