<template>
  <section>
    <a-card :title="isEditing ? '编辑说说' : '发布说说'">
      <a-alert type="info" :show-icon="true" :closable="false">说说内容按纯文本/已有 HTML 原样交给后端处理，图片只能通过后端对象存储接口上传。</a-alert>
      <a-alert v-if="errorMessage" type="error" closable @close="errorMessage = ''">{{ errorMessage }}</a-alert>
      <a-form class="talk-form" :model="editor" layout="vertical" @submit-success="save">
        <a-form-item field="content" label="内容" :rules="[{ required: true, message: '内容不能为空' }]">
          <a-textarea v-model="editor.content" :max-length="100000" show-word-limit :auto-size="{ minRows: 12, maxRows: 28 }" />
        </a-form-item>
        <a-form-item label="图片">
          <a-space direction="vertical" fill>
            <a-space wrap>
              <a-tag v-for="image in editor.images" :key="image" closable @close="removeImage(image)">{{ image }}</a-tag>
              <span v-if="editor.images.length === 0" class="field-hint">暂无图片</span>
            </a-space>
            <a-space>
              <input ref="imageInput" type="file" accept="image/*" hidden @change="selectImage" />
              <a-button :loading="uploading" @click="imageInput?.click()">上传图片</a-button>
              <span class="field-hint">上传地址由后端返回，不接受前端直接拼接对象存储 URL。</span>
            </a-space>
          </a-space>
        </a-form-item>
        <a-space>
          <span>状态</span>
          <a-radio-group v-model="editor.status">
            <a-radio :value="1">公开</a-radio>
            <a-radio :value="2">私密</a-radio>
          </a-radio-group>
          <a-checkbox v-model="editor.isTop" :checked-value="1" :unchecked-value="0">置顶</a-checkbox>
        </a-space>
        <a-space class="form-actions">
          <a-button type="primary" html-type="submit" :loading="saving">保存</a-button>
          <a-button @click="router.push('/talk-list')">返回列表</a-button>
        </a-space>
      </a-form>
    </a-card>
  </section>
</template>

<script setup lang="ts">
import { computed, onMounted, reactive, ref } from 'vue'
import { Message } from '@arco-design/web-vue'
import { useRoute, useRouter } from 'vue-router'

import { apiErrorMessage, getAdminTalk, saveAdminTalk, uploadAdminTalkImage } from '@/api/http'

const route = useRoute()
const router = useRouter()
const saving = ref(false)
const uploading = ref(false)
const errorMessage = ref('')
const imageInput = ref<HTMLInputElement | null>(null)
const editor = reactive({ id: 0, content: '', images: [] as string[], isTop: 0, status: 1 })
const talkId = computed(() => String(route.params.talkId || route.params.id || route.params.articleId || ''))
const isEditing = computed(() => /^\d+$/.test(talkId.value))

onMounted(() => {
  if (isEditing.value) void load()
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
  }
}

async function save(): Promise<void> {
  if (!editor.content.trim()) {
    Message.error('说说内容不能为空')
    return
  }
  saving.value = true
  try {
    await saveAdminTalk({
      id: editor.id || undefined,
      content: editor.content.trim(),
      images: JSON.stringify(editor.images),
      isTop: editor.isTop,
      status: editor.status
    })
    Message.success('说说已保存')
    await router.push('/talk-list')
  } catch (error) {
    errorMessage.value = apiErrorMessage(error, '说说保存失败')
  } finally {
    saving.value = false
  }
}

function removeImage(image: string): void {
  editor.images = editor.images.filter((item) => item !== image)
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
.talk-form { margin-top: 18px; }
.field-hint { color: var(--color-text-3); font-size: 12px; }
.form-actions { margin-top: 24px; }
</style>
