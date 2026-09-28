import placeholder from '@/assets/avatar-placeholder.svg'

const legacyTalkImagePatterns = [
  /^https?:\/\/[^/]+\.aliyuncs\.com\/?talks\//i
]

const legacyPhotoImagePatterns = [
  /^https?:\/\/[^/]+\.aliyuncs\.com\/photos\//i
]

export function safeTalkImageUrl(url: any): string {
  if (typeof url !== 'string' || url.trim() === '') return placeholder
  if (legacyTalkImagePatterns.some((pattern) => pattern.test(url.trim()))) return placeholder
  return url
}

export function safePhotoImageUrl(url: any): string {
  if (typeof url !== 'string' || url.trim() === '') return placeholder
  if (legacyPhotoImagePatterns.some((pattern) => pattern.test(url.trim()))) return placeholder
  return url
}

export function safeAvatarImageUrl(url: any): string {
  return typeof url === 'string' && url.trim() !== '' ? url : placeholder
}
