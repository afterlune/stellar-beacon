<template>
  <section class="admin-page">
    <AdminPageHeader
      :title="isEditing ? t('comments.talks.editTitle') : t('comments.talks.create')"
      :description="isEditing ? t('comments.talks.editDescription') : t('comments.talks.createDescription')"
      :eyebrow="t('comments.talks.eyebrow')" />
    <a-card class="admin-form-panel admin-form-card" :bordered="false">
      <a-alert v-if="errorMessage" type="error" closable @close="errorMessage = ''">{{ errorMessage }}</a-alert>
      <a-spin v-if="!editorReady" class="talk-editor-loading" :tip="t('comments.talks.loading')" />
      <a-form v-else ref="formRef" class="talk-form" :model="editor" layout="vertical">
        <a-form-item field="content" :label="t('comments.common.content')" :rules="[{ required: true, message: t('comments.talks.contentRequired') }]">
          <a-textarea v-model="editor.content" class="talk-content-editor" :max-length="100000" show-word-limit :auto-size="{ minRows: 12, maxRows: 28 }" />
        </a-form-item>
        <a-form-item :label="t('common.image')">
          <a-space direction="vertical" fill>
            <div v-if="editor.images.length" class="talk-image-grid">
              <div
                v-for="(image, index) in editor.images"
                :key="image"
                class="talk-image-item"
                draggable="true"
                @dragstart="draggingIndex = index"
                @dragover.prevent
                @drop="dropImage(index)"
                @dragend="draggingIndex = -1">
                <AdminImagePreview
                  :src="image"
                  :alt="t('comments.talks.imageAlt')"
                  :width="132"
                  :height="92" />
                <button type="button" class="talk-image-remove" :aria-label="t('comments.talks.removeImage')" @click="removeImage(image)">×</button>
              </div>
            </div>
            <template v-else>
              <span class="field-hint">{{ t('comments.talks.noImages') }}</span>
            </template>
            <a-space wrap>
              <input ref="imageInput" type="file" accept="image/*" multiple hidden @change="selectImage" />
              <a-button :loading="uploading" @click="imageInput?.click()">{{ t('comments.talks.uploadImage') }}</a-button>
              <a-button v-if="editor.images.length" @click="editor.images = []">{{ t('comments.talks.clearImages') }}</a-button>
              <span class="field-hint">{{ t('comments.talks.uploadHint') }}</span>
            </a-space>
            <a-space v-if="editor.images.length" wrap :size="4">
              <a-tag
                v-for="image in editor.images"
                :key="`tag-${image}`"
                closable
                @close="removeImage(image)">
                {{ shortenUrl(image) }}
              </a-tag>
            </a-space>
          </a-space>
        </a-form-item>
        <a-space>
          <span>{{ t('common.status') }}</span>
          <a-radio-group v-model="editor.status">
            <a-radio :value="1">{{ t('status.published') }}</a-radio>
            <a-radio :value="2">{{ t('status.private') }}</a-radio>
          </a-radio-group>
          <AdminFlagCheckbox v-model="editor.isTop">{{ t('status.pinned') }}</AdminFlagCheckbox>
        </a-space>
        <div class="admin-form-actions">
          <a-button type="primary" :loading="saving" @click="submit">{{ t('common.save') }}</a-button>
          <a-button :disabled="saving" @click="router.push('/talk-list')">{{ t('common.cancel') }}</a-button>
        </div>
      </a-form>
    </a-card>

    <AdminLeaveGuard :visible="leaveVisible" @ok="confirmLeave" @cancel="cancelLeave">
      {{ t('comments.talks.unsavedHint') }}
    </AdminLeaveGuard>
  </section>
</template>

<script setup lang="ts">
import { computed, onMounted, reactive, ref } from 'vue'
import { Message } from '@arco-design/web-vue'
import { useRoute, useRouter } from 'vue-router'

import { apiErrorMessage, getAdminTalk, saveAdminTalk, uploadAdminTalkImage } from '@/api/http'
import AdminFlagCheckbox from '@/components/AdminFlagCheckbox.vue'
import AdminImagePreview from '@/components/AdminImagePreview.vue'
import AdminLeaveGuard from '@/components/AdminLeaveGuard.vue'
import AdminPageHeader from '@/components/AdminPageHeader.vue'
import { useUnsavedGuard } from '@/composables/useUnsavedGuard'
import { t } from '@/i18n'

const MAX_UPLOAD_BYTES = 10 * 1024 * 1024
const MAX_IMAGES = 9
const route = useRoute()
const router = useRouter()
const saving = ref(false)
const uploading = ref(false)
const draggingIndex = ref(-1)
const errorMessage = ref('')
const imageInput = ref<HTMLInputElement | null>(null)
const editorReady = ref(false)
const formRef = ref<{ validate: () => Promise<Record<string, unknown> | undefined> } | null>(null)
const editor = reactive({ id: 0, content: '', images: [] as string[], isTop: 0, status: 1 })
// `/talks/*` is normalised to `/talks/:articleId` (see types.ts), so the
// wildcard parameter must stay part of the lookup chain.
const talkId = computed(() => String(route.params.talkId || route.params.id || route.params.articleId || ''))
const isEditing = computed(() => /^\d+$/.test(talkId.value))

