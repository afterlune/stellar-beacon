<template>
  <section class="admin-page">
    <AdminPageHeader title="在线用户" description="查看当前登录会话及其状态。">
      <template #actions>
        <a-input-search
          v-model="keywords"
          class="admin-filter-input"
          placeholder="搜索用户昵称"
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
          <span class="admin-live-badge"><span class="admin-live-dot" aria-hidden="true" />当前在线 {{ total }} 人</span>
          <a-tag v-if="keywords.trim()" color="arcoblue">关键词：{{ keywords.trim() }}</a-tag>
          <a-button v-if="keywords.trim()" type="text" size="small" @click="clearKeywords">清空搜索</a-button>
        </div>
      </div>

      <AdminErrorState v-if="errorMessage" :error="errorMessage" title="在线用户加载失败" @retry="load" />

      <AdminBatchBar :count="selectedIds.length" :hint="`本页 ${records.length} 人`" @clear="clearSelection">
        <a-button :disabled="selectedIds.length === 0" size="small" status="danger" @click="removeSelected">
          强制下线选中（{{ selectedIds.length }}）
        </a-button>
      </AdminBatchBar>

      <div class="admin-table-shell">
        <a-table
          v-model:selected-keys="selectedKeys"
          :row-selection="{ type: 'checkbox', showCheckedAll: true, onlyCurrent: true }"
          :data="records"
          :columns="columns"
          :loading="loading"
          :pagination="pagination"
          row-key="userInfoId"
          @page-change="changePage"
          @page-size-change="changePageSize">
          <template #user="{ record }">
            <div class="online-user-cell">
              <a-avatar :size="30" :image-url="record.avatar">{{ initialOf(record.nickname || record.username) }}</a-avatar>
              <span class="online-user-copy">
                <strong :title="String(record.nickname || '')">{{ record.nickname || record.username || '未命名用户' }}</strong>
                <small>ID {{ record.userInfoId ?? '—' }}</small>
              </span>
            </div>
          </template>
          <template #client="{ record }">
            <span class="admin-muted-cell" :title="`${record.browser || ''} / ${record.os || ''}`">
              {{ clientLabel(record) }}
            </span>
          </template>
          <template #ip="{ record }">
            <a-tooltip v-if="record.ipSource" :content="String(record.ipSource)">
              <span class="admin-mono-cell">{{ record.ipAddress || '—' }}</span>
            </a-tooltip>
            <span v-else class="admin-mono-cell">{{ record.ipAddress || '—' }}</span>
          </template>
          <template #time="{ record }"><span class="admin-cell-nowrap">{{ formatDateTime(record.lastLoginTime) }}</span></template>
          <template #actions="{ record }">
            <a-popconfirm
              :content="`强制「${record.nickname || record.username || '该用户'}」下线？对方需要重新登录。`"
              @ok="removeOne(record)">
              <a-button type="text" status="danger" size="small" :loading="pendingId === Number(record.userInfoId)">
                强制下线
              </a-button>
            </a-popconfirm>
          </template>
          <template #empty>
            <AdminEmptyState
              :icon="IconUser"
              :title="keywords.trim() ? '没有匹配的会话' : '当前没有活跃会话'"
              :description="keywords.trim() ? '换个关键词再试一次。' : '当有用户登录并保持会话时，会出现在这里。'">
              <a-button v-if="keywords.trim()" size="small" @click="clearKeywords">清空搜索</a-button>
            </AdminEmptyState>
          </template>
        </a-table>
      </div>
    </a-card>
  </section>
</template>

<script setup lang="ts">
import { computed, ref } from 'vue'
import { Message } from '@arco-design/web-vue'
import { IconRefresh, IconUser } from '@arco-design/web-vue/es/icon'

import { apiErrorMessage, listAdminOnlineUsers, removeAdminOnlineUser } from '@/api/http'
import AdminBatchBar from '@/components/AdminBatchBar.vue'
import AdminEmptyState from '@/components/AdminEmptyState.vue'
import AdminErrorState from '@/components/AdminErrorState.vue'
import AdminPageHeader from '@/components/AdminPageHeader.vue'
import { useAsyncList } from '@/composables/useAsyncList'
import { useQueryFilters } from '@/composables/useQueryFilters'
import { readStoredPageSize, useStoredPageSize } from '@/composables/useTablePrefs'
import { formatDateTime, initialOf } from '@/utils/format'
import { tablePagination } from '@/utils/pagination'
import type { AdminUser } from '@stellar-beacon/api-contract'

