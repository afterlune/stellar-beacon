<template>
  <div class="studio-profile-page">
    <header class="studio-profile-head">
      <button type="button" class="studio-profile-back" @click="backToDashboard">← 返回总览</button>
      <div>
        <p>WORKSPACE / 08</p>
        <h1>公开资料</h1>
        <span>决定你在公共空间和作者主页上的身份呈现。</span>
      </div>
      <span class="studio-profile-state">{{ saveState }}</span>
    </header>

    <p v-if="loading" class="studio-profile-state-message">正在载入公开资料…</p>
    <p v-else-if="error" class="studio-profile-state-message is-error">{{ error }}</p>

    <form v-else class="studio-profile-form" @submit.prevent="save">
      <div class="studio-profile-layout">
        <section class="studio-profile-main">
          <section class="studio-profile-panel studio-profile-panel--identity">
            <header>
              <div><p>IDENTITY</p><h2>作者身份</h2></div>
              <span>{{ completion.completed }}/{{ completion.total }} 已完成</span>
            </header>
            <div class="studio-profile-identity">
              <div class="studio-profile-avatar">
                <button id="studio-profile-avatar" type="button" :disabled="uploadingAvatar" @click="showCropper = true">
                  <img :src="avatarPreview" :alt="form.nickname || '作者头像'" />
                  <span>{{ uploadingAvatar ? '上传中…' : '更换头像' }}</span>
                </button>
                <AvatarCropper
                  v-model="showCropper"
                  trigger="#studio-profile-avatar"
                  upload-url="/api/v1/auth/me/avatar"
                  mimes="image/png, image/jpeg, image/gif, image/webp"
                  :labels="{ submit: '上传头像', cancel: '取消' }"
                  :request-options="cropperOptions"
                  @uploaded="handleAvatarUploaded" />
                <small>建议使用清晰的人像或品牌头像，支持 JPG、PNG、GIF、WebP。</small>
              </div>

              <div class="studio-profile-fields">
                <label class="studio-profile-field">
                  <span>Handle</span>
                  <div class="studio-profile-handle">
                    <span>/u/</span>
                    <input
                      :value="form.handle"
                      required
                      autocomplete="off"
                      spellcheck="false"
                      placeholder="your-handle"
                      @input="updateHandle" />
                  </div>
                  <small>{{ form.handle.length }}/40 · 小写字母、数字和连字符</small>
                </label>
                <label class="studio-profile-field">
                  <span>昵称</span>
                  <input v-model.trim="form.nickname" required maxlength="30" autocomplete="nickname" placeholder="作者昵称" />
                  <small>{{ form.nickname.length }}/30</small>
                </label>
              </div>
            </div>
          </section>

          <section class="studio-profile-panel">
            <header><div><p>ABOUT</p><h2>主页介绍</h2></div></header>
            <label class="studio-profile-field">
              <span>个人简介</span>
              <textarea v-model.trim="form.intro" maxlength="255" rows="5" placeholder="介绍你的关注领域、写作方向或正在做的事。" />
              <small>{{ form.intro.length }}/255</small>
            </label>
            <label class="studio-profile-field">
              <span>个人网站</span>
              <input v-model.trim="form.website" maxlength="255" inputmode="url" placeholder="https://example.com" />
              <small>可选，必须以 http:// 或 https:// 开头</small>
            </label>
            <label class="studio-profile-field studio-profile-long-about">
              <span>详细介绍</span>
              <textarea v-model="form.about" maxlength="20000" rows="10" placeholder="介绍你的创作方向、兴趣或个人经历。支持 Markdown。" />
              <small>{{ form.about.length }}/20000</small>
            </label>
          </section>

          <section class="studio-profile-panel">
            <header>
              <div><p>LINKS</p><h2>个人外链</h2></div>
              <button type="button" class="studio-profile-add-link" @click="form.links.push({ label: '', url: '', description: '' })">添加链接</button>
            </header>
            <div v-if="form.links.length" class="studio-profile-links">
              <div v-for="(link, index) in form.links" :key="index" class="studio-profile-link-row">
                <label class="studio-profile-field">
                  <span>名称</span>
                  <input v-model.trim="link.label" maxlength="40" placeholder="GitHub、个人主页…" />
                </label>
                <label class="studio-profile-field">
                  <span>链接</span>
                  <input v-model.trim="link.url" maxlength="1000" inputmode="url" placeholder="https://example.com" />
                </label>
                <label class="studio-profile-field">
                  <span>说明（可选）</span>
                  <input v-model.trim="link.description" maxlength="160" placeholder="简短说明" />
                </label>
                <button type="button" class="studio-profile-remove-link" @click="form.links.splice(index, 1)">移除</button>
              </div>
            </div>
            <p v-else class="studio-profile-empty-links">添加社交账号、个人网站或其他公开链接。</p>
          </section>
        </section>

        <aside class="studio-profile-sidebar">
          <section class="studio-profile-panel studio-profile-preview">
            <header><div><p>PUBLIC PREVIEW</p><h2>主页预览</h2></div></header>
            <div class="studio-public-card">
              <img :src="avatarPreview" :alt="form.nickname || '作者头像'" />
              <span>@{{ form.handle || 'your-handle' }}</span>
              <h2>{{ form.nickname || '未设置昵称' }}</h2>
              <p>{{ form.intro || '这位作者还没有写下简介。' }}</p>
              <a v-if="validPreviewWebsite" :href="form.website" target="_blank" rel="noopener noreferrer">{{ form.website }}</a>
              <a v-for="(link, index) in form.links.filter((item) => item.label && item.url)" :key="index" :href="link.url" target="_blank" rel="noopener noreferrer">{{ link.label }}</a>
              <router-link v-if="validHandle" :to="`/u/${form.handle}`">打开公开主页 →</router-link>
            </div>
          </section>

          <section class="studio-profile-panel studio-profile-completion">
            <header><div><p>READINESS</p><h2>身份完成度</h2></div></header>
            <div class="studio-profile-progress">
              <strong>{{ completion.completed }}/{{ completion.total }}</strong>
              <span><i :style="{ width: `${completion.completed * 25}%` }" /></span>
            </div>
            <ul>
              <li v-for="item in completion.checks" :key="item.key" :class="{ done: item.done }">
                <span>{{ item.done ? '✓' : '○' }}</span>{{ item.label }}
              </li>
            </ul>
            <p v-if="completion.missing.length">还缺：{{ completion.missing.map((item) => item.label).join('、') }}</p>
            <p v-else>公开身份已经完整，可以继续保持。</p>
          </section>
        </aside>
      </div>

      <footer class="studio-profile-footer">
        <span>{{ form.handle ? `公开地址：/u/${form.handle}` : '设置 Handle 后会生成公开地址' }}</span>
        <div>
          <router-link v-if="validHandle" :to="`/u/${form.handle}`" target="_blank" class="studio-profile-link">查看公开主页</router-link>
          <button type="button" class="studio-profile-secondary" @click="backToDashboard">返回总览</button>
          <button type="submit" class="studio-profile-primary" :disabled="saving">{{ saving ? '保存中…' : '保存公开资料' }}</button>
        </div>
      </footer>
    </form>

    <div v-if="leaveVisible" class="studio-profile-leave">
      <section>
        <h2>还有未保存的修改</h2>
        <p>关闭前是否放弃这些公开资料改动？</p>
        <div>
          <button type="button" @click="cancelLeave">继续编辑</button>
          <button type="button" class="danger" @click="confirmLeave">放弃修改</button>
        </div>
      </section>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed, onMounted, reactive, ref } from 'vue'
