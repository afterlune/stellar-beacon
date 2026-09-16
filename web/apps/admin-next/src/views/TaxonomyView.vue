<template>
  <section class="admin-page">
    <AdminPageHeader :title="config.title" :description="config.description">
      <template #actions>
        <a-input-search
          v-model="keywords"
          class="admin-filter-input"
          :placeholder="config.placeholder"
          allow-clear
          @search="reload" />
        <a-button type="primary" @click="openCreate">
          <template #icon><IconPlus /></template>
          {{ t('taxonomy.common.create') }}
        </a-button>
      </template>
    </AdminPageHeader>

    <a-card class="admin-panel" :bordered="false">
      <div class="admin-table-toolbar">
        <div class="admin-table-toolbar-main">
          <span class="admin-toolbar-caption">{{ t('taxonomy.list.total', { total, unit: config.unit }) }}</span>
          <a-tag v-if="keywords.trim()" color="arcoblue">{{ t('taxonomy.list.keywords', { keywords: keywords.trim() }) }}</a-tag>
        </div>
        <a-button :loading="loading" size="small" @click="load">
          <template #icon><IconRefresh /></template>
          {{ t('taxonomy.common.refresh') }}
        </a-button>
      </div>

      <AdminErrorState v-if="errorMessage" :error="errorMessage" :title="t('taxonomy.list.loadFailed')" @retry="load" />

      <AdminBatchBar :count="selectedIds.length" :hint="t('taxonomy.list.pageCount', { count: records.length, unit: config.unit })" @clear="clearSelection">
        <a-button size="small" status="danger" :loading="batchDeleting" @click="batchDelete">{{ t('taxonomy.batch.delete') }}</a-button>
      </AdminBatchBar>

      <div class="admin-table-shell">
        <a-table
          v-model:selected-keys="selectedKeys"
          :row-selection="{ type: 'checkbox', showCheckedAll: true, onlyCurrent: true }"
          :data="records"
          :columns="columns"
          :loading="loading"
          :pagination="pagination"
          row-key="id"
          @page-change="changePage"
          @page-size-change="changePageSize">
          <template #id="{ record }"><span class="admin-id-cell">#{{ record.id }}</span></template>
          <template #name="{ record }">
            <span class="admin-title-cell">{{ record[config.nameKey] || t('taxonomy.common.untitled') }}</span>
          </template>
          <template #articleCount="{ record }">
            <span class="admin-num-cell">{{ formatNumber(record.articleCount ?? 0) }}</span>
          </template>
          <template #createTime="{ record }"><span class="admin-cell-nowrap">{{ formatDateTime(record.createTime) }}</span></template>
          <template #actions="{ record }">
            <a-space class="admin-action-space">
              <a-button type="text" size="small" @click="openEdit(record)">{{ t('taxonomy.common.edit') }}</a-button>
              <a-popconfirm
                :content="t('taxonomy.list.deleteConfirm', { name: record[config.nameKey] || t('taxonomy.list.deleteTarget'), unit: config.unit })"
                @ok="remove(record.id)">
                <a-button type="text" status="danger" size="small">{{ t('taxonomy.common.delete') }}</a-button>
              </a-popconfirm>
            </a-space>
          </template>
          <template #empty>
            <AdminEmptyState
              :icon="props.kind === 'categories' ? IconFolder : IconTags"
              :title="keywords.trim() ? t('taxonomy.list.emptyFilteredTitle', { unit: config.unit }) : t('taxonomy.list.emptyTitle', { unit: config.unit })"
              :description="keywords.trim() ? t('taxonomy.list.emptyFilteredHint') : t('taxonomy.list.emptyHint', { unit: config.unit })">
              <a-button v-if="keywords.trim()" size="small" @click="clearKeywords">{{ t('taxonomy.common.clearKeywords') }}</a-button>
              <a-button v-else type="primary" size="small" @click="openCreate">{{ t('taxonomy.list.createFirst', { unit: config.unit }) }}</a-button>
            </AdminEmptyState>
          </template>
        </a-table>
      </div>
    </a-card>

    <a-modal
      v-model:visible="dialogVisible"
      :title="editing ? config.editTitle : config.createTitle"
      :ok-loading="saving"
      :mask-closable="false"
      width="480px"
      @before-ok="save">
      <a-form :model="form" layout="vertical">
        <a-form-item
          field="name"
          :label="config.fieldLabel"
          :rules="[{ required: true, message: t('taxonomy.form.required', { field: config.fieldLabel }) }]">
          <a-input
            v-model="form.name"
            :placeholder="config.inputPlaceholder"
            :max-length="config.maxLength"
            show-word-limit
            @press-enter="submitFromInput" />
          <template #help>{{ config.help }}</template>
        </a-form-item>
      </a-form>
      <a-alert v-if="dialogError" type="error" closable @close="dialogError = ''">{{ dialogError }}</a-alert>
    </a-modal>
  </section>
