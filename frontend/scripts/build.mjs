// 文件职责：在前端构建前同步项目元数据并调用 Vite 构建。
// 说明：注释覆盖文件、类型、方法和关键变量；代码执行路径保持不变。

import { spawnSync } from 'node:child_process'

// mode 保存 在前端构建前同步项目元数据并调用 Vite 构建 使用的配置、引用或中间结果。
const mode = process.argv.includes('--dev') ? 'development' : 'production'
// minify 保存 在前端构建前同步项目元数据并调用 Vite 构建 使用的配置、引用或中间结果。
const minify = mode === 'production'

run(process.execPath, ['node_modules/vue-tsc/bin/vue-tsc.js', '--noEmit'])
run(process.execPath, ['node_modules/vite/bin/vite.js', 'build', '--mode', mode, ...(minify ? [] : ['--minify', 'false'])])

// run 处理 在前端构建前同步项目元数据并调用 Vite 构建 中的用户动作、生命周期动作或数据转换。
function run(command, args) {
  // result 保存 在前端构建前同步项目元数据并调用 Vite 构建 使用的配置、引用或中间结果。
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