import { useRouter } from 'vue-router'
import { notify } from '@/services/notifications'
import { confirm } from '@/services/confirm'
import AvatarCropper from 'vue-avatar-cropper'

import api from '@/api/api'
import { useUnsavedGuard } from '@/composables/useUnsavedGuard'
import { stableSnapshot } from '@/composables/useStudioDraft'
import { useAppStore } from '@/stores/app'
import { useUserStore } from '@/stores/user'
import {
  isValidStudioHandle,
  isValidStudioWebsite,
  normalizeStudioHandle,
  studioProfileCompletion,
  type StudioProfile
} from '@/utils/studioProfile'

const router = useRouter()
const appStore = useAppStore()
const userStore = useUserStore()
const defaultAvatar = 'data:image/svg+xml,%3Csvg xmlns="http://www.w3.org/2000/svg" width="160" height="160"%3E%3Crect width="160" height="160" rx="80" fill="%23172554"/%3E%3Ccircle cx="80" cy="61" r="28" fill="%239bb8ff"/%3E%3Cpath d="M27 148c8-38 27-56 53-56s45 18 53 56" fill="%239bb8ff"/%3E%3C/svg%3E'

const loading = ref(true)
const saving = ref(false)
const uploadingAvatar = ref(false)
const showCropper = ref(false)
const error = ref('')
const saveState = ref('')
const form = reactive<StudioProfile>({ handle: '', nickname: '', avatar: '', intro: '', website: '', about: '', links: [] })
let savedHandle = ''

