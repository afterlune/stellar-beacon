<template>
  <section class="admin-page">
    <AdminPageHeader title="网站配置" description="集中维护博客的名称、作者信息、对外链接与互动开关。">
      <template #actions>
        <a-button :loading="loading" @click="load">
          <template #icon><IconRefresh /></template>
          重新载入
        </a-button>
      </template>
    </AdminPageHeader>

    <a-card class="admin-form-panel admin-form-card" :bordered="false">
      <a-alert v-if="errorMessage" type="error" closable @close="errorMessage = ''">{{ errorMessage }}</a-alert>
      <a-alert v-if="unknownFieldCount > 0" type="info" :closable="false">
        后端还返回了 {{ unknownFieldCount }} 个未在此表单展示的配置项，保存时会原样保留。
      </a-alert>

      <a-spin v-if="!ready" class="website-loading" tip="正在加载站点配置…" />
      <template v-else>
        <!-- A hand-rolled panel instead of `a-tabs`: Arco keeps every tab pane
             mounted (hidden with CSS), which would leave several textareas in
             the document and break keyboard/AT ordering. Only the active
             section is rendered here. -->
        <div class="config-sections" role="tablist" aria-label="网站配置分区">
          <button
            v-for="section in sections"
            :key="section.key"
            type="button"
            role="tab"
            class="config-section-tab"
            :class="{ 'is-active': activeSection === section.key }"
            :aria-selected="activeSection === section.key"
            :aria-controls="`config-panel-${section.key}`"
            @click="activeSection = section.key">
            <span class="config-section-tab-label">{{ section.label }}</span>
            <small>{{ section.hint }}</small>
          </button>
        </div>

        <div :id="`config-panel-${activeSection}`" role="tabpanel" class="config-panel">
          <a-form v-if="activeSection === 'basic'" class="config-form" :model="form" layout="vertical" @submit-success="save">
            <div class="config-grid">
              <a-form-item field="name" label="网站名称">
                <a-input v-model="form.name" placeholder="展示在浏览器标签与页头" />
              </a-form-item>
              <a-form-item field="englishName" label="英文名称">
                <a-input v-model="form.englishName" placeholder="用于 Logo 与国际化展示" />
              </a-form-item>
            </div>
            <a-form-item field="notice" label="公告">
              <a-textarea v-model="form.notice" :auto-size="{ minRows: 3, maxRows: 8 }" placeholder="展示在博客顶部的公告内容" />
            </a-form-item>
            <div class="config-grid">
              <a-form-item field="websiteCreateTime" label="建站时间">
                <a-input v-model="form.websiteCreateTime" placeholder="例如 2024-01-01" />
                <template #help>用于前台展示站点运行时长。</template>
              </a-form-item>
              <a-form-item field="beianNumber" label="备案号">
                <a-input v-model="form.beianNumber" placeholder="例如 京ICP备00000000号" />
              </a-form-item>
              <a-form-item field="multiLanguage" label="多语言">
                <a-switch v-model="switches.multiLanguage" :checked-value="1" :unchecked-value="0" />
                <template #help>开启后前台展示语言切换入口。</template>
              </a-form-item>
            </div>
          </a-form>

          <a-form v-else-if="activeSection === 'author'" class="config-form" :model="form" layout="vertical" @submit-success="save">
            <div class="config-grid">
              <a-form-item field="author" label="作者">
                <a-input v-model="form.author" placeholder="站点作者名" />
              </a-form-item>
              <a-form-item field="authorAvatar" label="作者头像">
                <a-input v-model="form.authorAvatar" placeholder="HTTPS 图片地址" />
              </a-form-item>
            </div>
            <a-form-item field="authorIntro" label="作者简介">
              <a-textarea v-model="form.authorIntro" :auto-size="{ minRows: 2, maxRows: 5 }" placeholder="一句话介绍作者" />
            </a-form-item>
            <div class="config-grid">
              <a-form-item v-for="field in socialFields" :key="field.key" :label="field.label">
                <a-input v-model="form[field.key]" :placeholder="field.placeholder" />
              </a-form-item>
            </div>
          </a-form>

          <a-form v-else-if="activeSection === 'images'" class="config-form" :model="form" layout="vertical" @submit-success="save">
            <a-form-item field="logo" label="站点 Logo">
              <div class="config-image-field">
                <AdminImagePreview v-if="isHttpUrl(form.logo)" :src="form.logo" alt="站点 Logo" :width="120" :height="80" />
                <div class="config-image-input">
                  <a-input v-model="form.logo" placeholder="HTTPS 图片地址" />
                  <a-space wrap>
                    <a-button @click="openPicker('logo')">从资源库选择</a-button>
                    <a-button :disabled="!form.logo" @click="form.logo = ''">清除</a-button>
                  </a-space>
                  <span class="admin-field-hint">先在「图片资源」上传文件，再从这里选择，地址会自动填入。</span>
                </div>
              </div>
            </a-form-item>
            <div class="config-grid">
              <a-form-item v-for="field in imageFields" :key="field.key" :label="field.label">
                <a-input v-model="form[field.key]" placeholder="HTTPS 图片地址" />
                <a-button type="text" size="mini" @click="openPicker(field.key)">从资源库选择</a-button>
              </a-form-item>
            </div>
          </a-form>

          <a-form v-else class="config-form" :model="form" layout="vertical" @submit-success="save">
            <div class="config-switch-list">
              <div class="config-switch-item">
                <div>
                  <strong>评论需要审核</strong>
                  <small>开启后，新评论需要管理员审核才会公开展示。</small>
                </div>
                <a-switch v-model="switches.isCommentReview" :checked-value="1" :unchecked-value="0" />
              </div>
              <div class="config-switch-item">
                <div>
                  <strong>邮件通知</strong>
                  <small>收到新评论或留言时发送邮件提醒。</small>
                </div>
                <a-switch v-model="switches.isEmailNotice" :checked-value="1" :unchecked-value="0" />
              </div>
              <div class="config-switch-item">
                <div>
                  <strong>开启赞赏</strong>
                  <small>在文章底部展示赞赏二维码。</small>
                </div>
                <a-switch v-model="switches.isReward" :checked-value="1" :unchecked-value="0" />
              </div>
            </div>
          </a-form>
        </div>
      </template>

      <div v-if="ready" class="admin-form-actions website-actions">
        <a-button type="primary" :loading="saving" @click="save">保存</a-button>
        <a-button :disabled="saving || !dirty" @click="reset">还原修改</a-button>
        <span class="website-status">
          <template v-if="dirty">有未保存的修改</template>
          <template v-else>已与服务器同步</template>
        </span>
      </div>
    </a-card>

    <AdminMediaPicker v-model="pickerVisible" @select="applyPickedImage" />
  </section>
