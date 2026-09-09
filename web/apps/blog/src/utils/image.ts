import placeholder from '@/assets/avatar-placeholder.svg'

const legacyTalkImagePatterns = [
  /^https?:\/\/i\.demo-user\.com\/talks\//i,
  /^https?:\/\/benetnasch\.oss-cn-shanghai\.aliyuncs\.com\/?talks\//i,
  /^https?:\/\/benetnasch\.oss-cn-shanghai\.aliyuncs\.comtalks\//i
]

const legacyPhotoImagePatterns = [
  /^https?:\/\/example-bucket\.oss-cn-shanghai\.aliyuncs\.com\/photos\//i,
  /^https?:\/\/benetnasch\.oss-cn-shanghai\.aliyuncs\.com\/photos\//i
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
