/**
 * 轻量 i18n 内核。
 *
 * 后台只有中英两种语言、没有复数/日期格式化的复杂需求，因此不引入 vue-i18n，
 * 只用「扁平词典 + 点号路径 + {占位符}」这一层薄封装：
 *   - 词典按业务域拆到 ./messages/*.ts，键名是 `域.小驼峰`
 *   - 模板里直接用全局 `t('域.键')`，脚本里按需 import { t }
 *   - 缺少 en 词条时回退到 zh，再回退到键名本身（构建期不会静默丢字）
 */
import { computed, ref, watch, type ComputedRef, type Ref } from 'vue'

import { messages } from './messages'

export type Locale = 'zh-CN' | 'en-US'

const STORAGE_KEY = 'stellar-beacon.admin.locale'
const LEGACY_STORAGE_KEY = 'benetnasch.admin.locale'

export const locale: Ref<Locale> = ref(readLocale())

// 语言变化的唯一副作用点：写 localStorage + 同步 <html lang>。
// 放在这里（而不是 setLocale/toggleLocale 里）保证任何修改入口都会持久化。
watch(locale, (value) => {
  document.documentElement.setAttribute('lang', value)
  try {
    localStorage.setItem(STORAGE_KEY, value)
  } catch {
    /* 隐私模式：只在内存里保留选择 */
  }
}, { immediate: true })

/** 当前语言的词典（en 缺失的键在 resolve 时回退）。 */
export const t = (key: string, params?: Record<string, string | number>): string => {
  const dict = messages[locale.value] as Record<string, string | undefined>
  const raw = dict[key] ?? (messages['zh-CN'] as Record<string, string | undefined>)[key] ?? key
  return params ? interpolate(raw, params) : raw
}

/** 判断某键是否已有当前语言的词条，用于「有则翻译、无则原样」的场景（例如后端返回的菜单名）。 */
export const te = (key: string): boolean => {
  const dict = messages[locale.value] as Record<string, string | undefined>
  return typeof dict[key] === 'string'
}

export const localeOptions: ReadonlyArray<{ value: Locale; label: string; short: string }> = [
  { value: 'zh-CN', label: '简体中文', short: '中' },
  { value: 'en-US', label: 'English', short: 'EN' }
]

export const isEnglish: ComputedRef<boolean> = computed(() => locale.value === 'en-US')

export function setLocale(next: Locale): void {
  locale.value = next
}

export function toggleLocale(): void {
  locale.value = locale.value === 'zh-CN' ? 'en-US' : 'zh-CN'
}

function interpolate(template: string, params: Record<string, string | number>): string {
  return template.replace(/\{(\w+)\}/g, (match, name: string) => (
    Object.prototype.hasOwnProperty.call(params, name) ? String(params[name]) : match
  ))
}

function readLocale(): Locale {
  try {
    let value = localStorage.getItem(STORAGE_KEY)
    if (value === null) {
      value = localStorage.getItem(LEGACY_STORAGE_KEY)
      if (value !== null) localStorage.setItem(STORAGE_KEY, value)
    }
    if (value === 'zh-CN' || value === 'en-US') return value
  } catch {
    /* 读不到就按浏览器语言判断 */
  }
  return detectBrowserLocale()
}

function detectBrowserLocale(): Locale {
  const languages = Array.from(navigator.languages ?? [])
  if (languages.length === 0 && navigator.language) languages.push(navigator.language)
  // 只要浏览器首选语言里出现过中文，就按中文处理：后台的主要用户是中文使用者。
  return languages.some((item) => item.toLowerCase().startsWith('zh')) ? 'zh-CN' : 'en-US'
}