</template>

<script setup lang="ts">
import { computed, reactive, ref } from 'vue'
import { Message, Modal } from '@arco-design/web-vue'
import { IconFolder, IconPlus, IconRefresh, IconTags } from '@arco-design/web-vue/es/icon'

import { apiErrorMessage, deleteTaxonomy, listAdminPage, saveTaxonomy } from '@/api/http'
import AdminBatchBar from '@/components/AdminBatchBar.vue'
import AdminEmptyState from '@/components/AdminEmptyState.vue'
import AdminErrorState from '@/components/AdminErrorState.vue'
import AdminPageHeader from '@/components/AdminPageHeader.vue'
import { useAsyncList } from '@/composables/useAsyncList'
import { useQueryFilters } from '@/composables/useQueryFilters'
import { readStoredPageSize, useStoredPageSize } from '@/composables/useTablePrefs'
import { t } from '@/i18n'
import { formatDateTime, formatNumber } from '@/utils/format'
import { tablePagination } from '@/utils/pagination'

type Kind = 'categories' | 'tags'

interface Row extends Record<string, unknown> {
  id: number
  categoryName?: string
  tagName?: string
  articleCount?: number | string
  createTime?: string
}

const props = defineProps<{ kind: Kind }>()

// 文案字段用 getter：config 是 computed，语言切换后这些 getter 会重新求值。
const configs = {
  categories: {
    title: () => t('taxonomy.categories.title'),
    description: () => t('taxonomy.categories.description'),
    placeholder: () => t('taxonomy.categories.searchPlaceholder'),
    fieldLabel: () => t('taxonomy.categories.name'),
    unit: () => t('taxonomy.categories.unit'),
    createTitle: () => t('taxonomy.categories.createTitle'),
    editTitle: () => t('taxonomy.categories.editTitle'),
    inputPlaceholder: () => t('taxonomy.categories.namePlaceholder'),
    maxLength: 20,
    help: () => t('taxonomy.categories.nameHelp'),
    nameKey: 'categoryName' as const,
    endpoint: 'admin/categories'
  },
  tags: {
    title: () => t('taxonomy.tags.title'),
    description: () => t('taxonomy.tags.description'),
    placeholder: () => t('taxonomy.tags.searchPlaceholder'),
    fieldLabel: () => t('taxonomy.tags.name'),
    unit: () => t('taxonomy.tags.unit'),
    createTitle: () => t('taxonomy.tags.createTitle'),
    editTitle: () => t('taxonomy.tags.editTitle'),
    inputPlaceholder: () => t('taxonomy.tags.namePlaceholder'),
    maxLength: 20,
    help: () => t('taxonomy.tags.nameHelp'),
    nameKey: 'tagName' as const,
    endpoint: 'admin/tags'
  }
}

/**
 * 把上面带 getter 的配置对象展开成普通值。
 *
 * 视图既要在模板里读 `config.unit` 这种字符串（拼进 `t()` 的占位符），也要读
 * `config.value.endpoint` 这种普通字段，所以这里统一拍平一次：computed 依赖
 * locale，语言切换后整份配置会重新求值。
 */
const config = computed(() => {
  const source = configs[props.kind]
  return {
    title: source.title(),
    description: source.description(),
    placeholder: source.placeholder(),
    fieldLabel: source.fieldLabel(),
    unit: source.unit(),
    createTitle: source.createTitle(),
    editTitle: source.editTitle(),
    inputPlaceholder: source.inputPlaceholder(),
    maxLength: source.maxLength,
    help: source.help(),
    nameKey: source.nameKey as 'categoryName' | 'tagName',
    endpoint: source.endpoint
  }
})
const columns = computed(() => [
  { title: 'ID', dataIndex: 'id', width: 84, slotName: 'id' },
  { title: config.value.fieldLabel, dataIndex: config.value.nameKey, slotName: 'name' },
  { title: t('taxonomy.common.articleCount'), dataIndex: 'articleCount', width: 110, slotName: 'articleCount' },
  { title: t('taxonomy.common.createTime'), dataIndex: 'createTime', width: 180, slotName: 'createTime' },
  { title: t('taxonomy.common.actions'), dataIndex: 'actions', width: 150, slotName: 'actions' }
])

