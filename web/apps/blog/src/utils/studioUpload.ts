import api from '@/api/api'

export const MAX_STUDIO_UPLOAD_BYTES = 10 * 1024 * 1024
export const STUDIO_IMAGE_TYPES = ['image/jpeg', 'image/png', 'image/gif', 'image/webp', 'image/svg+xml']

export function validateStudioImage(file: File): string {
  if (!STUDIO_IMAGE_TYPES.includes(file.type)) return '仅支持 JPG、PNG、GIF、WebP 或 SVG 图片'
  if (file.size > MAX_STUDIO_UPLOAD_BYTES) return '图片不能超过 10MB'
  return ''
}

export async function uploadStudioImage(file: File, kind: 'article-cover' | 'article-inline' | 'talk-image' | 'series-cover' | 'avatar'): Promise<string> {
  const validation = validateStudioImage(file)
  if (validation) throw new Error(validation)
  const response = await api.uploadStudioAsset(file, kind)
  if (!response?.data?.flag || typeof response.data.data !== 'string') {
    throw new Error(response?.data?.message || '图片上传失败')
  }
  return response.data.data
}

export function parseTalkImages(value: unknown): string[] {
  if (Array.isArray(value)) return value.map(String).filter(Boolean)
  if (typeof value !== 'string' || !value.trim()) return []
  try {
    const parsed = JSON.parse(value)
    if (Array.isArray(parsed)) return parsed.map(String).filter(Boolean)
  } catch {
    // Legacy studio data allowed comma-separated image URLs.
  }
  return value.split(',').map((item) => item.trim()).filter(Boolean)
}