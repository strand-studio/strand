import react from '@vitejs/plugin-react'
import { defineConfig } from 'vite'
import { createHash } from 'node:crypto'
import { copyFileSync, existsSync, mkdirSync, readFileSync } from 'node:fs'
import { fileURLToPath } from 'node:url'
import { dirname, join } from 'node:path'

const webRoot = dirname(fileURLToPath(import.meta.url))
const chartJsPath = join(webRoot, 'public/planning/chart.js')
const chartFallbackPath = join(webRoot, 'public/planning/chart-fallback.js')
const chartRevision = createHash('sha256')
  .update(readFileSync(chartJsPath))
  .update(readFileSync(chartFallbackPath))
  .digest('hex')
  .slice(0, 16)
const hasChartingLibrary = existsSync(
  join(webRoot, 'public/charting_library/charting_library.standalone.js')
)

const lwcSrc = join(
  webRoot,
  'node_modules/lightweight-charts/dist/lightweight-charts.standalone.production.js'
)
const lwcDestDir = join(webRoot, 'public/vendor')
const lwcDest = join(
  lwcDestDir,
  'lightweight-charts.standalone.production.js'
)

function syncLightweightChartsVendor() {
  if (!existsSync(lwcSrc)) return
  mkdirSync(lwcDestDir, { recursive: true })
  copyFileSync(lwcSrc, lwcDest)
}

function vendorLightweightCharts() {
  return {
    name: 'strand-vendor-lightweight-charts',
    buildStart() {
      syncLightweightChartsVendor()
    },
    configureServer() {
      syncLightweightChartsVendor()
    },
  }
}

export default defineConfig({
  plugins: [react(), vendorLightweightCharts()],
  define: {
    __PLANNING_CHART_VERSION__: JSON.stringify(chartRevision),
    __HAS_CHARTING_LIBRARY__: JSON.stringify(hasChartingLibrary),
  },
  server: {
    host: '127.0.0.1',
    port: 3000,
    proxy: {
      '/api': {
        // Prefer loopback literal so local API is never routed via HTTP_PROXY.
        target: 'http://127.0.0.1:8080',
        changeOrigin: true,
        timeout: 90_000,
        proxyTimeout: 90_000,
      },
    },
  },
})
