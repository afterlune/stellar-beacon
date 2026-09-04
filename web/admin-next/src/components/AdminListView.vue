<template>
  <section>
    <a-card :title="config.title">
      <template #extra>
        <a-input-search
          v-model="keywords"
          :placeholder="config.placeholder"
          allow-clear
          style="width: 260px"
          @search="reload" />
      </template>
      <a-alert v-if="errorMessage" type="error" closable @close="errorMessage = ''">{{ errorMessage }}</a-alert>
      <a-table
        :data="records"
        :columns="tableColumns"
        :loading="loading"
        :pagination="pagination"
        row-key="id"
        @page-change="changePage"
        @page-size-change="changePageSize">
        <template #empty>
          <a-empty description="暂无数据" />
        </template>
        <template #formatted="{ record, column }">
          {{ formatCell(record[column.dataIndex]) }}
        </template>
      </a-table>
    </a-card>
  </section>
</template>

<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { Message } from '@arco-design/web-vue'

import { apiErrorMessage, listAdminPage } from '@/api/http'

type ListMode =
  | 'articles'
  | 'categories'
  | 'tags'
  | 'comments'
  | 'users'
  | 'onlineUsers'
  | 'roles'
  | 'operationLogs'
  | 'exceptionLogs'
  | 'jobLogs'
  | 'jobs'
  | 'albums'
  | 'talks'
interface TableColumn {
  title: string
  dataIndex: string
  slotName?: string
  ellipsis?: boolean
  tooltip?: boolean
}

const props = defineProps<{ mode: ListMode }>()
const keywords = ref('')
const current = ref(1)
const pageSize = ref(10)
const loading = ref(false)
const errorMessage = ref('')
const records = ref<Record<string, unknown>[]>([])
const total = ref(0)

