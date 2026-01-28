import { defineConfig, loadEnv } from 'vite'
import react from '@vitejs/plugin-react'
import { fileURLToPath, URL } from 'node:url'
import { resolve, dirname } from 'node:path'

// Vite + React 配置（shadcn/ui 需要检测到 React 框架）
export default defineConfig(({ mode }) => {
  // 获取项目根目录（从 frontend 目录向上查找包含 .env 文件的目录）
  // 使用 import.meta.url 获取当前文件路径，然后向上查找项目根目录
  const currentDir = dirname(fileURLToPath(import.meta.url))
  const rootDir = resolve(currentDir, '..')
  
  // 加载环境变量（从项目根目录的 .env 文件）
  // Vite 会自动查找 .env, .env.local, .env.[mode], .env.[mode].local 等文件
  const env = loadEnv(mode, rootDir, '')
  
  // 从环境变量读取后端 API 地址，默认使用 docker-compose.yml 中的配置
  const apiUrl = env.VITE_API_URL || 'http://127.0.0.1:28080'
  
  return {
    plugins: [react()],
    base: '/', // 根路径直接服务前端
    resolve: {
      alias: {
        '@': fileURLToPath(new URL('./src', import.meta.url)),
      },
    },
    server: {
      port: 5173,
      strictPort: true,
      proxy: {
        '^/api': {
          target: apiUrl,
          changeOrigin: true,
        },
      },
    },
  }
})