</template>

<script setup lang="ts">
import { computed, onMounted, reactive, ref } from 'vue'
import { Message } from '@arco-design/web-vue'
import { IconRefresh } from '@arco-design/web-vue/es/icon'

import { apiErrorMessage, getWebsiteConfig, updateWebsiteConfig } from '@/api/http'
import AdminImagePreview from '@/components/AdminImagePreview.vue'
import AdminMediaPicker from '@/components/AdminMediaPicker.vue'
import AdminPageHeader from '@/components/AdminPageHeader.vue'
import { isHttpUrl } from '@/utils/format'

/**
 * Fields the editor renders as text inputs. Anything the backend returns
 * outside this set (or the switch set) is preserved verbatim on save, so the
 * payload stays complete even when the server adds new configuration keys.
 */
const TEXT_FIELDS = [
  'name', 'englishName', 'author', 'authorAvatar', 'authorIntro', 'logo', 'notice',
  'websiteCreateTime', 'beianNumber', 'github', 'gitee', 'qq', 'weChat', 'weibo',
  'csdn', 'zhihu', 'juejin', 'twitter', 'stackoverflow', 'touristAvatar',
  'userAvatar', 'weiXinQRCode', 'alipayQRCode'
] as const

const SWITCH_FIELDS = ['multiLanguage', 'isCommentReview', 'isEmailNotice', 'isReward'] as const

const socialFields = [
  { key: 'github', label: 'GitHub', placeholder: 'https://github.com/…' },
  { key: 'gitee', label: 'Gitee', placeholder: 'https://gitee.com/…' },
  { key: 'qq', label: 'QQ', placeholder: 'QQ 号或链接' },
  { key: 'weChat', label: '微信', placeholder: '微信号' },
  { key: 'weibo', label: '微博', placeholder: 'https://weibo.com/…' },
  { key: 'csdn', label: 'CSDN', placeholder: 'https://blog.csdn.net/…' },
  { key: 'zhihu', label: '知乎', placeholder: 'https://www.zhihu.com/…' },
  { key: 'juejin', label: '掘金', placeholder: 'https://juejin.cn/…' },
  { key: 'twitter', label: 'Twitter', placeholder: 'https://twitter.com/…' },
  { key: 'stackoverflow', label: 'Stack Overflow', placeholder: 'https://stackoverflow.com/…' }
] as const

const imageFields = [
  { key: 'userAvatar', label: '默认用户头像' },
  { key: 'touristAvatar', label: '游客头像' },
  { key: 'weiXinQRCode', label: '微信二维码' },
  { key: 'alipayQRCode', label: '支付宝二维码' }
] as const

const sections = [
  { key: 'basic', label: '基础信息', hint: '站点名称、公告与备案' },
  { key: 'author', label: '作者与社交', hint: '作者资料与外部链接' },
  { key: 'images', label: '图片资源', hint: 'Logo、头像与二维码' },
  { key: 'interaction', label: '互动与功能', hint: '评论审核与通知开关' }
] as const

type SectionKey = (typeof sections)[number]['key']

type TextField = (typeof TEXT_FIELDS)[number]
type SwitchField = (typeof SWITCH_FIELDS)[number]

const form = reactive<Record<TextField, string>>(
  Object.fromEntries(TEXT_FIELDS.map((key) => [key, ''])) as Record<TextField, string>
)
const switches = reactive<Record<SwitchField, number>>(
  Object.fromEntries(SWITCH_FIELDS.map((key) => [key, 0])) as Record<SwitchField, number>
)

