import { createI18n } from 'vue-i18n'
import cookies from 'js-cookie'

function loadLocaleMessages(): {
  [key: string]: { [key: string]: { [key: string]: string } }
} {
  const locales = import.meta.glob<Record<string, Record<string, string>>>('../locales/languages/**/*.json', {
    eager: true,
    import: 'default'
  })
  const messages: {
    [key: string]: { [key: string]: { [key: string]: string } }
  } = {}
  Object.entries(locales).forEach(([path, message]) => {
    const matched = path.match(/([^/]+)\.json$/i)
    if (matched) {
      messages[matched[1]] = message
    }
  })
  return messages
}

export const i18n = createI18n({
  legacy: false,
  locale: cookies.get('locale') ? String(cookies.get('locale')) : 'en',
  fallbackLocale: cookies.get('locale') ? String(cookies.get('locale')) : 'en',
  messages: loadLocaleMessages()
})
