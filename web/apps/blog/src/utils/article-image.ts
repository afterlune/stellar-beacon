const FALLBACK_IMAGE =
  'data:image/svg+xml;utf8,' +
  encodeURIComponent(
    "<svg xmlns='http://www.w3.org/2000/svg' width='640' height='360'>" +
      "<rect width='640' height='360' fill='#0e1320'/>" +
      "<rect x='24' y='24' width='592' height='312' fill='none' stroke='#1f2a3d' stroke-width='2'/>" +
      "<text x='50%' y='52%' fill='#657084' font-family='monospace' font-size='18' text-anchor='middle'>IMAGE UNAVAILABLE</text>" +
      '</svg>'
  )

/** Swaps a broken article image for an inline placeholder without looping on error. */
export function applyArticleImageFallback(image: HTMLImageElement): void {
  if (!image || image.dataset.fallbackApplied === 'true') return
  image.dataset.fallbackApplied = 'true'
  image.classList.add('article-image-fallback')
  image.src = FALLBACK_IMAGE
}
