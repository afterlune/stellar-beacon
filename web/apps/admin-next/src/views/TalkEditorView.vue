<template>
  <section class="admin-page">
    <AdminPageHeader
      :title="isEditing ? '编辑说说' : '发布说说'"
      :description="isEditing ? '设置说说内容、图片和可见范围。' : '发布短动态和图片。'"
      eyebrow="STELLAR BEACON / 说说" />
    <a-card class="admin-form-panel admin-form-card" :bordered="false">
      <a-alert v-if="errorMessage" type="error" closable @close="errorMessage = ''">{{ errorMessage }}</a-alert>
      <a-spin v-if="!editorReady" class="talk-editor-loading" tip="正在加载说说…" />
      <a-form v-else ref="formRef" class="talk-form" :model="editor" layout="vertical">
        <a-form-item field="content" label="内容" :rules="[{ required: true, message: '内容不能为空' }]">
          <a-textarea v-model="editor.content" class="talk-content-editor" :max-length="100000" show-word-limit :auto-size="{ minRows: 12, maxRows: 28 }" />
        </a-form-item>
        <a-form-item label="图片">
          <a-space direction="vertical" fill>
            <div v-if="editor.images.length" class="talk-image-grid">
              <AdminImagePreview
                v-for="image in editor.images"
                :key="image"
                :src="image"
                alt="说说图片"
                :width="112"
                :height="78" />
            </div>
            <template v-else>
              <span class="field-hint">暂无图片，可以上传 1–9 张配图。</span>
            </template>
            <a-space wrap>
              <input ref="imageInput" type="file" accept="image/*" hidden @change="selectImage" />
              <a-button :loading="uploading" @click="imageInput?.click()">上传图片</a-button>
              <a-button v-if="editor.images.length" @click="editor.images = []">清空图片</a-button>
              <span class="field-hint">上传地址由后端返回，不接受前端直接拼接对象存储 URL。</span>
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
          <span>状态</span>
          <a-radio-group v-model="editor.status">
            <a-radio :value="1">公开</a-radio>
            <a-radio :value="2">私密</a-radio>
          </a-radio-group>
          <AdminFlagCheckbox v-model="editor.isTop">置顶</AdminFlagCheckbox>
        </a-space>
        <div class="admin-form-actions">
          <a-button type="primary" :loading="saving" @click="submit">保存</a-button>
          <a-button :disabled="saving" @click="router.push('/talk-list')">取消</a-button>
        </div>
      </a-form>
    </a-card>

    <AdminLeaveGuard :visible="leaveVisible" @ok="confirmLeave" @cancel="cancelLeave">
      当前页面还有未保存的修改，离开后这些内容会丢失。
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

const route = useRoute()
const router = useRouter()
const saving = ref(false)
const uploading = ref(false)
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
    errorMessage.value = apiErrorMessage(error, '说说加载失败')
  } finally {
    editorReady.value = true
    markClean()
  }
}

async function save(): Promise<void> {
  if (!editor.content.trim()) {
    Message.error('说说内容不能为空')
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
    Message.success(isEditing.value ? '说说已更新' : '说说已发布')
    await router.push('/talk-list')
  } catch (error) {
    errorMessage.value = apiErrorMessage(error, '说说保存失败')
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
    Message.error('请先填写说说内容')
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
  const file = input.files?.[0]
  input.value = ''
  if (!file) return
  uploading.value = true
  try {
    editor.images.push(await uploadAdminTalkImage(file))
    Message.success('图片上传成功')
  } catch (error) {
    Message.error(apiErrorMessage(error, '图片上传失败'))
  } finally {
    uploading.value = false
  }
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
</style>
