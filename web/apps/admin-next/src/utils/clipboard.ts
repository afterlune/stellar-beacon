import { Message } from '@arco-design/web-vue'

import { t } from '@/i18n'

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
    Message.success(options.success ?? t('common.copySuccess'))
    return true
  } catch {
    Message.warning(options.failure ?? t('common.copyFailed'))
    return false
  }
}
