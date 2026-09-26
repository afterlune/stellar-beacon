<template>
  <section class="admin-page">
    <AdminPageHeader :title="t('site.website.title')" :description="t('site.website.description')">
      <template #actions>
        <a-button :loading="loading" @click="load">
          <template #icon><IconRefresh /></template>
          {{ t('site.website.reload') }}
        </a-button>
      </template>
    </AdminPageHeader>

    <a-card class="admin-form-panel admin-form-card" :bordered="false">
      <a-alert v-if="errorMessage" type="error" closable @close="errorMessage = ''">{{ errorMessage }}</a-alert>
      <a-alert v-if="unknownFieldCount > 0" type="info" :closable="false">
        {{ t('site.website.unknownFields', { count: unknownFieldCount }) }}
      </a-alert>

      <a-spin v-if="!ready" class="website-loading" :tip="t('site.website.loading')" />
      <template v-else>
        <!-- A hand-rolled panel instead of `a-tabs`: Arco keeps every tab pane
             mounted (hidden with CSS), which would leave several textareas in
             the document and break keyboard/AT ordering. Only the active
             section is rendered here. -->
        <div class="config-sections" role="tablist" :aria-label="t('site.website.sectionsLabel')">
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
              <a-form-item field="name" :label="t('site.website.name')">
                <a-input v-model="form.name" :placeholder="t('site.website.namePlaceholder')" />
              </a-form-item>
              <a-form-item field="englishName" :label="t('site.website.englishName')">
                <a-input v-model="form.englishName" :placeholder="t('site.website.englishNamePlaceholder')" />
              </a-form-item>
            </div>
            <a-form-item field="notice" :label="t('site.website.notice')">
              <a-textarea v-model="form.notice" :auto-size="{ minRows: 3, maxRows: 8 }" :placeholder="t('site.website.noticePlaceholder')" />
            </a-form-item>
            <div class="config-grid">
              <a-form-item field="websiteCreateTime" :label="t('site.website.createTime')">
                <a-input v-model="form.websiteCreateTime" :placeholder="t('site.website.createTimePlaceholder')" />
                <template #help>{{ t('site.website.createTimeHelp') }}</template>
              </a-form-item>
              <a-form-item field="beianNumber" :label="t('site.website.beian')">
                <a-input v-model="form.beianNumber" :placeholder="t('site.website.beianPlaceholder')" />
              </a-form-item>
              <a-form-item field="multiLanguage" :label="t('site.website.multiLanguage')">
                <a-switch v-model="switches.multiLanguage" :checked-value="1" :unchecked-value="0" />
                <template #help>{{ t('site.website.multiLanguageHelp') }}</template>
              </a-form-item>
            </div>
          </a-form>

          <a-form v-else-if="activeSection === 'images'" class="config-form" :model="form" layout="vertical" @submit-success="save">
            <a-form-item field="logo" :label="t('site.website.logo')">
              <div class="config-image-field">
                <AdminImagePreview v-if="isHttpUrl(form.logo)" :src="form.logo" :alt="t('site.website.logo')" :width="120" :height="80" />
                <div class="config-image-input">
                  <a-input v-model="form.logo" :placeholder="t('site.website.imageUrlPlaceholder')" />
                  <a-space wrap>
                    <a-button @click="openPicker('logo')">{{ t('site.website.pickFromLibrary') }}</a-button>
                    <a-button :disabled="!form.logo" @click="form.logo = ''">{{ t('site.website.clear') }}</a-button>
                  </a-space>
                  <span class="admin-field-hint">{{ t('site.website.pickHint') }}</span>
                </div>
              </div>
            </a-form-item>
            <div class="config-grid">
              <a-form-item v-for="field in imageFields" :key="field.key" :label="field.label">
                <a-input v-model="form[field.key]" :placeholder="t('site.website.imageUrlPlaceholder')" />
                <a-button type="text" size="mini" @click="openPicker(field.key)">{{ t('site.website.pickFromLibrary') }}</a-button>
              </a-form-item>
            </div>
          </a-form>

          <a-form v-else class="config-form" :model="form" layout="vertical" @submit-success="save">
            <div class="config-switch-list">
              <div class="config-switch-item">
                <div>
                  <strong>{{ t('site.website.commentReview') }}</strong>
                  <small>{{ t('site.website.commentReviewHint') }}</small>
                </div>
                <a-switch v-model="switches.isCommentReview" :checked-value="1" :unchecked-value="0" />
              </div>
              <div class="config-switch-item">
                <div>
                  <strong>{{ t('site.website.emailNotice') }}</strong>
                  <small>{{ t('site.website.emailNoticeHint') }}</small>
                </div>
                <a-switch v-model="switches.isEmailNotice" :checked-value="1" :unchecked-value="0" />
              </div>
            </div>
          </a-form>
        </div>
      </template>

      <div v-if="ready" class="admin-form-actions website-actions">
        <a-button type="primary" :loading="saving" @click="save">{{ t('common.save') }}</a-button>
        <a-button :disabled="saving || !dirty" @click="reset">{{ t('site.revert') }}</a-button>
        <span class="website-status">
          <template v-if="dirty">{{ t('site.unsavedChanges') }}</template>
          <template v-else>{{ t('site.synced') }}</template>
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
import { t } from '@/i18n'
import { isHttpUrl } from '@/utils/format'

/**
 * Fields the editor renders as text inputs. Anything the backend returns
 * outside this set (or the switch set) is preserved verbatim on save, so the
 * payload stays complete even when the server adds new configuration keys.
 */
const TEXT_FIELDS = [
  'name', 'englishName', 'logo', 'notice', 'websiteCreateTime', 'beianNumber',
  'touristAvatar', 'userAvatar'
] as const

const SWITCH_FIELDS = ['multiLanguage', 'isCommentReview', 'isEmailNotice'] as const

const imageFields = computed<ImageField[]>(() => [
  { key: 'userAvatar', label: t('site.website.userAvatar') },
  { key: 'touristAvatar', label: t('site.website.touristAvatar') }
])

const sections = computed<SectionDef[]>(() => [
  { key: 'basic', label: t('site.website.sectionBasic'), hint: t('site.website.sectionBasicHint') },
  { key: 'images', label: t('site.website.sectionImages'), hint: t('site.website.sectionImagesHint') },
  { key: 'interaction', label: t('site.website.sectionInteraction'), hint: t('site.website.sectionInteractionHint') }
])

type SectionKey = 'basic' | 'images' | 'interaction'

type ImageField = { key: TextField; label: string }
type SectionDef = { key: SectionKey; label: string; hint: string }

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
    errorMessage.value = apiErrorMessage(error, t('site.website.loadFailed'))
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
    Message.success(t('site.website.saveSuccess'))
  } catch (error) {
    errorMessage.value = apiErrorMessage(error, t('site.website.saveFailed'))
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
  Message.success(t('site.website.imageApplied'))
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
