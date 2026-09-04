import { createI18n } from 'vue-i18n'
import cookies from 'js-cookie'

function loadLocaleMessages(): {
  [key: string]: { [key: string]: { [key: string]: string } }
} {
  const locales = import.meta.glob('./languages/*.json', {
    eager: true,
    import: 'default'
  }) as Record<string, { [key: string]: { [key: string]: string } }>
  const messages: {
    [key: string]: { [key: string]: { [key: string]: string } }
  } = {}
  Object.entries(locales).forEach(([key, message]) => {
    const matched = key.match(/([^/]+)\.json$/i)
    if (matched && matched.length > 1) {
      const locale = matched[1]
      messages[locale] = message
    }
  })
  if (Object.keys(messages).length > 0) return messages

  // Keep the explicitly named legacy build usable during the migration. The
  // Vue CLI webpack config injects these files through require.context, while
  // Vite uses import.meta.glob above.
  if (typeof require !== 'undefined' && typeof require.context === 'function') {
    const context = require.context('../locales/languages', false, /\.json$/i)
    context.keys().forEach((key: string) => {
      const matched = key.match(/([^/]+)\.json$/i)
      if (matched && matched.length > 1) messages[matched[1]] = context(key)
    })
  }
  return messages
}

export const i18n = createI18n({
  legacy: false,
  locale: cookies.get('locale') ? String(cookies.get('locale')) : 'en',
  fallbackLocale: cookies.get('locale') ? String(cookies.get('locale')) : 'en',
  messages: loadLocaleMessages()
})