const stableForm = () => stableSnapshot(form)
const { visible: leaveVisible, markClean, confirmLeave, cancelLeave } = useUnsavedGuard(stableForm)

const avatarPreview = computed(() => form.avatar || appStore.websiteConfig?.userAvatar || defaultAvatar)
const validHandle = computed(() => isValidStudioHandle(form.handle))
const validPreviewWebsite = computed(() => Boolean(form.website) && isValidStudioWebsite(form.website))
const completion = computed(() => studioProfileCompletion(form, appStore.websiteConfig?.userAvatar || ''))
const cropperOptions = computed(() => ({
  method: 'POST',
  headers: { Authorization: 'Bearer ' + userStore.token }
}))

function updateHandle(event: Event): void {
  form.handle = normalizeStudioHandle((event.target as HTMLInputElement).value).slice(0, 40)
}

function validate(): string {
  form.handle = normalizeStudioHandle(form.handle)
  form.nickname = String(form.nickname || '').trim()
  form.intro = String(form.intro || '').trim()
  form.website = String(form.website || '').trim()
  form.about = String(form.about || '').trim()
  if (!isValidStudioHandle(form.handle)) return 'Handle 需为 3-40 位小写字母、数字或连字符，且必须以字母或数字开头'
  if (!form.nickname) return '昵称不能为空'
  if ([...form.nickname].length > 30) return '昵称不能超过 30 个字'
  if ([...form.intro].length > 255) return '个人简介不能超过 255 个字'
  if ([...form.website].length > 255) return '个人网站不能超过 255 个字'
  if (!isValidStudioWebsite(form.website)) return '个人网站必须是有效的 HTTP(S) 地址'
  if ([...form.about].length > 20000) return '主页介绍不能超过 20000 个字'
  if (form.links.length > 100) return '个人外链不能超过 100 条'
  for (const link of form.links) {
    link.label = String(link.label || '').trim()
    link.url = String(link.url || '').trim()
    link.description = String(link.description || '').trim()
    if (!link.label && !link.url && !link.description) continue
    if (!link.label || [...link.label].length > 40) return '每条外链都需要填写不超过 40 个字的名称'
    if (!isValidStudioWebsite(link.url)) return '个人外链必须是有效的 HTTP(S) 地址'
    if ([...link.description].length > 160) return '外链说明不能超过 160 个字'
  }
  return ''
}

async function load(): Promise<void> {
  loading.value = true
  error.value = ''
  try {
    const response = await api.getStudioProfile()
    if (!response?.data?.flag) throw new Error(response?.data?.message || '公开资料加载失败')
    const profile = response.data.data || {}
    Object.assign(form, {
      handle: normalizeStudioHandle(profile.handle),
      nickname: String(profile.nickname || ''),
      avatar: String(profile.avatar || ''),
      intro: String(profile.intro || ''),
      website: String(profile.website || ''),
      about: String(profile.about || ''),
      links: Array.isArray(profile.links) ? profile.links.map((link: any) => ({ label: String(link.label || ''), url: String(link.url || ''), description: String(link.description || '') })) : []
    })
    savedHandle = form.handle
    markClean()
  } catch (reason: any) {
    error.value = reason?.response?.data?.message || reason?.message || '公开资料加载失败'
  } finally {
    loading.value = false
  }
}

