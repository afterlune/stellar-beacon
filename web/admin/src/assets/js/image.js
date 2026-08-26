import placeholder from '@/assets/image-placeholder.svg'

const legacyTalkImagePatterns = [
  /^https?:\/\/i\.demo-user\.com\/talks\//i,
  /^https?:\/\/benetnasch\.oss-cn-shanghai\.aliyuncs\.com\/?talks\//i,
  /^https?:\/\/benetnasch\.oss-cn-shanghai\.aliyuncs\.comtalks\//i
]

const legacyPhotoImagePatterns = [
  /^https?:\/\/example-bucket\.oss-cn-shanghai\.aliyuncs\.com\/photos\//i,
  /^https?:\/\/benetnasch\.oss-cn-shanghai\.aliyuncs\.com\/photos\//i
]

export function isLegacyTalkImage(url) {
  return typeof url === 'string' && legacyTalkImagePatterns.some((pattern) => pattern.test(url.trim()))
}

export function safeTalkImageUrl(url) {
  return typeof url === 'string' && url.trim() !== '' && !isLegacyTalkImage(url) ? url : placeholder
}

export function previewTalkImages(images) {
  if (!Array.isArray(images)) {
    return []
  }
  return images.filter((image) => typeof image === 'string' && image.trim() !== '' && !isLegacyTalkImage(image))
}

export function safePhotoImageUrl(url) {
  return typeof url === 'string' && url.trim() !== '' && !legacyPhotoImagePatterns.some((pattern) => pattern.test(url.trim()))
    ? url
    : placeholder
}

export function safeAvatarImageUrl(url) {
  return typeof url === 'string' && url.trim() !== '' ? url : placeholder
}
