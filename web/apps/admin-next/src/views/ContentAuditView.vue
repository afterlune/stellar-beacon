<template>
  <section class="admin-page">
    <AdminPageHeader :title="t('logs.contentAudit.title')" :description="t('logs.contentAudit.description')">
      <template #actions>
        <a-button :loading="loading" @click="load">
          <template #icon><IconRefresh /></template>
          {{ t('common.refresh') }}
        </a-button>
      </template>
    </AdminPageHeader>

    <a-card class="admin-panel" :bordered="false">
      <div class="admin-table-toolbar">
        <div class="admin-table-toolbar-main">
          <a-select v-model="filters.contentType" allow-clear :placeholder="t('logs.contentAudit.contentType')" style="width: 130px" @change="reload">
            <a-option value="article">{{ t('logs.contentAudit.article') }}</a-option>
            <a-option value="talk">{{ t('logs.contentAudit.talk') }}</a-option>
            <a-option value="series">{{ t('logs.contentAudit.series') }}</a-option>
          </a-select>
          <a-select v-model="filters.operation" allow-clear :placeholder="t('logs.contentAudit.operation')" style="width: 150px" @change="reload">
            <a-option value="batch_status">{{ t('logs.contentAudit.batchStatus') }}</a-option>
            <a-option value="batch_delete">{{ t('logs.contentAudit.batchDelete') }}</a-option>
            <a-option value="publish_retry">{{ t('logs.contentAudit.publishRetry') }}</a-option>
          </a-select>
          <a-select v-model="filters.result" allow-clear :placeholder="t('logs.contentAudit.result')" style="width: 120px" @change="reload">
            <a-option value="success">{{ t('logs.contentAudit.success') }}</a-option>
            <a-option value="failed">{{ t('logs.contentAudit.failed') }}</a-option>
          </a-select>
          <input v-model="filters.startDate" class="content-audit-date" type="date" :aria-label="t('logs.contentAudit.startDate')" @change="reload" />
          <input v-model="filters.endDate" class="content-audit-date" type="date" :aria-label="t('logs.contentAudit.endDate')" @change="reload" />
          <a-input-search v-model="filters.keywords" :placeholder="t('logs.contentAudit.search')" allow-clear style="width: 220px" @search="reload" />
        </div>
        <span class="admin-toolbar-caption">{{ t('pagination.total', { total }) }}</span>
      </div>

      <AdminErrorState v-if="error" :error="error" :title="t('logs.contentAudit.loadFailed')" @retry="load" />
      <div class="admin-table-shell">
        <a-table :data="records" :columns="columns" :loading="loading" :pagination="pagination" row-key="id" @page-change="changePage" @page-size-change="changePageSize">
          <template #operator="{ record }">
            <strong>{{ record.operatorNickname || `#${record.operatorId}` }}</strong>
          </template>
          <template #kind="{ record }">{{ kindLabel(record.contentType) }}</template>
          <template #operation="{ record }">{{ operationLabel(record.operation) }}</template>
          <template #scope="{ record }">{{ record.targetMode === 'filter' ? t('logs.contentAudit.filterScope') : t('logs.contentAudit.idScope') }}</template>
          <template #result="{ record }">
            <a-tag :color="record.result === 'success' ? 'green' : 'red'">{{ record.result === 'success' ? t('logs.contentAudit.success') : t('logs.contentAudit.failed') }}</a-tag>
          </template>
          <template #time="{ record }">{{ formatDateTime(record.createTime) }}</template>
          <template #actions="{ record }">
            <a-button type="text" size="small" @click="openDetail(record)">{{ t('common.detail') }}</a-button>
          </template>
          <template #empty>
            <AdminEmptyState :icon="IconHistory" :title="t('logs.contentAudit.empty')" :description="t('logs.contentAudit.emptyHint')" />
          </template>
        </a-table>
      </div>
    </a-card>

    <a-drawer v-model:visible="detailVisible" :title="t('logs.contentAudit.detailTitle')" width="820px" :footer="false">
      <template v-if="selected">
        <a-descriptions :column="2" size="small" bordered>
          <a-descriptions-item :label="t('logs.contentAudit.operator')">{{ selected.operatorNickname || `#${selected.operatorId}` }}</a-descriptions-item>
          <a-descriptions-item :label="t('logs.contentAudit.time')">{{ formatDateTime(selected.createTime) }}</a-descriptions-item>
          <a-descriptions-item :label="t('logs.contentAudit.contentType')">{{ kindLabel(selected.contentType) }}</a-descriptions-item>
          <a-descriptions-item :label="t('logs.contentAudit.operation')">{{ operationLabel(selected.operation) }}</a-descriptions-item>
          <a-descriptions-item :label="t('logs.contentAudit.requested')">{{ selected.requestedCount }}</a-descriptions-item>
          <a-descriptions-item :label="t('logs.contentAudit.affected')">{{ selected.affectedCount }}</a-descriptions-item>
          <a-descriptions-item :label="t('logs.contentAudit.ip')">{{ selected.ipAddress || '-' }}</a-descriptions-item>
          <a-descriptions-item :label="t('logs.contentAudit.result')">{{ selected.result === 'success' ? t('logs.contentAudit.success') : t('logs.contentAudit.failed') }}</a-descriptions-item>
        </a-descriptions>
        <h4>{{ t('logs.contentAudit.filterSnapshot') }}</h4>
        <pre class="content-audit-json">{{ prettyJSON(selected.filterSnapshot) }}</pre>
        <p v-if="selected.errorMessage" class="content-audit-error">{{ selected.errorMessage }}</p>
        <h4>{{ t('logs.contentAudit.items') }}</h4>
        <a-table :data="items" :columns="itemColumns" :loading="itemsLoading" :pagination="itemPagination" row-key="id" size="small" @page-change="changeItemPage" @page-size-change="changeItemPageSize">
          <template #title="{ record }">{{ record.title || `#${record.contentId}` }}</template>
          <template #previousStatus="{ record }">{{ statusLabel(record.previousStatus) }}</template>
          <template #nextStatus="{ record }">{{ statusLabel(record.nextStatus) }}</template>
        </a-table>
      </template>
    </a-drawer>
  </section>
