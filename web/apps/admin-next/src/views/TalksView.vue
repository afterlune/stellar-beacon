<template>
  <section class="admin-page">
    <AdminPageHeader title="说说管理" description="记录那些不必写成文章的片刻。">
      <template #actions>
        <a-input-search
          v-model="keywords"
          class="admin-filter-input"
          placeholder="在本页搜索说说内容"
          allow-clear />
        <a-button v-if="keywords.trim()" @click="keywords = ''">清空搜索</a-button>
        <a-button type="primary" @click="router.push('/talks')">
          <template #icon><IconPlus /></template>
          新增
        </a-button>
      </template>
    </AdminPageHeader>

    <a-card class="admin-panel" :bordered="false">
      <div class="admin-table-toolbar">
        <div class="admin-table-toolbar-main">
          <a-radio-group v-model="statusFilter" type="button" size="small" @change="reload">
            <a-radio value="all">全部</a-radio>
            <a-radio value="1">公开</a-radio>
            <a-radio value="2">私密</a-radio>
          </a-radio-group>
          <a-button v-if="hasFilters" type="text" size="small" @click="resetFilters">重置筛选</a-button>
        </div>
        <div class="admin-table-toolbar-actions">
          <span class="admin-toolbar-caption">共 {{ total }} 条说说 · 本页 {{ visibleTalks.length }} 条</span>
          <a-button :loading="loading" size="small" @click="load">
            <template #icon><IconRefresh /></template>
            刷新
          </a-button>
        </div>
      </div>

      <a-alert v-if="errorMessage" type="error" closable @close="errorMessage = ''">{{ errorMessage }}</a-alert>

      <div class="admin-table-shell">
        <a-table
          :data="visibleTalks"
          :columns="columns"
          :loading="loading"
          :pagination="pagination"
          row-key="id"
          @page-change="changePage"
          @page-size-change="changePageSize">
          <template #id="{ record }"><span class="admin-id-cell">#{{ record.id }}</span></template>
          <template #content="{ record }">
            <span class="talk-content" :title="plainText(record.content)">{{ plainText(record.content) || '—' }}</span>
          </template>
          <template #images="{ record }">
            <div v-if="talkImages(record).length" class="talk-images">
              <AdminImagePreview :src="talkImages(record)[0]" alt="说说图片" :width="76" :height="54" />
              <a-tooltip v-if="talkImages(record).length > 1" :content="`共 ${talkImages(record).length} 张图片`">
                <span class="talk-image-count">+{{ talkImages(record).length - 1 }}</span>
              </a-tooltip>
            </div>
            <span v-else class="admin-muted-cell">—</span>
          </template>
          <template #status="{ record }">
            <AdminStatusTag :kind="Number(record.status) === 1 ? 'public' : 'private'" />
          </template>
          <template #top="{ record }">
            <a-tooltip :content="Number(record.isTop) === 1 ? '点击取消置顶' : '点击置顶这条说说'">
              <a-switch
                :model-value="Number(record.isTop) === 1"
                :loading="pendingTopId === Number(record.id)"
                @change="(value) => toggleTop(record, value)" />
            </a-tooltip>
          </template>
          <template #time="{ record }"><span class="admin-cell-nowrap">{{ formatDateTime(record.createTime) }}</span></template>
          <template #actions="{ record }">
            <a-space class="admin-action-space">
              <a-button type="text" size="small" @click="router.push(`/talks/${record.id}`)">编辑</a-button>
              <a-popconfirm content="确定删除这条说说吗？删除后无法恢复。" @ok="deleteTalk(record.id)">
                <a-button type="text" status="danger" size="small">删除</a-button>
              </a-popconfirm>
            </a-space>
          </template>
          <template #empty>
            <AdminEmptyState
              :icon="IconMessage"
              :title="hasFilters ? '没有匹配的说说' : '还没有说说'"
              :description="hasFilters ? '换个关键词或重置筛选条件再试一次。' : '发布第一条说说，记录当下的片段。'">
              <a-button v-if="hasFilters" size="small" @click="resetFilters">重置筛选</a-button>
              <a-button v-else type="primary" size="small" @click="router.push('/talks')">发布说说</a-button>
            </AdminEmptyState>
          </template>
        </a-table>
      </div>
    </a-card>
  </section>
</template>

<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { Message } from '@arco-design/web-vue'
import { IconMessage, IconPlus, IconRefresh } from '@arco-design/web-vue/es/icon'
import { useRouter } from 'vue-router'

import { apiErrorMessage, deleteAdminTalks, getAdminTalk, listAdminTalks, saveAdminTalk } from '@/api/http'
import AdminEmptyState from '@/components/AdminEmptyState.vue'
import AdminImagePreview from '@/components/AdminImagePreview.vue'
import AdminPageHeader from '@/components/AdminPageHeader.vue'
import AdminStatusTag from '@/components/AdminStatusTag.vue'
import { formatDateTime, isHttpUrl, plainText } from '@/utils/format'
import { tablePagination } from '@/utils/pagination'
import type { AdminTalk } from '@benetnasch/api-contract'

