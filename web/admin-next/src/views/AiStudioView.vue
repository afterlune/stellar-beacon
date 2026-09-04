<template>
  <section class="ai-studio">
    <a-alert type="info" :show-icon="true" :closable="false">
      AI 只生成预览和审核记录，不会自动写入文章。发布前必须经过明确的人工审核动作。
    </a-alert>
    <a-card title="AI 写作工作台" class="ai-card">
      <a-form :model="form" layout="vertical" @submit-success="generate">
        <div class="ai-form-grid">
          <a-form-item field="operation" label="操作">
            <a-select v-model="form.operation">
              <a-option value="continue">续写</a-option>
              <a-option value="polish">润色</a-option>
              <a-option value="summary">摘要</a-option>
              <a-option value="title">标题</a-option>
              <a-option value="correct">纠错</a-option>
            </a-select>
          </a-form-item>
          <a-form-item field="articleId" label="文章 ID（可选）">
            <a-input v-model="form.articleId" inputmode="numeric" placeholder="关联已有文章" />
          </a-form-item>
        </div>
        <a-form-item field="title" label="标题">
          <a-input v-model="form.title" :max-length="256" show-word-limit />
        </a-form-item>
        <a-form-item field="content" label="正文" :rules="[{ required: true, message: '正文不能为空' }]">
          <a-textarea v-model="form.content" :max-length="100000" show-word-limit :auto-size="{ minRows: 8, maxRows: 20 }" />
        </a-form-item>
        <a-form-item field="instruction" label="补充要求">
          <a-textarea v-model="form.instruction" :max-length="2000" :auto-size="{ minRows: 3, maxRows: 8 }" />
        </a-form-item>
      <a-button type="primary" html-type="submit" :loading="generating">生成预览</a-button>
    </a-form>
    </a-card>

    <a-card title="AI 视觉理解（默认关闭）" class="ai-card">
      <a-alert type="info" :show-icon="true" :closable="false">
        图片只用于本次理解请求，结果会进入审核队列，不会自动修改文章。图片 URL 与 base64 二选一。
      </a-alert>
      <a-form :model="visionForm" layout="vertical" @submit-success="generateVision">
        <a-form-item field="prompt" label="理解要求" :rules="[{ required: true, message: '理解要求不能为空' }]">
          <a-textarea v-model="visionForm.prompt" :max-length="4000" show-word-limit :auto-size="{ minRows: 3, maxRows: 8 }" placeholder="例如：请描述图片中的主要对象和可见文字" />
        </a-form-item>
        <a-form-item field="imageUrl" label="图片 URL（可选）">
          <a-input v-model="visionForm.imageUrl" placeholder="https://example.com/image.png" @input="clearVisionBase64" />
        </a-form-item>
        <a-form-item label="上传图片（可选）">
          <input type="file" accept="image/jpeg,image/png,image/gif,image/webp" @change="selectVisionFile" />
          <span v-if="visionFileName" class="vision-file-name">{{ visionFileName }}</span>
        </a-form-item>
        <a-form-item field="imageBase64" label="图片 base64（可选）">
          <a-textarea v-model="visionForm.imageBase64" :max-length="6291456" show-word-limit :auto-size="{ minRows: 3, maxRows: 8 }" placeholder="原始 base64，不要包含 data:image/... 前缀" @input="clearVisionUrl" />
        </a-form-item>
        <div class="ai-form-grid">
          <a-form-item field="mimeType" label="MIME 类型（base64 必填）">
            <a-select v-model="visionForm.mimeType">
              <a-option value="image/png">image/png</a-option>
              <a-option value="image/jpeg">image/jpeg</a-option>
              <a-option value="image/webp">image/webp</a-option>
              <a-option value="image/gif">image/gif</a-option>
            </a-select>
          </a-form-item>
          <a-form-item field="detail" label="图片细节级别">
            <a-select v-model="visionForm.detail">
              <a-option value="auto">自动</a-option>
              <a-option value="low">低</a-option>
              <a-option value="high">高</a-option>
            </a-select>
          </a-form-item>
        </div>
        <a-button type="primary" html-type="submit" :loading="visionGenerating">生成视觉预览</a-button>
      </a-form>
    </a-card>

    <a-card v-if="visionReview" title="当前视觉预览" class="ai-card">
      <template #extra><a-tag color="arcoblue">{{ visionReview.operation }} · {{ visionReview.runId }}</a-tag></template>
      <pre class="ai-preview vision-preview">{{ visionReview.preview }}</pre>
      <a-typography-text type="secondary">结果已进入审核队列，请在审核队列中执行人工操作。</a-typography-text>
    </a-card>

    <a-card v-if="currentReview" title="当前预览" class="ai-card">
      <template #extra><a-tag color="arcoblue">{{ currentReview.operation }} · {{ currentReview.runId }}</a-tag></template>
      <div class="ai-preview-grid">
        <div>
          <h3>生成内容</h3>
          <pre class="ai-preview">{{ currentReview.preview }}</pre>
        </div>
        <div>
          <h3>Diff</h3>
          <pre class="ai-diff">{{ currentReview.diff || '无差异说明' }}</pre>
        </div>
      </div>
      <a-space wrap>
        <a-button type="primary" @click="approve">记录接受</a-button>
        <a-button @click="partialDialog = true">部分接受</a-button>
        <a-button status="danger" @click="rejectDialog = true">记录拒绝</a-button>
        <a-button @click="regenerate">记录重新生成</a-button>
      </a-space>
    </a-card>

    <a-card title="审核队列" class="ai-card">
      <a-table :data="reviews" :columns="reviewColumns" :loading="listLoading" :pagination="pagination" row-key="id" @page-change="changePage">
        <template #status="{ record }">
          <a-tag :color="statusColor(record.status)">{{ record.status }}</a-tag>
        </template>
        <template #createdAt="{ record }">{{ formatTime(record.createdAt) }}</template>
        <template #operations="{ record }">
          <a-button type="text" size="small" @click="selectReview(record)">查看</a-button>
        </template>
      </a-table>
    </a-card>

    <a-modal v-model:visible="partialDialog" title="部分接受" @before-ok="partialAccept">
      <a-textarea v-model="partialContent" :max-length="100000" show-word-limit :auto-size="{ minRows: 8, maxRows: 16 }" />
    </a-modal>
    <a-modal v-model:visible="rejectDialog" title="拒绝生成物" @before-ok="reject">
      <a-textarea v-model="rejectReason" :max-length="2000" placeholder="填写拒绝原因" />
    </a-modal>
  </section>