async function save(): Promise<void> {
  const validation = validate()
  if (validation) {
    notify.warning(validation)
    return
  }
  if (savedHandle && form.handle !== savedHandle) {
    const accepted = await confirm({
      message: `Handle 将从 @${savedHandle} 改为 @${form.handle}。旧主页 /u/${savedHandle} 将不再可访问。`,
      title: '确认更换公开地址',
      confirmText: '确认更换'
    })
    if (!accepted) return
  }

  saving.value = true
  saveState.value = '正在保存…'
  try {
    const response = await api.saveStudioProfile({
      handle: form.handle,
      nickname: form.nickname,
      intro: form.intro,
      website: form.website,
      about: form.about,
      links: form.links
    })
    if (!response?.data?.flag) throw new Error(response?.data?.message || '保存失败')
    const updated = response.data.data || { ...form }
    Object.assign(form, {
      handle: normalizeStudioHandle(updated.handle),
      nickname: String(updated.nickname || ''),
      avatar: String(updated.avatar || form.avatar || ''),
      intro: String(updated.intro || ''),
      website: String(updated.website || ''),
      about: String(updated.about || ''),
      links: Array.isArray(updated.links) ? updated.links : []
    })
    savedHandle = form.handle
    markClean()
    userStore.userInfo = { ...(userStore.userInfo || {}), ...form }
    saveState.value = `已保存 · ${new Date().toLocaleTimeString('zh-CN', { hour: '2-digit', minute: '2-digit' })}`
    notify.success('公开资料已更新')
  } catch (reason: any) {
    saveState.value = '保存失败'
    notify.error(reason?.response?.data?.message || reason?.message || '保存失败')
  } finally {
    saving.value = false
  }
}

async function handleAvatarUploaded(payload: any): Promise<void> {
  uploadingAvatar.value = true
  try {
    const response = payload?.response
    const data = response && typeof response.json === 'function' ? await response.json() : payload
    const success = data?.flag === true || data?.code === 'OK' || data?.code === 'SUCCESS'
    if (!success || typeof data.data !== 'string') throw new Error(data?.message || '头像上传失败')
    form.avatar = data.data
    userStore.userInfo = { ...(userStore.userInfo || {}), avatar: data.data }
    notify.success('头像已更新')
  } catch (reason: any) {
    notify.error(reason?.message || '头像上传失败')
  } finally {
    uploadingAvatar.value = false
    showCropper.value = false
  }
}

function backToDashboard(): void {
  void router.push('/studio/dashboard')
}

onMounted(() => {
  void load()
})
</script>

