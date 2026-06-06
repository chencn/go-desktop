// 文件职责：提供前端 className 合并工具函数。
// 说明：注释覆盖文件、类型、方法和关键变量；代码执行路径保持不变。

import { clsx, type ClassValue } from 'clsx'
import { twMerge } from 'tailwind-merge'

// cn 处理 提供前端 className 合并工具函数 中的用户动作、生命周期动作或数据转换。
export function cn(...inputs: ClassValue[]) {
  return twMerge(clsx(inputs))
}
