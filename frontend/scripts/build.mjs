// 文件职责：先执行 TypeScript 类型检查，再按指定模式调用 Vite 构建。

import { spawnSync } from 'node:child_process'

// --dev 走 development mode，生产构建默认开启 Vite 压缩。
const mode = process.argv.includes('--dev') ? 'development' : 'production'
const minify = mode === 'production'

run(process.execPath, ['node_modules/vue-tsc/bin/vue-tsc.js', '--noEmit'])
run(process.execPath, ['node_modules/vite/bin/vite.js', 'build', '--mode', mode, ...(minify ? [] : ['--minify', 'false'])])

// 子命令继承 stdio，保持 vue-tsc / vite 的原始错误输出和退出码。
function run(command, args) {
  const result = spawnSync(command, args, {
    stdio: 'inherit',
    shell: false,
  })
  if (result.error) {
    console.error(result.error.message)
    process.exit(1)
  }
  if (result.status !== 0) {
    process.exit(result.status ?? 1)
  }
}