const VIEW_KEY = 'online-users'

const columns = [
  { title: '用户', dataIndex: 'nickname', slotName: 'user', minWidth: 190 },
  { title: '浏览器 / 系统', dataIndex: 'browser', slotName: 'client', minWidth: 170 },
  { title: 'IP 地址', dataIndex: 'ipAddress', slotName: 'ip', width: 160 },
  { title: '最后活跃', dataIndex: 'lastLoginTime', slotName: 'time', width: 168 },
  { title: '操作', dataIndex: 'actions', slotName: 'actions', width: 120 }
]

const selectedKeys = ref<Array<string | number>>([])
const keywords = ref('')
const pendingId = ref(0)

const {
  items: records,
  total,
  current,
  pageSize,
  loading,
  error: errorMessage,
  load,
  reload,
  changePage: gotoPage,
  changePageSize: applyPageSize
} = useAsyncList<AdminUser>(
  async ({ current: page, pageSize: size, signal }) => {
    const result = await listAdminOnlineUsers({
      current: page,
      size,
      keywords: keywords.value.trim()
    }, { signal })
    // 重新加载后丢掉已不在当前页的勾选，避免批量下线误伤已下线的会话。
    const available = new Set(result.items.map((record) => Number(record.userInfoId ?? record.id)))
    selectedKeys.value = selectedKeys.value.filter((key) => available.has(Number(key)))
    return result
  },
  { pageSize: readStoredPageSize(VIEW_KEY), fallbackMessage: '在线用户加载失败' }
)

useStoredPageSize(VIEW_KEY, pageSize)
useQueryFilters([
  { key: 'keywords', ref: keywords, debounce: true },
  { key: 'page', ref: current }
], { onRestore: () => void load(), onSearch: () => void reload() })

const pagination = computed(() => tablePagination(current.value, pageSize.value, total.value))
const selectedIds = computed(() =>
  [...new Set(selectedKeys.value.map(Number).filter((id) => Number.isInteger(id) && id > 0))]
)

function clearKeywords(): void {
  keywords.value = ''
  void reload()
}

function changePage(page: number): void {
  gotoPage(page)
}

function changePageSize(size: number): void {
  applyPageSize(size)
}

function clearSelection(): void {
  selectedKeys.value = []
}

async function removeOne(record: AdminUser): Promise<void> {
  const id = Number(record.userInfoId ?? record.id)
  if (!Number.isInteger(id) || id <= 0) return
  pendingId.value = id
  try {
    await removeAdminOnlineUser(id)
    Message.success('该会话已强制下线')
    await load()
  } catch (error) {
    Message.error(apiErrorMessage(error, '强制下线失败'))
  } finally {
    pendingId.value = 0
  }
}

async function removeSelected(): Promise<void> {
  const ids = selectedIds.value
  if (!ids.length) return
  try {
    for (const id of ids) await removeAdminOnlineUser(id)
    selectedKeys.value = []
    Message.success(`已强制 ${ids.length} 个会话下线`)
    await load()
  } catch (error) {
    Message.error(apiErrorMessage(error, '批量强制下线失败'))
    await load()
  }
}

function clientLabel(record: AdminUser): string {
  const browser = String(record.browser || '').trim()
  const os = String(record.os || '').trim()
  if (browser && os) return `${browser} · ${os}`
  return browser || os || '未知客户端'
}
</script>

<style scoped>
.online-user-cell {
  display: flex;
  align-items: center;
  gap: 10px;
  min-width: 0;
}

.online-user-copy {
  min-width: 0;
  display: grid;
  gap: 1px;
}

.online-user-copy strong,
.online-user-copy small {
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.online-user-copy strong {
  color: var(--admin-ink-strong);
  font-size: 13px;
  font-weight: 650;
}

.online-user-copy small {
  color: var(--admin-subtle);
  font-size: 11px;
}
</style>
