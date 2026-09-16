<template>
  <section class="admin-page">
    <AdminPageHeader :title="t('taxonomy.online.title')" :description="t('taxonomy.online.description')">
      <template #actions>
        <a-input-search
          v-model="keywords"
          class="admin-filter-input"
          :placeholder="t('taxonomy.online.searchPlaceholder')"
          allow-clear
          @search="reload" />
        <a-button :loading="loading" @click="load">
          <template #icon><IconRefresh /></template>
          {{ t('taxonomy.common.refresh') }}
        </a-button>
      </template>
    </AdminPageHeader>

    <a-card class="admin-panel" :bordered="false">
      <div class="admin-table-toolbar">
        <div class="admin-table-toolbar-main">
          <span class="admin-live-badge"><span class="admin-live-dot" aria-hidden="true" />{{ t('taxonomy.online.total', { total }) }}</span>
          <a-tag v-if="keywords.trim()" color="arcoblue">{{ t('taxonomy.online.keywords', { keywords: keywords.trim() }) }}</a-tag>
          <a-button v-if="keywords.trim()" type="text" size="small" @click="clearKeywords">{{ t('taxonomy.online.clearKeywords') }}</a-button>
        </div>
      </div>

      <AdminErrorState v-if="errorMessage" :error="errorMessage" :title="t('taxonomy.online.loadFailed')" @retry="load" />

      <AdminBatchBar :count="selectedIds.length" :hint="t('taxonomy.online.pageCount', { count: records.length })" @clear="clearSelection">
        <a-button :disabled="selectedIds.length === 0" size="small" status="danger" @click="removeSelected">
          {{ t('taxonomy.online.forceOfflineSelected', { count: selectedIds.length }) }}
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
                <strong :title="String(record.nickname || '')">{{ record.nickname || record.username || t('taxonomy.online.unnamed') }}</strong>
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
              :content="t('taxonomy.online.forceConfirm', { name: record.nickname || record.username || t('taxonomy.online.forceTarget') })"
              @ok="removeOne(record)">
              <a-button type="text" status="danger" size="small" :loading="pendingId === Number(record.userInfoId)">
                {{ t('taxonomy.online.forceOffline') }}
              </a-button>
            </a-popconfirm>
          </template>
          <template #empty>
            <AdminEmptyState
              :icon="IconUser"
              :title="keywords.trim() ? t('taxonomy.online.emptyFilteredTitle') : t('taxonomy.online.emptyTitle')"
              :description="keywords.trim() ? t('taxonomy.online.emptyFilteredHint') : t('taxonomy.online.emptyHint')">
              <a-button v-if="keywords.trim()" size="small" @click="clearKeywords">{{ t('taxonomy.online.clearKeywords') }}</a-button>
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
import { t } from '@/i18n'
import { formatDateTime, initialOf } from '@/utils/format'
import { tablePagination } from '@/utils/pagination'
import type { AdminUser } from '@stellar-beacon/api-contract'

const VIEW_KEY = 'online-users'

// 列定义里的标题要跟着语言切换更新，因此用 computed 而不是模块级常量。
const columns = computed(() => [
  { title: t('taxonomy.online.user'), dataIndex: 'nickname', slotName: 'user', minWidth: 190 },
  { title: t('taxonomy.online.client'), dataIndex: 'browser', slotName: 'client', minWidth: 170 },
  { title: t('taxonomy.online.ip'), dataIndex: 'ipAddress', slotName: 'ip', width: 160 },
  { title: t('taxonomy.online.lastActive'), dataIndex: 'lastLoginTime', slotName: 'time', width: 168 },
  { title: t('taxonomy.online.actions'), dataIndex: 'actions', slotName: 'actions', width: 120 }
])

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
  // useAsyncList 只在初始化时读一次 options，兜底文案取当前语言即可。
  { pageSize: readStoredPageSize(VIEW_KEY), fallbackMessage: t('taxonomy.online.loadFailed') }
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
    Message.success(t('taxonomy.online.forceSuccess'))
    await load()
  } catch (error) {
    Message.error(apiErrorMessage(error, t('taxonomy.online.forceFailed')))
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
    Message.success(t('taxonomy.online.batchForceSuccess', { count: ids.length }))
    await load()
  } catch (error) {
    Message.error(apiErrorMessage(error, t('taxonomy.online.batchForceFailed')))
    await load()
  }
}

function clientLabel(record: AdminUser): string {
  const browser = String(record.browser || '').trim()
  const os = String(record.os || '').trim()
  if (browser && os) return `${browser} · ${os}`
  return browser || os || t('taxonomy.online.unknownClient')
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
