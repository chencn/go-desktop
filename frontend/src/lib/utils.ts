import { clsx, type ClassValue } from 'clsx'
import { twMerge } from 'tailwind-merge'

// cn 是项目 className 合并的唯一入口，先展开条件 class，再用 tailwind-merge 消除 Tailwind 冲突。
export function cn(...inputs: ClassValue[]) {
  return twMerge(clsx(inputs))
}
