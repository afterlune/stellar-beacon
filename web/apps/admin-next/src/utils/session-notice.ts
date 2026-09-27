/**
 * 会话过期提示的一次性开关。
 *
 * 一个页面往往同时有多个请求在飞，token 失效时它们会一起返回 401，直接弹提示会
 * 叠出一排重复消息。这里保证同一轮失效只提示一次，登录成功后重新放开。
 */
let notified = false

export function shouldNotifySessionExpired(): boolean {
  if (notified) return false
  notified = true
  return true
}

export function resetSessionExpiredNotice(): void {
  notified = false
}