/** 只在内容/图片/可见性真正变化时拦截离开；加载完成即建立基线。 */
const { visible: leaveVisible, markClean, confirmLeave, cancelLeave } = useUnsavedGuard(() => JSON.stringify([
  editor.content,
  editor.images,
  editor.isTop,
  editor.status
]))

onMounted(() => {
  if (isEditing.value) {
    void load()
  } else {
    editorReady.value = true
    markClean()
  }
})

async function load(): Promise<void> {
  try {
    const talk = await getAdminTalk(Number(talkId.value))
    editor.id = Number(talk.id || 0)
    editor.content = String(talk.content || '')
    editor.images = normalizeImages(talk.imgs, talk.images)
    editor.isTop = Number(talk.isTop || 0)
    editor.status = Number(talk.status || 1)
  } catch (error) {
    errorMessage.value = apiErrorMessage(error, t('comments.talks.editorLoadFailed'))
  } finally {
    editorReady.value = true
    markClean()
  }
}

async function save(): Promise<void> {
  if (!editor.content.trim()) {
    Message.error(t('comments.talks.contentEmpty'))
    return
  }
  saving.value = true
  errorMessage.value = ''
  try {
    await saveAdminTalk({
      id: editor.id || undefined,
      content: editor.content.trim(),
      images: JSON.stringify(editor.images),
      isTop: editor.isTop,
      status: editor.status
    })
    Message.success(isEditing.value ? t('comments.talks.updated') : t('comments.talks.published'))
    await router.push('/talk-list')
  } catch (error) {
    errorMessage.value = apiErrorMessage(error, t('comments.talks.saveFailed'))
    Message.error(errorMessage.value)
  } finally {
    saving.value = false
  }
}

/** Header-level save: validate first, then persist exactly once. */
async function submit(): Promise<void> {
  if (saving.value) return
  const errors = await formRef.value?.validate()
  if (errors) {
    Message.error(t('comments.talks.fillContent'))
    return
  }
  await save()
}

function removeImage(image: string): void {
  editor.images = editor.images.filter((item) => item !== image)
}

/** Object-store URLs are long; show a recognizable tail in the tag list. */
function shortenUrl(url: string): string {
  const value = String(url || '')
  if (value.length <= 42) return value
  return `…${value.slice(-40)}`
}

async function selectImage(event: Event): Promise<void> {
  const input = event.target as HTMLInputElement
  const files = Array.from(input.files || [])
  input.value = ''
  if (!files.length) return
  uploading.value = true
  let uploaded = 0
  const failures: string[] = []
  try {
    for (const file of files) {
      if (editor.images.length + uploaded >= MAX_IMAGES) {
        failures.push(`${file.name}：${t('comments.talks.imageLimit')}`)
        continue
      }
      if (!file.type.startsWith('image/')) {
        failures.push(`${file.name}：${t('comments.talks.imageTypeInvalid')}`)
        continue
      }
      if (file.size > MAX_UPLOAD_BYTES) {
        failures.push(`${file.name}：${t('comments.talks.imageTooLarge')}`)
        continue
      }
      try {
        editor.images.push(await uploadAdminTalkImage(file))
        uploaded += 1
      } catch (error) {
        failures.push(`${file.name}：${apiErrorMessage(error, t('upload.failed'))}`)
      }
    }
    if (uploaded > 0) Message.success(t('comments.talks.uploadSuccessCount', { count: uploaded }))
    if (failures.length) Message.error(failures.join('；'))
  } finally {
    uploading.value = false
  }
}

function dropImage(index: number): void {
  const from = draggingIndex.value
  draggingIndex.value = -1
  if (from < 0 || from === index) return
  const [image] = editor.images.splice(from, 1)
  if (image) editor.images.splice(index, 0, image)
}

function normalizeImages(images: unknown, serialized: unknown): string[] {
  if (Array.isArray(images)) return images.map(String).filter(Boolean)
  if (typeof serialized !== 'string' || !serialized) return []
  try {
    const value: unknown = JSON.parse(serialized)
    return Array.isArray(value) ? value.map(String).filter(Boolean) : []
  } catch {
    return []
  }
}
</script>

<style scoped>
.talk-form { margin-top: 4px; }
.talk-editor-loading { display: flex; justify-content: center; padding: 56px 0; }
.field-hint { color: var(--admin-muted); font-size: 12px; }
.talk-image-grid {
  display: flex;
  flex-wrap: wrap;
  gap: 10px;
}

.talk-image-item {
  position: relative;
  cursor: grab;
}

.talk-image-item:active {
  cursor: grabbing;
}

.talk-image-remove {
  position: absolute;
  top: -7px;
  right: -7px;
  display: grid;
  width: 22px;
  height: 22px;
  place-items: center;
  border: 1px solid var(--admin-surface);
  border-radius: 50%;
  color: #fff;
  background: var(--admin-danger, #d14b58);
  cursor: pointer;
}
</style>