<style lang="scss" scoped>
.studio-profile-page { min-width: 0; padding-bottom: 76px; }
.studio-profile-head { display: grid; grid-template-columns: auto 1fr auto; gap: 18px; align-items: center; padding: 22px 24px; border: 1px solid var(--border-hairline); border-radius: 18px; background: radial-gradient(circle at 88% 0, rgba(97, 73, 184, .18), transparent 38%), color-mix(in srgb, var(--background-primary-alt) 94%, transparent); }
.studio-profile-head p, .studio-profile-panel header p { margin: 0 0 5px; color: var(--color-ob); font-size: 10px; letter-spacing: .18em; }
.studio-profile-head h1 { margin: 0 0 5px; font-size: clamp(1.7rem, 3.6vw, 2.7rem); }
.studio-profile-head span { color: var(--text-ob-dim); font-size: 12px; }
.studio-profile-state { justify-self: end; }
.studio-profile-back, .studio-profile-secondary, .studio-profile-primary { padding: 9px 14px; border: 1px solid var(--border-hairline); border-radius: 999px; background: transparent; color: inherit; cursor: pointer; }
.studio-profile-state-message { padding: 60px 0; color: var(--text-ob-dim); text-align: center; }
.studio-profile-state-message.is-error { color: #df8177; }
.studio-profile-form { margin-top: 18px; }
.studio-profile-layout { display: grid; grid-template-columns: minmax(0, 1fr) 330px; gap: 18px; align-items: start; }
.studio-profile-main, .studio-profile-sidebar { display: grid; gap: 14px; }
.studio-profile-sidebar { position: sticky; top: 18px; }
.studio-profile-panel { padding: 22px; border: 1px solid var(--border-hairline); border-radius: 18px; background: color-mix(in srgb, var(--background-primary-alt) 92%, transparent); }
.studio-profile-panel > header { display: flex; align-items: flex-start; justify-content: space-between; gap: 14px; margin-bottom: 18px; }
.studio-profile-panel h2 { margin: 0; font-size: 1.05rem; }
.studio-profile-panel > header > span { color: var(--text-ob-dim); font-size: 11px; }
.studio-profile-identity { display: grid; grid-template-columns: 170px minmax(0, 1fr); gap: 22px; align-items: start; }
.studio-profile-avatar { display: grid; gap: 10px; justify-items: center; text-align: center; }
.studio-profile-avatar button { position: relative; width: 150px; height: 150px; overflow: hidden; padding: 0; border: 1px solid color-mix(in srgb, var(--color-ob) 48%, transparent); border-radius: 50%; background: transparent; cursor: pointer; }
.studio-profile-avatar img { width: 100%; height: 100%; object-fit: cover; }
.studio-profile-avatar button span { position: absolute; right: 0; bottom: 0; left: 0; padding: 7px; background: rgba(4, 8, 25, .78); color: #fff; font-size: 11px; }
.studio-profile-avatar small, .studio-profile-field small { color: var(--text-ob-dim); font-size: 10px; line-height: 1.6; }
.studio-profile-fields { display: grid; gap: 14px; }
.studio-profile-field { display: grid; gap: 7px; color: var(--text-ob-dim); font-size: 11px; }
.studio-profile-field input, .studio-profile-field textarea, .studio-profile-handle { width: 100%; border: 1px solid var(--border-hairline); border-radius: 10px; outline: none; background: var(--background-primary); color: inherit; }
.studio-profile-field input, .studio-profile-field textarea { padding: 10px 12px; font: inherit; }
.studio-profile-field textarea { resize: vertical; line-height: 1.7; }
.studio-profile-handle { display: flex; align-items: center; overflow: hidden; }
.studio-profile-handle > span { padding-left: 12px; color: var(--text-ob-dim); font-family: ui-monospace, SFMono-Regular, Consolas, monospace; }
.studio-profile-handle input { border: 0; border-radius: 0; }
.studio-profile-field input:focus, .studio-profile-field textarea:focus, .studio-profile-handle:focus-within { border-color: var(--color-ob); }
.studio-profile-field small { justify-self: end; }
.studio-public-card { display: grid; justify-items: center; padding: 28px 20px; border: 1px solid var(--border-hairline); border-radius: 16px; background: radial-gradient(circle at 50% 0, rgba(97, 73, 184, .2), transparent 46%), var(--background-primary); text-align: center; }
.studio-public-card img { width: 92px; height: 92px; border-radius: 50%; object-fit: cover; }
.studio-public-card > span { margin-top: 12px; color: var(--color-ob); font-size: 11px; }
.studio-public-card h2 { margin: 5px 0 8px; }
.studio-public-card p { margin: 0 0 12px; color: var(--text-ob-dim); font-size: 12px; line-height: 1.7; }
.studio-public-card a { color: var(--color-ob); font-size: 11px; }
.studio-profile-progress { display: grid; grid-template-columns: auto 1fr; gap: 12px; align-items: center; }
.studio-profile-progress strong { font-size: 1.8rem; }
.studio-profile-progress > span { height: 5px; overflow: hidden; border-radius: 999px; background: color-mix(in srgb, var(--text-ob-dim) 18%, transparent); }
.studio-profile-progress i { display: block; height: 100%; border-radius: inherit; background: var(--color-ob); transition: width .2s ease; }
.studio-profile-completion ul { display: grid; gap: 8px; margin: 18px 0 12px; padding: 0; list-style: none; color: var(--text-ob-dim); font-size: 12px; }
.studio-profile-completion li { display: flex; gap: 8px; }
.studio-profile-completion li.done { color: var(--color-ob); }
.studio-profile-completion > p { margin: 0; color: var(--text-ob-dim); font-size: 11px; line-height: 1.7; }
.studio-profile-footer { position: sticky; bottom: 12px; display: flex; align-items: center; justify-content: space-between; gap: 14px; margin-top: 18px; padding: 13px 16px; border: 1px solid var(--border-hairline); border-radius: 16px; background: color-mix(in srgb, var(--background-primary-alt) 96%, transparent); backdrop-filter: blur(16px); }
.studio-profile-footer > span { color: var(--text-ob-dim); font-size: 11px; }
.studio-profile-footer > div { display: flex; align-items: center; gap: 8px; }
.studio-profile-primary { border-color: transparent; background: var(--color-ob); color: #081127; font-weight: 700; }
.studio-profile-primary:disabled, .studio-profile-avatar button:disabled { opacity: .55; cursor: wait; }
.studio-profile-link { color: var(--color-ob); font-size: 12px; text-decoration: none; }
.studio-profile-leave { position: fixed; z-index: 3000; inset: 0; display: grid; place-items: center; padding: 20px; background: rgba(2, 6, 20, .72); backdrop-filter: blur(8px); }
.studio-profile-leave section { width: min(440px, 100%); padding: 26px; border: 1px solid var(--border-hairline); border-radius: 18px; background: var(--background-primary-alt); }
.studio-profile-leave h2 { margin: 0 0 10px; }
.studio-profile-leave p { color: var(--text-ob-dim); line-height: 1.7; }
.studio-profile-leave section > div { display: flex; justify-content: flex-end; gap: 8px; }
.studio-profile-leave button { padding: 9px 14px; border: 1px solid var(--border-hairline); border-radius: 999px; background: transparent; color: inherit; cursor: pointer; }
.studio-profile-leave button.danger { border-color: transparent; background: #df8177; color: #160a08; font-weight: 700; }
@media (max-width: 1000px) { .studio-profile-layout { grid-template-columns: 1fr; } .studio-profile-sidebar { position: static; grid-template-columns: repeat(2, minmax(0, 1fr)); } }
@media (max-width: 680px) { .studio-profile-head { grid-template-columns: 1fr; } .studio-profile-state { justify-self: start; } .studio-profile-identity, .studio-profile-sidebar { grid-template-columns: 1fr; } .studio-profile-footer { align-items: stretch; flex-direction: column; } .studio-profile-footer > div { display: grid; grid-template-columns: 1fr 1fr; } .studio-profile-link { grid-column: 1 / -1; text-align: center; } }
.studio-profile-links { display: grid; gap: 12px; }
.studio-profile-link-row { display: grid; grid-template-columns: minmax(120px, .7fr) minmax(220px, 1.4fr) minmax(140px, 1fr) auto; gap: 12px; align-items: end; padding: 14px; border: 1px solid var(--border-hairline); border-radius: 14px; }
.studio-profile-add-link, .studio-profile-remove-link { min-height: 34px; padding: 0 12px; border: 1px solid var(--border-hairline); border-radius: 999px; background: transparent; color: inherit; cursor: pointer; }
.studio-profile-add-link { color: var(--color-ob); }
.studio-profile-remove-link { color: #e2776c; }
.studio-profile-empty-links { margin: 0; color: var(--text-ob-dim); font-size: 13px; }
@media (max-width: 800px) { .studio-profile-link-row { grid-template-columns: 1fr 1fr; } .studio-profile-remove-link { justify-self: start; } }
</style>