</template>

<script setup lang="ts">
import { computed, onMounted, reactive, ref } from 'vue'
import { IconHistory, IconRefresh } from '@arco-design/web-vue/es/icon'
import type { ContentAuditItem, ContentAuditRecord } from '@stellar-beacon/api-contract'
import { apiErrorMessage, listAdminContentAuditItems, listAdminContentAudits } from '@/api/http'
import AdminEmptyState from '@/components/AdminEmptyState.vue'
import AdminErrorState from '@/components/AdminErrorState.vue'
import AdminPageHeader from '@/components/AdminPageHeader.vue'
import { t } from '@/i18n'
import { formatDateTime } from '@/utils/format'

const records = ref<ContentAuditRecord[]>([])
const items = ref<ContentAuditItem[]>([])
const selected = ref<ContentAuditRecord | null>(null)
const loading = ref(false)
const itemsLoading = ref(false)
const error = ref('')
const detailVisible = ref(false)
const current = ref(1)
const pageSize = ref(20)
const total = ref(0)
const itemCurrent = ref(1)
const itemPageSize = ref(20)
const itemTotal = ref(0)
const filters = reactive({ contentType: '', operation: '', result: '', keywords: '', startDate: '', endDate: '' })
const columns = computed(() => [
  { title: t('logs.contentAudit.time'), dataIndex: 'createTime', slotName: 'time', width: 180 },
  { title: t('logs.contentAudit.operator'), dataIndex: 'operatorNickname', slotName: 'operator', width: 130 },
  { title: t('logs.contentAudit.contentType'), dataIndex: 'contentType', slotName: 'kind', width: 90 },
  { title: t('logs.contentAudit.operation'), dataIndex: 'operation', slotName: 'operation', width: 130 },
  { title: t('logs.contentAudit.scope'), dataIndex: 'targetMode', slotName: 'scope', width: 110 },
  { title: t('logs.contentAudit.affected'), dataIndex: 'affectedCount', width: 90 },
  { title: t('logs.contentAudit.result'), dataIndex: 'result', slotName: 'result', width: 90 },
  { title: t('common.actions'), dataIndex: 'actions', slotName: 'actions', width: 80 }
])
const itemColumns = computed(() => [
  { title: t('logs.contentAudit.itemTitle'), dataIndex: 'title', slotName: 'title', width: 280 },
  { title: t('logs.contentAudit.previousStatus'), dataIndex: 'previousStatus', slotName: 'previousStatus', width: 110 },
  { title: t('logs.contentAudit.nextStatus'), dataIndex: 'nextStatus', slotName: 'nextStatus', width: 110 }
])
const pagination = computed(() => ({ current: current.value, pageSize: pageSize.value, total: total.value, showTotal: true, showPageSize: true }))
const itemPagination = computed(() => ({ current: itemCurrent.value, pageSize: itemPageSize.value, total: itemTotal.value, showTotal: true, showPageSize: true }))