</template>

<script setup lang="ts">
import { computed, onMounted, reactive, ref } from 'vue'
import { Message } from '@arco-design/web-vue'

import { apiErrorMessage, listAIReviews, previewVision, previewWriting, reviewAction } from '@/api/http'
import type { AIReview, AIVisionPreview, AIWritingPreview } from '@shared/api-contract'

const form = reactive({ operation: 'polish', articleId: '', title: '', content: '', instruction: '' })
const visionForm = reactive({ prompt: '', imageUrl: '', imageBase64: '', mimeType: 'image/png', detail: 'auto' })
const currentReview = ref<AIWritingPreview | null>(null)
const visionReview = ref<AIVisionPreview | null>(null)
const reviews = ref<AIReview[]>([])
const generating = ref(false)
const visionGenerating = ref(false)
const visionFileName = ref('')
const listLoading = ref(false)
const current = ref(1)
const size = ref(10)
const total = ref(0)
const partialDialog = ref(false)
const rejectDialog = ref(false)
const partialContent = ref('')
const rejectReason = ref('')

const reviewColumns = [
  { title: '操作', dataIndex: 'operation' },
  { title: '目标', dataIndex: 'targetId' },
  { title: '状态', dataIndex: 'status', slotName: 'status' },
  { title: 'Run ID', dataIndex: 'runId', ellipsis: true, tooltip: true },
  { title: '创建时间', dataIndex: 'createdAt', slotName: 'createdAt' },
  { title: '操作', slotName: 'operations' }
]
const pagination = computed(() => ({ current: current.value, pageSize: size.value, total: total.value, showTotal: true }))

onMounted(() => void loadReviews())

async function generate(): Promise<void> {
  if (!form.content.trim()) return
  generating.value = true
  try {
    const payload: Record<string, unknown> = { operation: form.operation, title: form.title, content: form.content, instruction: form.instruction }
    if (form.articleId.trim()) payload.articleId = Number(form.articleId)
    currentReview.value = await previewWriting(payload)
    partialContent.value = currentReview.value.preview
    await loadReviews()
    Message.success('预览已生成，等待审核')
  } catch (error) {
    Message.error(apiErrorMessage(error, '生成预览失败'))
  } finally {
    generating.value = false
  }
}

async function generateVision(): Promise<void> {
  if (!visionForm.prompt.trim() || (!visionForm.imageUrl.trim() && !visionForm.imageBase64.trim())) return
  visionGenerating.value = true
  try {
    const payload: Record<string, unknown> = {
      prompt: visionForm.prompt,
      imageUrl: visionForm.imageUrl,
      imageBase64: visionForm.imageBase64,
      mimeType: visionForm.mimeType,
      detail: visionForm.detail
    }
    visionReview.value = await previewVision(payload)
    await loadReviews()
    Message.success('视觉预览已生成，等待审核')
  } catch (error) {
    Message.error(apiErrorMessage(error, '生成视觉预览失败'))
  } finally {
    visionGenerating.value = false
  }
}