const configs: Record<ListMode, { title: string; placeholder: string; endpoint: string; columns: TableColumn[] }> = {
  articles: {
    title: '文章列表',
    placeholder: '搜索文章标题',
    endpoint: 'admin/articles',
    columns: [
      { title: 'ID', dataIndex: 'id', width: 80 },
      { title: '标题', dataIndex: 'articleTitle', ellipsis: true, tooltip: true },
      { title: '分类', dataIndex: 'categoryName' },
      { title: '状态', dataIndex: 'status' },
      { title: '浏览量', dataIndex: 'viewsCount' },
      { title: '创建时间', dataIndex: 'createTime' }
    ]
  },
  categories: {
    title: '分类管理',
    placeholder: '搜索分类名',
    endpoint: 'admin/categories',
    columns: [
      { title: 'ID', dataIndex: 'id', width: 80 },
      { title: '分类名', dataIndex: 'categoryName' },
      { title: '文章量', dataIndex: 'articleCount' },
      { title: '创建时间', dataIndex: 'createTime' }
    ]
  },
  tags: {
    title: '标签管理',
    placeholder: '搜索标签名',
    endpoint: 'admin/tags',
    columns: [
      { title: 'ID', dataIndex: 'id', width: 80 },
      { title: '标签名', dataIndex: 'tagName' },
      { title: '文章量', dataIndex: 'articleCount' },
      { title: '创建时间', dataIndex: 'createTime' }
    ]
  },
  comments: {
    title: '评论管理',
    placeholder: '搜索评论内容',
    endpoint: 'admin/comments',
    columns: [
      { title: 'ID', dataIndex: 'id', width: 80 },
      { title: '用户', dataIndex: 'nickname' },
      { title: '文章', dataIndex: 'articleTitle' },
      { title: '内容', dataIndex: 'commentContent', ellipsis: true, tooltip: true },
      { title: '审核状态', dataIndex: 'isReview' },
      { title: '创建时间', dataIndex: 'createTime' }
    ]
  },
  users: {
    title: '用户管理',
    placeholder: '搜索用户昵称',
    endpoint: 'admin/users',
    columns: [
      { title: 'ID', dataIndex: 'id', width: 80 },
      { title: '昵称', dataIndex: 'nickname' },
      { title: '登录类型', dataIndex: 'loginType' },
      { title: 'IP', dataIndex: 'ipAddress' },
      { title: '状态', dataIndex: 'isDisable' },
      { title: '最后登录', dataIndex: 'lastLoginTime' }
    ]
  },
  onlineUsers: {
    title: '在线用户',
    placeholder: '搜索用户昵称',
    endpoint: 'admin/users/online',
    columns: [
      { title: '用户 ID', dataIndex: 'userInfoId', width: 100 },
      { title: '昵称', dataIndex: 'nickname' },
      { title: '浏览器', dataIndex: 'browser' },
      { title: '操作系统', dataIndex: 'os' },
      { title: 'IP', dataIndex: 'ipAddress' },
      { title: '最后登录', dataIndex: 'lastLoginTime' }
    ]
  },
  roles: {
    title: '角色管理',
    placeholder: '搜索角色名',
    endpoint: 'admin/roles',
    columns: [
      { title: 'ID', dataIndex: 'id', width: 80 },
      { title: '角色名', dataIndex: 'roleName' },
      { title: '状态', dataIndex: 'isDisable' },
      { title: '创建时间', dataIndex: 'createTime' }
    ]
  },
  operationLogs: {
    title: '操作日志',
    placeholder: '搜索操作模块',
    endpoint: 'admin/operation/logs',
    columns: [
      { title: 'ID', dataIndex: 'id', width: 80 },
      { title: '模块', dataIndex: 'optModule' },
      { title: '类型', dataIndex: 'optType' },
      { title: 'URI', dataIndex: 'optUri' },
      { title: '方法', dataIndex: 'optMethod' },
      { title: '用户', dataIndex: 'nickname' },
      { title: '创建时间', dataIndex: 'createTime' }
    ]
  },
  exceptionLogs: {
    title: '异常日志',
    placeholder: '搜索请求 URI',
    endpoint: 'admin/exception/logs',
    columns: [
      { title: 'ID', dataIndex: 'id', width: 80 },
      { title: 'URI', dataIndex: 'optUri' },
      { title: '方法', dataIndex: 'optMethod' },
      { title: '描述', dataIndex: 'optDesc' },
      { title: '异常', dataIndex: 'exceptionInfo', ellipsis: true, tooltip: true },
      { title: '创建时间', dataIndex: 'createTime' }
    ]
  },
  jobLogs: {
    title: '任务日志',
    placeholder: '搜索任务名',
    endpoint: 'admin/jobLogs',
    columns: [
      { title: 'ID', dataIndex: 'id', width: 80 },
      { title: '任务', dataIndex: 'jobName' },
      { title: '任务组', dataIndex: 'jobGroup' },
      { title: '状态', dataIndex: 'status' },
      { title: '耗时', dataIndex: 'time' },
      { title: '创建时间', dataIndex: 'createTime' }
    ]
  },
  jobs: {
    title: '定时任务',
    placeholder: '搜索任务名',
    endpoint: 'admin/jobs',
    columns: [
      { title: 'ID', dataIndex: 'id', width: 80 },
      { title: '任务', dataIndex: 'jobName' },
      { title: '任务组', dataIndex: 'jobGroup' },
      { title: 'Cron', dataIndex: 'cronExpression' },
      { title: '状态', dataIndex: 'status' },
      { title: '创建时间', dataIndex: 'createTime' }
    ]
  },
  albums: {
    title: '相册管理',
    placeholder: '搜索相册名',
    endpoint: 'admin/photos/albums',
    columns: [
      { title: 'ID', dataIndex: 'id', width: 80 },
      { title: '相册名', dataIndex: 'albumName' },
      { title: '照片数', dataIndex: 'photoCount' },
      { title: '状态', dataIndex: 'status' }
    ]
  },
  talks: {
    title: '说说管理',
    placeholder: '搜索说说内容',
    endpoint: 'admin/talks',
    columns: [
      { title: 'ID', dataIndex: 'id', width: 80 },
      { title: '内容', dataIndex: 'content', ellipsis: true, tooltip: true },
      { title: '评论数', dataIndex: 'commentCount' },
      { title: '状态', dataIndex: 'status' },
      { title: '创建时间', dataIndex: 'createTime' }
    ]
  }
}

const config = computed(() => configs[props.mode])
const tableColumns = computed(() => config.value.columns.map((column) => ({ ...column, slotName: 'formatted' })))
const pagination = computed(() => ({ current: current.value, pageSize: pageSize.value, total: total.value, showTotal: true, showJumper: true, showPageSize: true }))

onMounted(() => void reload())

async function reload(): Promise<void> {
  current.value = 1
  await load()
}

async function load(): Promise<void> {
  loading.value = true
  errorMessage.value = ''
  try {
    const page = await listAdminPage<Record<string, unknown>>(config.value.endpoint, {
      current: current.value,
      size: pageSize.value,
      keywords: keywords.value.trim()
    })
    records.value = page.records
    total.value = page.count
  } catch (error) {
    errorMessage.value = apiErrorMessage(error, '列表加载失败')
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

function formatCell(value: unknown): string {
  if (value === null || value === undefined || value === '') return '—'
  if (typeof value === 'string' && value.includes('T')) return value.replace('T', ' ').replace(/\.\d+Z$/, '')
  return String(value)
}
</script>