const keywords = ref('')
const saving = ref(false)
const dialogError = ref('')
const dialogVisible = ref(false)
const editing = ref(false)
const selectedKeys = ref<number[]>([])
const batchDeleting = ref(false)
const form = reactive({ id: 0, name: '' })

const VIEW_KEY = 'taxonomy'

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
} = useAsyncList<Row>(
  ({ current: page, pageSize: size, signal }) => listAdminPage<Row>(config.value.endpoint, {
    current: page,
    size,
    keywords: keywords.value.trim()
  }, { signal }),
  // useAsyncList 只在初始化时读一次 options，兜底文案取当前语言即可。
  { pageSize: readStoredPageSize(VIEW_KEY), fallbackMessage: t('taxonomy.list.loadFailed') }
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

function openCreate(): void {
  editing.value = false
  form.id = 0
  form.name = ''
  dialogError.value = ''
  dialogVisible.value = true
}

function openEdit(row: Row): void {
  editing.value = true
  form.id = row.id
  form.name = String(row[config.value.nameKey] || '')
  dialogError.value = ''
  dialogVisible.value = true
}

/** Textarea-style submit: pressing Enter inside the single input saves. */
function submitFromInput(): void {
  if (!form.name.trim() || saving.value) return
  void commit()
}

async function save(done: (closed: boolean) => void): Promise<void> {
  const applied = await commit()
  done(applied)
}

async function commit(): Promise<boolean> {
  const name = form.name.trim()
  if (!name) {
    dialogError.value = t('taxonomy.form.required', { field: config.value.fieldLabel })
    return false
  }
  saving.value = true
  dialogError.value = ''
  try {
    await saveTaxonomy(props.kind, { id: form.id, [config.value.nameKey]: name })
    await load()
    Message.success(t(
      editing.value ? 'taxonomy.item.updated' : 'taxonomy.item.created',
      { unit: config.value.unit }
    ))
    return true
  } catch (error) {
    dialogError.value = apiErrorMessage(error, t('taxonomy.form.saveFailed'))
    Message.error(dialogError.value)
    return false
  } finally {
    saving.value = false
  }
}

async function remove(id: unknown): Promise<void> {
  const targetId = Number(id)
  if (!Number.isInteger(targetId) || targetId <= 0) return
  try {
    await deleteTaxonomy(props.kind, [targetId])
    if (records.value.length === 1 && current.value > 1) current.value -= 1
    await load()
    Message.success(t('taxonomy.item.deleted', { unit: config.value.unit }))
  } catch (error) {
    Message.error(apiErrorMessage(error, t('taxonomy.item.deleteFailed')))
  }
}

function changePage(page: number): void {
  clearSelection()
  gotoPage(page)
}

function changePageSize(size: number): void {
  clearSelection()
  applyPageSize(size)
}

function clearSelection(): void {
  selectedKeys.value = []
}

function batchDelete(): void {
  const ids = selectedIds.value
  if (ids.length === 0) return
  Modal.confirm({
    title: t('taxonomy.batch.delete'),
    content: t('taxonomy.batch.deleteConfirm', { count: ids.length, unit: config.value.unit }),
    okText: t('taxonomy.batch.delete'),
    cancelText: t('taxonomy.common.cancel'),
    okButtonProps: { status: 'danger' },
    onOk: async () => {
      batchDeleting.value = true
      try {
        await deleteTaxonomy(props.kind, ids)
        Message.success(t('taxonomy.batch.deleteSuccess', { count: ids.length, unit: config.value.unit }))
        clearSelection()
        await load()
      } catch (error) {
        Message.error(apiErrorMessage(error, t('taxonomy.batch.deleteFailed')))
      } finally {
        batchDeleting.value = false
      }
    }
  })
}
</script>