function clearVisionBase64(): void {
  if (visionForm.imageUrl.trim()) {
    visionForm.imageBase64 = ''
    visionFileName.value = ''
  }
}

function clearVisionUrl(): void {
  if (visionForm.imageBase64.trim()) visionForm.imageUrl = ''
}

function selectVisionFile(event: Event): void {
  const input = event.target as HTMLInputElement
  const file = input.files?.[0]
  if (!file) return
  if (file.size > 4 * 1024 * 1024) {
    Message.error('图片不能超过 4 MiB')
    input.value = ''
    return
  }
  if (!['image/jpeg', 'image/png', 'image/gif', 'image/webp'].includes(file.type)) {
    Message.error('仅支持 JPEG、PNG、GIF 或 WebP 图片')
    input.value = ''
    return
  }
  const reader = new FileReader()
  reader.onload = () => {
    const value = typeof reader.result === 'string' ? reader.result : ''
    const separator = value.indexOf(',')
    if (separator < 0) {
      Message.error('图片读取失败')
      return
    }
    visionForm.imageBase64 = value.slice(separator + 1)
    visionForm.imageUrl = ''
    visionForm.mimeType = file.type
    visionFileName.value = file.name
  }
  reader.onerror = () => Message.error('图片读取失败')
  reader.readAsDataURL(file)
}

async function loadReviews(): Promise<void> {
  listLoading.value = true
  try {
    const page = await listAIReviews(current.value, size.value)
    reviews.value = page.records
    total.value = page.count
  } catch (error) {
    Message.error(apiErrorMessage(error, '加载审核记录失败'))
  } finally {
    listLoading.value = false
  }
}

function selectReview(review: AIReview): void {
  currentReview.value = { reviewId: review.id, runId: review.runId, operation: review.operation, preview: review.content || '', diff: review.diff || '' }
  partialContent.value = review.content || ''
}

async function apply(action: 'approve' | 'partial' | 'reject' | 'regenerate' | 'expire', payload: Record<string, unknown> = {}): Promise<void> {
  if (!currentReview.value?.reviewId) return
  try {
    await reviewAction(currentReview.value.reviewId, action, payload)
    await loadReviews()
    Message.success('审核操作已记录')
  } catch (error) {
    Message.error(apiErrorMessage(error, '审核操作失败'))
  }
}

function approve(): void { void apply('approve') }

async function partialAccept(done: (closed: boolean) => void): Promise<void> {
  if (!partialContent.value.trim()) {
    Message.error('部分接受内容不能为空')
    done(false)
    return
  }
  await apply('partial', { content: partialContent.value })
  done(true)
}

async function reject(done: (closed: boolean) => void): Promise<void> {
  if (!rejectReason.value.trim()) {
    Message.error('拒绝原因不能为空')
    done(false)
    return
  }
  await apply('reject', { rejectReason: rejectReason.value })
  rejectReason.value = ''
  done(true)
}

function regenerate(): void {
  void apply('regenerate', { runId: currentReview.value?.runId || '', content: currentReview.value?.preview || '' })
}

function changePage(page: number): void {
  current.value = page
  void loadReviews()
}

function formatTime(value: unknown): string {
  return typeof value === 'string' ? value.replace('T', ' ').replace(/\.\d+Z$/, '') : '—'
}

function statusColor(status: string): string {
  if (status === 'pending') return 'orange'
  if (status === 'approved' || status === 'partially_approved') return 'green'
  if (status === 'rejected' || status === 'expired') return 'red'
  return 'gray'
}
</script>

<style scoped>
.ai-studio {
  display: grid;
  gap: 16px;
}

.ai-form-grid,
.ai-preview-grid {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: 16px;
}

.ai-preview,
.ai-diff {
  min-height: 180px;
  max-height: 460px;
  overflow: auto;
  padding: 16px;
  white-space: pre-wrap;
  word-break: break-word;
  border-radius: 8px;
}

.ai-preview { background: var(--color-fill-2); }
.ai-diff { color: #e5e7eb; background: #111827; }
.vision-preview { min-height: 140px; }
.vision-file-name { margin-left: 12px; color: var(--color-text-2); }

@media (max-width: 900px) {
  .ai-form-grid,
  .ai-preview-grid { grid-template-columns: 1fr; }
}
</style>
