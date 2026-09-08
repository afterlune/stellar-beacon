<template>
  <section class="admin-page">
    <AdminPageHeader title="评论管理" description="保持交流友好，也让每一条反馈都有回应。">
      <template #actions>
        <a-input-search v-model="keywords" class="admin-filter-input" placeholder="搜索评论内容" allow-clear @search="reload" />
      </template>
    </AdminPageHeader>
    <a-card class="admin-panel" :bordered="false">
      <a-alert v-if="errorMessage" type="error" closable @close="errorMessage = ''">{{ errorMessage }}</a-alert>
      <div class="admin-table-shell">
        <a-table :data="records" :columns="columns" :loading="loading" :pagination="pagination" row-key="id" @page-change="changePage">
          <template #content="{ record }"><span class="comment-content">{{ record.commentContent || '—' }}</span></template>
          <template #review="{ record }"><a-tag class="admin-status-tag" :color="Number(record.isReview) === 1 ? 'green' : 'orange'">{{ Number(record.isReview) === 1 ? '已审核' : '待审核' }}</a-tag></template>
          <template #time="{ record }">{{ formatTime(record.createTime) }}</template>
          <template #actions="{ record }">
            <a-space class="admin-action-space">
              <a-button type="text" size="small" @click="toggleReview(record)">{{ Number(record.isReview) === 1 ? '取消审核' : '通过审核' }}</a-button>
              <a-popconfirm content="确认删除这条评论吗？" @ok="remove(record.id)"><a-button type="text" status="danger" size="small">删除</a-button></a-popconfirm>
            </a-space>
          </template>
          <template #empty><div class="admin-table-empty"><a-empty description="暂无评论" /></div></template>
        </a-table>
      </div>
    </a-card>
  </section>
</template>

<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { Message } from '@arco-design/web-vue'

import { apiErrorMessage, deleteComment, listAdminPage, reviewComment } from '@/api/http'
import AdminPageHeader from '@/components/AdminPageHeader.vue'
import { formatTime } from '@/utils/format'
import { tablePagination } from '@/utils/pagination'

interface CommentRow extends Record<string, unknown> { id: number; isReview?: number; commentContent?: string; createTime?: string }
const columns = [
  { title: 'ID', dataIndex: 'id', width: 80 },
  { title: '用户', dataIndex: 'nickname' },
  { title: '文章', dataIndex: 'articleTitle' },
  { title: '内容', dataIndex: 'commentContent', slotName: 'content', ellipsis: true, tooltip: true },
  { title: '审核', dataIndex: 'isReview', slotName: 'review' },
  { title: '创建时间', dataIndex: 'createTime', slotName: 'time' },
  { title: '操作', slotName: 'actions', width: 180 }
]
const keywords = ref('')
const current = ref(1)
const size = ref(10)
const total = ref(0)
const records = ref<CommentRow[]>([])
const loading = ref(false)
const errorMessage = ref('')
const pagination = computed(() => tablePagination(current.value, size.value, total.value))

onMounted(() => void reload())
async function reload(): Promise<void> { current.value = 1; await load() }
async function load(): Promise<void> {
  loading.value = true
  try {
    const page = await listAdminPage<CommentRow>('admin/comments', { current: current.value, size: size.value, keywords: keywords.value.trim() })
    records.value = page.records
    total.value = page.count
  } catch (error) {
    errorMessage.value = apiErrorMessage(error, '评论加载失败')
    Message.error(errorMessage.value)
  } finally { loading.value = false }
}
async function toggleReview(row: CommentRow): Promise<void> {
  try { await reviewComment(row.id, Number(row.isReview) === 1 ? 0 : 1); await load(); Message.success('审核状态已更新') }
  catch (error) { Message.error(apiErrorMessage(error, '审核操作失败')) }
}
async function remove(id: number): Promise<void> {
  try { await deleteComment(id); await load(); Message.success('删除成功') }
  catch (error) { Message.error(apiErrorMessage(error, '删除失败')) }
}
function changePage(page: number): void { current.value = page; void load() }
</script>

<style scoped>
.comment-content { display: inline-block; max-width: 320px; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; vertical-align: bottom; }
</style>