const kindLabel = (kind: string) => ({ article: t('logs.contentAudit.article'), talk: t('logs.contentAudit.talk'), series: t('logs.contentAudit.series') } as Record<string, string>)[kind] || kind
const operationLabel = (operation: string) => ({ batch_status: t('logs.contentAudit.batchStatus'), batch_delete: t('logs.contentAudit.batchDelete'), publish_retry: t('logs.contentAudit.publishRetry') } as Record<string, string>)[operation] || operation
const statusLabel = (status: number) => ({ 0: t('logs.contentAudit.deleted'), 1: t('logs.contentAudit.public'), 2: t('logs.contentAudit.private'), 3: t('logs.contentAudit.draft'), 4: t('logs.contentAudit.scheduled') } as Record<number, string>)[status] || String(status)
const prettyJSON = (value: string) => { try { return JSON.stringify(JSON.parse(value || '{}'), null, 2) } catch { return value || '{}' } }

async function load(): Promise<void> {
  loading.value = true
  error.value = ''
  try {
    const page = await listAdminContentAudits({ current: current.value, size: pageSize.value, ...filters })
    records.value = page.items
    total.value = page.total
  } catch (reason: unknown) {
    error.value = apiErrorMessage(reason, t('logs.contentAudit.loadFailed'))
  } finally {
    loading.value = false
  }
}
function reload(): void { current.value = 1; void load() }
function changePage(page: number): void { current.value = page; void load() }
function changePageSize(size: number): void { pageSize.value = size; current.value = 1; void load() }
async function openDetail(record: ContentAuditRecord): Promise<void> {
  selected.value = record
  detailVisible.value = true
  itemCurrent.value = 1
  await loadItems()
}
async function loadItems(): Promise<void> {
  if (!selected.value) return
  itemsLoading.value = true
  try {
    const page = await listAdminContentAuditItems(selected.value.id, { current: itemCurrent.value, size: itemPageSize.value })
    items.value = page.items
    itemTotal.value = page.total
  } finally {
    itemsLoading.value = false
  }
}
function changeItemPage(page: number): void { itemCurrent.value = page; void loadItems() }
function changeItemPageSize(size: number): void { itemPageSize.value = size; itemCurrent.value = 1; void loadItems() }
onMounted(load)
</script>

<style scoped>
.content-audit-date { min-width: 136px; height: 32px; padding: 0 10px; border: 1px solid var(--color-border-2); border-radius: 6px; background: transparent; color: var(--color-text-1); }
.content-audit-json { overflow: auto; max-height: 240px; padding: 12px; border-radius: 8px; background: var(--color-fill-2); font-size: 12px; line-height: 1.6; }
.content-audit-error { padding: 10px 12px; border-radius: 8px; background: var(--color-danger-light-1); color: rgb(var(--danger-6)); }
</style>