import { ref, watch } from 'vue'
import { defineStore } from 'pinia'

export type Theme = 'dark' | 'light'

const STORAGE_KEY = 'stellar-beacon.admin.theme'
const LEGACY_STORAGE_KEY = 'benetnasch.admin.theme'

export const useThemeStore = defineStore('admin-theme', () => {
  const theme = ref<Theme>(readTheme())

  watch(theme, (value) => {
    document.documentElement.setAttribute('data-theme', value)
    document.documentElement.style.colorScheme = value
    localStorage.setItem(STORAGE_KEY, value)
  }, { immediate: true })

  function toggle(): void {
    theme.value = theme.value === 'dark' ? 'light' : 'dark'
  }

  return { theme, toggle }
})

function readTheme(): Theme {
  try {
    let value = localStorage.getItem(STORAGE_KEY)
    if (value === null) {
      value = localStorage.getItem(LEGACY_STORAGE_KEY)
      if (value !== null) localStorage.setItem(STORAGE_KEY, value)
    }
    return value === 'light' ? 'light' : 'dark'
  } catch {
    return 'dark'
  }
}
