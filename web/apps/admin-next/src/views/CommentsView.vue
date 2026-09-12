<template>
  <section class="admin-page">
    <AdminPageHeader title="评论管理" description="保持交流友好，也让每一条反馈都及时得到回应。">
      <template #actions>
        <a-input-search
          v-model="keywords"
          class="admin-filter-input"
          placeholder="搜索评论内容"
          allow-clear
          @search="reload" />
        <a-button :loading="loading" @click="load">
          <template #icon><IconRefresh /></template>
          刷新
        </a-button>
      </template>
    </AdminPageHeader>

    <a-card class="admin-panel" :bordered="false">
      <div class="admin-table-toolbar">
        <div class="admin-table-toolbar-main">
          <a-radio-group v-model="reviewFilter" type="button" size="small" @change="reload">
            <a-radio :value="0">全部</a-radio>
            <a-radio :value="2">待审核</a-radio>
            <a-radio :value="1">已通过</a-radio>
          </a-radio-group>
          <span class="admin-toolbar-caption">共 {{ total }} 条评论</span>
        </div>
        <div class="admin-table-toolbar-actions">
          <a-button
            v-if="pendingCount > 0"
            size="small"
            type="primary"
            :loading="approvingAll"
            @click="approveAllPending">
            一键通过本页 {{ pendingCount }} 条待审核
          </a-button>
        </div>
      </div>

      <a-alert v-if="errorMessage" type="error" closable @close="errorMessage = ''">{{ errorMessage }}</a-alert>

      <div class="admin-table-shell">
        <a-table
          :data="records"
          :columns="columns"
          :loading="loading"
          :pagination="pagination"
          row-key="id"
          @page-change="changePage"
          @page-size-change="changePageSize">
          <template #id="{ record }"><span class="admin-id-cell">#{{ record.id }}</span></template>
          <template #article="{ record }">
            <span v-if="record.articleTitle" :title="String(record.articleTitle)" class="comment-article">
              {{ record.articleTitle }}
            </span>
            <span v-else class="admin-muted-cell">未关联文章</span>
          </template>
          <template #content="{ record }">
            <span class="comment-content" :title="plainText(record.commentContent)">{{ plainText(record.commentContent) || '—' }}</span>
          </template>
          <template #review="{ record }">
            <AdminStatusTag :kind="Number(record.isReview) === 1 ? 'reviewed' : 'pending'" />
          </template>
          <template #time="{ record }"><span class="admin-cell-nowrap">{{ formatDateTime(record.createTime) }}</span></template>
          <template #actions="{ record }">
            <a-space class="admin-action-space">
              <a-button type="text" size="small" :loading="busyId === record.id" @click="toggleReview(record)">
                {{ Number(record.isReview) === 1 ? '取消审核' : '通过审核' }}
              </a-button>
              <a-popconfirm content="确认删除这条评论吗？删除后无法恢复。" @ok="remove(record.id)">
                <a-button type="text" status="danger" size="small">删除</a-button>
              </a-popconfirm>
            </a-space>
          </template>
          <template #empty>
            <AdminEmptyState
              :icon="IconMessage"
              :title="keywords.trim() ? '没有匹配的评论' : '暂无评论'"
              :description="keywords.trim() ? '换个关键词再试一次。' : '当读者留下第一条评论时，它会出现在这里。'">
              <a-button v-if="keywords.trim()" size="small" @click="clearKeywords">清空搜索</a-button>
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
import { IconMessage, IconRefresh } from '@arco-design/web-vue/es/icon'

import { apiErrorMessage, deleteComment, listAdminPage, reviewComment } from '@/api/http'
import AdminEmptyState from '@/components/AdminEmptyState.vue'
import AdminPageHeader from '@/components/AdminPageHeader.vue'
import AdminStatusTag from '@/components/AdminStatusTag.vue'
import { formatDateTime, plainText } from '@/utils/format'
import { tablePagination } from '@/utils/pagination'

interface CommentRow extends Record<string, unknown> {
  id: number
  isReview?: number
  commentContent?: string
  createTime?: string
}

const columns = [
  { title: 'ID', dataIndex: 'id', width: 78, slotName: 'id' },
  { title: '用户', dataIndex: 'nickname', width: 140, ellipsis: true, tooltip: true },
  { title: '文章', dataIndex: 'articleTitle', width: 200, slotName: 'article' },
  { title: '内容', dataIndex: 'commentContent', minWidth: 260, slotName: 'content' },
  { title: '审核', dataIndex: 'isReview', width: 104, slotName: 'review' },
  { title: '创建时间', dataIndex: 'createTime', width: 184, slotName: 'time' },
  { title: '操作', dataIndex: 'actions', width: 168, slotName: 'actions' }
]

const keywords = ref('')
const reviewFilter = ref(0)
const current = ref(1)
const size = ref(10)
const total = ref(0)
const records = ref<CommentRow[]>([])
const loading = ref(false)
const approvingAll = ref(false)
const busyId = ref(0)
const errorMessage = ref('')

const pagination = computed(() => tablePagination(current.value, size.value, total.value))
const pendingCount = computed(() => records.value.filter((row) => Number(row.isReview) !== 1).length)

onMounted(() => void reload())

async function reload(): Promise<void> {
  current.value = 1
  await load()
}

function clearKeywords(): void {
  keywords.value = ''
  void reload()
}

async function load(): Promise<void> {
  loading.value = true
  errorMessage.value = ''
  try {
    const page = await listAdminPage<CommentRow>('admin/comments', {
      current: current.value,
      size: size.value,
      keywords: keywords.value.trim(),
      isReview: reviewFilter.value
    })
    records.value = page.items
    total.value = page.total
  } catch (error) {
    errorMessage.value = apiErrorMessage(error, '评论加载失败')
    Message.error(errorMessage.value)
  } finally {
    loading.value = false
  }
}

async function toggleReview(row: CommentRow): Promise<void> {
  const next = Number(row.isReview) === 1 ? 0 : 1
  busyId.value = Number(row.id)
  try {
    await reviewComment(Number(row.id), next)
    row.isReview = next
    Message.success(next === 1 ? '评论已通过审核' : '已取消审核')
    await load()
  } catch (error) {
    Message.error(apiErrorMessage(error, '审核操作失败'))
  } finally {
    busyId.value = 0
  }
}

async function approveAllPending(): Promise<void> {
  const pending = records.value.filter((row) => Number(row.isReview) !== 1)
  if (pending.length === 0) return
  approvingAll.value = true
  try {
    for (const row of pending) await reviewComment(Number(row.id), 1)
    Message.success(`已通过本页 ${pending.length} 条评论`)
    await load()
  } catch (error) {
    Message.error(apiErrorMessage(error, '批量审核失败'))
    await load()
  } finally {
    approvingAll.value = false
  }
}

async function remove(id: unknown): Promise<void> {
  const commentId = Number(id)
  if (!Number.isInteger(commentId) || commentId <= 0) return
  try {
    await deleteComment(commentId)
    if (records.value.length === 1 && current.value > 1) current.value -= 1
    await load()
    Message.success('评论已删除')
  } catch (error) {
    Message.error(apiErrorMessage(error, '删除失败'))
  }
}

function changePage(page: number): void {
  current.value = page
  void load()
}

function changePageSize(pageSize: number): void {
  size.value = pageSize
  current.value = 1
  void load()
}
</script>

<style scoped>
.comment-content {
  display: inline-block;
  max-width: 340px;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  vertical-align: bottom;
}

.comment-article {
  display: inline-block;
  max-width: 100%;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  vertical-align: bottom;
}
</style>
