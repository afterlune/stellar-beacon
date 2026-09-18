export function scrollToArticleHeading(id: string): boolean {
  if (!id || typeof document === 'undefined') return false
  const target = document.getElementById(id)
  if (!target) return false
  const reducedMotion = typeof window.matchMedia === 'function' && window.matchMedia('(prefers-reduced-motion: reduce)').matches
  window.history.replaceState(null, '', `${window.location.pathname}${window.location.search}#${encodeURIComponent(id)}`)
  const top = Math.max(0, target.getBoundingClientRect().top + window.scrollY - 88)
  window.scrollTo({ top, behavior: reducedMotion ? 'auto' : 'smooth' })
  return true
}