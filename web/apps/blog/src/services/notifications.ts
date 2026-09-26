import { readonly, ref } from 'vue'

export type NoticeTone = 'success' | 'info' | 'warning' | 'error'
export type NoticeOptions = {
  title?: string
  message?: string
  type?: NoticeTone
  tone?: NoticeTone
  duration?: number
}
export type Notice = Required<Pick<NoticeOptions, 'title' | 'message'>> & {
  id: number
  tone: NoticeTone
}

const notices = ref<Notice[]>([])
const timers = new Map<number, number>()
let nextId = 1

export function dismissNotice(id: number) {
  const timer = timers.get(id)
  if (timer) window.clearTimeout(timer)
  timers.delete(id)
  notices.value = notices.value.filter((notice) => notice.id !== id)
}

function pushNotice(options: NoticeOptions = {}) {
  const id = nextId++
  const tone = options.tone || options.type || 'info'
  notices.value = [...notices.value, {
    id,
    title: options.title || '',
    message: options.message || '',
    tone
  }]
  const duration = options.duration ?? (tone === 'error' ? 6000 : 4000)
  if (duration > 0) timers.set(id, window.setTimeout(() => dismissNotice(id), duration))
  return id
}

type Notify = ((options: NoticeOptions) => number) & {
  success: (message: string, title?: string) => number
  info: (message: string, title?: string) => number
  warning: (message: string, title?: string) => number
  error: (message: string, title?: string) => number
  notices: typeof notices
}

export const notify = Object.assign(
  (options: NoticeOptions) => pushNotice(options),
  {
    success: (message: string, title = '成功') => pushNotice({ message, title, tone: 'success' }),
    info: (message: string, title = '提示') => pushNotice({ message, title, tone: 'info' }),
    warning: (message: string, title = '提示') => pushNotice({ message, title, tone: 'warning' }),
    error: (message: string, title = '错误') => pushNotice({ message, title, tone: 'error' }),
    notices: readonly(notices)
  }
) as Notify

export function useNotices() {
  return { notices: readonly(notices), dismissNotice }
}
