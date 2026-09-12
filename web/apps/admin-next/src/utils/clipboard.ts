import { Message } from '@arco-design/web-vue'

/**
 * 复制文本到剪贴板。
 *
 * 后台可能部署在 http:// 局域网地址上，那种上下文里浏览器会直接拒绝
 * `navigator.clipboard`，所以失败必须降级成「请手动复制」的提示，而不是抛异常。
 */
export async function copyText(
  text: string,
  options: { success?: string; failure?: string } = {}
): Promise<boolean> {
  try {
    await navigator.clipboard.writeText(text)
    Message.success(options.success ?? '已复制到剪贴板')
    return true
  } catch {
    Message.warning(options.failure ?? '浏览器不允许自动复制，请手动复制')
    return false
  }
}
