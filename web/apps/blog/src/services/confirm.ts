import { reactive } from 'vue'

export type ConfirmOptions = {
  title: string
  message: string
  confirmText?: string
  cancelText?: string
  tone?: 'warning' | 'info'
}

export type PendingConfirm = ConfirmOptions & { id: number; resolve: (confirmed: boolean) => void }

export const pendingConfirms = reactive<PendingConfirm[]>([])
let nextId = 1

export function confirm(options: ConfirmOptions): Promise<boolean> {
  return new Promise((resolve) => {
    pendingConfirms.push({
      id: nextId++,
      tone: 'warning',
      confirmText: '确认',
      cancelText: '取消',
      ...options,
      resolve
    })
  })
}

export function finishConfirm(confirmed: boolean) {
  const pending = pendingConfirms.shift()
  pending?.resolve(confirmed)
}