// `commentCount` is intentionally absent: the admin talk DTO does not expose it
// yet, so the column would always render an empty cell.
const columns = [
  { title: 'ID', dataIndex: 'id', width: 78, slotName: 'id' },
  { title: '内容', dataIndex: 'content', minWidth: 260, slotName: 'content' },
  { title: '图片', dataIndex: 'images', width: 112, slotName: 'images' },
  { title: '作者', dataIndex: 'nickname', width: 130, ellipsis: true, tooltip: true },
  { title: '置顶', dataIndex: 'isTop', width: 86, slotName: 'top' },
  { title: '状态', dataIndex: 'status', width: 96, slotName: 'status' },
  { title: '创建时间', dataIndex: 'createTime', width: 184, slotName: 'time' },
  { title: '操作', dataIndex: 'actions', width: 150, slotName: 'actions' }
]

const router = useRouter()
const talks = ref<AdminTalk[]>([])
const keywords = ref('')
const statusFilter = ref<'all' | '1' | '2'>('all')
const current = ref(1)
const pageSize = ref(10)
const total = ref(0)
const loading = ref(false)
const pendingTopId = ref(0)
const errorMessage = ref('')

const pagination = computed(() => tablePagination(current.value, pageSize.value, total.value))
const hasFilters = computed(() => Boolean(keywords.value.trim()) || statusFilter.value !== 'all')
/**
 * The talk list endpoint filters by status only, so the keyword box narrows the
 * current page client-side and the caption always reports both numbers.
 */
const visibleTalks = computed(() => {
  const query = keywords.value.trim().toLowerCase()
  const scoped = statusFilter.value === 'all'
    ? talks.value
    : talks.value.filter((talk) => Number(talk.status) === Number(statusFilter.value))
  if (!query) return scoped
  return scoped.filter((talk) => plainText(talk.content).toLowerCase().includes(query))
})

onMounted(() => void load())

async function reload(): Promise<void> {
  current.value = 1
  await load()
}

function resetFilters(): void {
  keywords.value = ''
  statusFilter.value = 'all'
  void reload()
}

async function load(): Promise<void> {
  loading.value = true
  errorMessage.value = ''
  try {
    const page = await listAdminTalks({ current: current.value, size: pageSize.value })
    talks.value = page.items
    total.value = page.total
  } catch (error) {
    errorMessage.value = apiErrorMessage(error, '说说列表加载失败')
    Message.error(errorMessage.value)
  } finally {
    loading.value = false
  }
}

function changePage(page: number): void {
  current.value = page
  void load()
}

function changePageSize(size: number): void {
  pageSize.value = size
  current.value = 1
  void load()
}

/**
 * The talk API has no dedicated "pin" endpoint, so the toggle re-reads the row
 * and writes it back through the regular save endpoint with `isTop` flipped.
 */
async function toggleTop(talk: AdminTalk, value: boolean | string | number): Promise<void> {
  const id = Number(talk.id)
  if (!Number.isInteger(id) || id <= 0) return
  pendingTopId.value = id
  const next = Boolean(value) ? 1 : 0
  try {
    const full = await getAdminTalk(id)
    await saveAdminTalk({
      id,
      content: String(full.content || ''),
      images: String(full.images || JSON.stringify(full.imgs || [])),
      isTop: next,
      status: Number(full.status ?? 1)
    })
    talk.isTop = next
    Message.success(next === 1 ? '说说已置顶' : '已取消置顶')
  } catch (error) {
    Message.error(apiErrorMessage(error, '置顶状态更新失败'))
  } finally {
    pendingTopId.value = 0
  }
}

async function deleteTalk(id: unknown): Promise<void> {
  const talkId = Number(id)
  if (!Number.isInteger(talkId) || talkId <= 0) return
  try {
    await deleteAdminTalks([talkId])
    if (talks.value.length === 1 && current.value > 1) current.value -= 1
    Message.success('说说已删除')
    await load()
  } catch (error) {
    Message.error(apiErrorMessage(error, '说说删除失败'))
  }
}

function talkImages(talk: AdminTalk): string[] {
  if (Array.isArray(talk.imgs)) return talk.imgs.filter(isHttpUrl)
  if (typeof talk.images !== 'string' || !talk.images.trim()) return []
  try {
    const parsed: unknown = JSON.parse(talk.images)
    if (Array.isArray(parsed)) return parsed.map(String).filter(isHttpUrl)
  } catch {
    // Legacy comma-separated image values.
  }
  return talk.images.split(',').map((value) => value.trim()).filter(isHttpUrl)
}
</script>

<style scoped>
.talk-content {
  display: block;
  max-width: 380px;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.talk-images {
  display: flex;
  align-items: center;
  gap: 6px;
}

.talk-image-count {
  color: var(--admin-muted);
  font-size: 12px;
  font-weight: 650;
}
</style>