const activeSection = ref<SectionKey>('basic')
const loading = ref(false)
const saving = ref(false)
const ready = ref(false)
const errorMessage = ref('')
const pickerVisible = ref(false)
const pickerTarget = ref<TextField>('logo')
const snapshot = ref('')
/** Backend keys this editor does not render, carried through on save. */
const passthrough = ref<Record<string, unknown>>({})

const dirty = computed(() => JSON.stringify(currentPayload()) !== snapshot.value)
const unknownFieldCount = computed(() => Object.keys(passthrough.value).length)

onMounted(() => void load())

async function load(): Promise<void> {
  loading.value = true
  errorMessage.value = ''
  try {
    const value = await getWebsiteConfig()
    const rest: Record<string, unknown> = {}
    for (const [key, item] of Object.entries(value)) {
      if (TEXT_FIELDS.includes(key as TextField)) {
        form[key as TextField] = item === null || item === undefined ? '' : String(item)
      } else if (SWITCH_FIELDS.includes(key as SwitchField)) {
        switches[key as SwitchField] = Number(item) || 0
      } else {
        rest[key] = item
      }
    }
    passthrough.value = rest
    snapshot.value = JSON.stringify(currentPayload())
  } catch (error) {
    errorMessage.value = apiErrorMessage(error, '网站配置加载失败')
  } finally {
    loading.value = false
    ready.value = true
  }
}

/** Merge edited fields with untouched backend fields. */
function currentPayload(): Record<string, unknown> {
  const payload: Record<string, unknown> = { ...passthrough.value }
  for (const key of TEXT_FIELDS) payload[key] = form[key]
  for (const key of SWITCH_FIELDS) payload[key] = switches[key]
  return payload
}

function reset(): void {
  if (!snapshot.value) return
  const restored = JSON.parse(snapshot.value) as Record<string, unknown>
  for (const key of TEXT_FIELDS) form[key] = String(restored[key] ?? '')
  for (const key of SWITCH_FIELDS) switches[key] = Number(restored[key]) || 0
}

async function save(): Promise<void> {
  saving.value = true
  errorMessage.value = ''
  try {
    const payload = currentPayload()
    await updateWebsiteConfig(payload)
    snapshot.value = JSON.stringify(payload)
    Message.success('网站配置已保存')
  } catch (error) {
    errorMessage.value = apiErrorMessage(error, '网站配置保存失败')
    Message.error(errorMessage.value)
  } finally {
    saving.value = false
  }
}

function openPicker(target: TextField): void {
  pickerTarget.value = target
  pickerVisible.value = true
}

/** Media-library selection fills whichever image field opened the picker. */
function applyPickedImage(asset: { url: string }): void {
  form[pickerTarget.value] = asset.url
  Message.success('已填入图片地址')
}
</script>

<style scoped>
.website-loading {
  display: flex;
  justify-content: center;
  padding: 64px 0;
}

.config-form {
  margin-top: 4px;
}

.config-sections {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(170px, 1fr));
  gap: 10px;
  margin-bottom: 22px;
}

.config-section-tab {
  display: grid;
  gap: 3px;
  padding: 12px 14px;
  border: 1px solid var(--admin-border);
  border-radius: var(--admin-radius-control);
  color: var(--admin-ink);
  background: var(--admin-surface);
  text-align: left;
  cursor: pointer;
  transition: border-color var(--admin-duration-fast) var(--admin-ease),
    background-color var(--admin-duration-fast) var(--admin-ease),
    color var(--admin-duration-fast) var(--admin-ease);
}

.config-section-tab:hover {
  border-color: var(--admin-brand-soft-strong);
  background: var(--admin-surface-soft);
}

.config-section-tab.is-active {
  border-color: var(--admin-brand);
  color: var(--admin-brand);
  background: var(--admin-brand-soft);
}

.config-section-tab-label {
  font-size: 13px;
  font-weight: 680;
}

.config-section-tab small {
  color: var(--admin-muted);
  font-size: 11px;
}

.config-section-tab.is-active small {
  color: var(--admin-brand);
  opacity: 0.85;
}

.config-image-field {
  display: flex;
  align-items: flex-start;
  gap: 16px;
}

.config-image-input {
  min-width: 0;
  display: grid;
  flex: 1;
  gap: 10px;
}

.config-switch-list {
  display: grid;
  gap: 10px;
}

.config-switch-item {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 20px;
  padding: 14px 16px;
  border: 1px solid var(--admin-border);
  border-radius: var(--admin-radius-control);
  background: var(--admin-surface-soft);
}

.config-switch-item strong {
  display: block;
  color: var(--admin-ink-strong);
  font-size: 13px;
  font-weight: 650;
}

.config-switch-item small {
  display: block;
  margin-top: 3px;
  color: var(--admin-muted);
  font-size: 12px;
}

.website-actions {
  margin-top: 4px;
}

.website-status {
  color: var(--admin-muted);
  font-size: 12px;
}
</style>
