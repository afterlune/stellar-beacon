<template>
  <section class="admin-page">
    <AdminPageHeader :title="t('comments.links.title')" :description="t('comments.links.description')">
      <template #actions>
        <a-input-search
          v-model="keywords"
          class="admin-filter-input"
          :placeholder="t('comments.links.searchPlaceholder')"
          allow-clear
          @search="reload" />
        <a-button type="primary" @click="openEditor()">
          <template #icon><IconPlus /></template>
          {{ t('common.create') }}
        </a-button>
      </template>
    </AdminPageHeader>

    <a-card class="admin-panel" :bordered="false">
      <div class="admin-table-toolbar">
        <div class="admin-table-toolbar-main">
          <a-tag v-if="keywords.trim()" color="arcoblue">{{ t('comments.links.keyword', { keyword: keywords.trim() }) }}</a-tag>
          <a-button v-if="keywords.trim()" type="text" size="small" @click="clearKeywords">{{ t('comments.common.clearSearch') }}</a-button>
        </div>
        <div class="admin-table-toolbar-actions">
          <span class="admin-toolbar-caption">
            {{ t('comments.links.total', { total }) }}<template v-if="selectedIds.length">{{ t('comments.common.selectedCount', { count: selectedIds.length }) }}</template>
          </span>
          <a-button :loading="loading" size="small" @click="load">
            <template #icon><IconRefresh /></template>
            {{ t('common.refresh') }}
          </a-button>
        </div>
      </div>

      <AdminErrorState v-if="errorMessage" :error="errorMessage" :title="t('comments.links.loadFailed')" @retry="load" />

      <AdminBatchBar :count="selectedIds.length" :hint="t('comments.common.pageCount', { count: links.length })" @clear="clearSelection">
        <a-popconfirm
          :content="t('comments.links.batchDeleteConfirm', { count: selectedIds.length })"
          :disabled="selectedIds.length === 0"
          @ok="deleteLinks(selectedIds)">
          <a-button status="danger" :disabled="selectedIds.length === 0">
            <template #icon><IconDelete /></template>
            {{ t('comments.common.batchDelete') }}
          </a-button>
        </a-popconfirm>
      </AdminBatchBar>

      <div class="admin-table-shell">
        <a-table
          v-model:selected-keys="selectedKeys"
          :row-selection="{ type: 'checkbox', showCheckedAll: true, onlyCurrent: true }"
          :data="links"
          :columns="columns"
          :loading="loading"
          :pagination="pagination"
          row-key="id"
          @page-change="changePage"
          @page-size-change="changePageSize">
          <template #avatar="{ record }">
            <AdminImagePreview
              v-if="isHttpUrl(record.linkAvatar)"
              :src="String(record.linkAvatar)"
              :alt="t('comments.links.avatarAlt', { name: String(record.linkName || t('comments.links.fallbackName')) })"
              :width="44"
              :height="44" />
            <span v-else class="admin-cover-cell friend-link-avatar-fallback" aria-hidden="true"><IconLink /></span>
          </template>
          <template #linkName="{ record }">
            <span class="admin-title-cell">{{ record.linkName || t('comments.links.untitled') }}</span>
          </template>
          <template #address="{ record }">
            <a v-if="isHttpUrl(record.linkAddress)" class="link-address" :href="String(record.linkAddress)" target="_blank" rel="noreferrer noopener" :title="String(record.linkAddress)">
              {{ record.linkAddress }}
            </a>
            <span v-else class="link-address" :title="String(record.linkAddress || '')">{{ record.linkAddress || '—' }}</span>
          </template>
          <template #intro="{ record }">
            <span class="link-intro" :title="String(record.linkIntro || '')">{{ record.linkIntro || '—' }}</span>
          </template>
          <template #createTime="{ record }"><span class="admin-cell-nowrap">{{ formatDateTime(record.createTime) }}</span></template>
          <template #status="{ record }">
            <a-tag v-if="Number(record.status) === 0" color="orange" data-testid="link-status-pending">{{ t('comments.links.statusPending') }}</a-tag>
            <a-tag v-else-if="Number(record.status) === 2" color="red">{{ t('comments.links.statusRejected') }}</a-tag>
            <a-tag v-else color="green">{{ t('comments.links.statusApproved') }}</a-tag>
          </template>
          <template #actions="{ record }">
            <a-space class="admin-action-space">
              <a-button
                v-if="Number(record.status) === 0"
                type="text"
                size="small"
                data-testid="link-approve"
                @click="reviewLinks([Number(record.id)], 1)">{{ t('comments.links.approve') }}</a-button>
              <a-button
                v-if="Number(record.status) === 0"
                type="text"
                size="small"
                status="danger"
                @click="reviewLinks([Number(record.id)], 2)">{{ t('comments.links.reject') }}</a-button>
              <a-button type="text" size="small" @click="openEditor(record)">{{ t('common.edit') }}</a-button>
              <a-popconfirm :content="t('comments.links.deleteConfirm')" @ok="deleteLinks([Number(record.id)])">
                <a-button type="text" status="danger" size="small">{{ t('common.delete') }}</a-button>
              </a-popconfirm>
            </a-space>
          </template>
          <template #empty>
            <AdminEmptyState
              :icon="IconLink"
              :title="keywords.trim() ? t('comments.links.emptySearch') : t('comments.links.empty')"
              :description="keywords.trim() ? t('comments.common.noMatch') : t('comments.links.emptyHint')">
              <a-button v-if="keywords.trim()" size="small" @click="clearKeywords">{{ t('comments.common.clearSearch') }}</a-button>
              <a-button v-else type="primary" size="small" @click="openEditor()">{{ t('comments.links.createTitle') }}</a-button>
            </AdminEmptyState>
          </template>
        </a-table>
      </div>
    </a-card>

    <a-modal
      v-model:visible="editorVisible"
      :title="editor.id ? t('comments.links.editTitle') : t('comments.links.createTitle')"
      :ok-loading="saving"
      :mask-closable="false"
      width="620px"
      @before-ok="saveEditor">
      <a-form :model="editor" layout="vertical">
        <a-form-item field="linkName" :label="t('comments.links.linkName')" required>
          <a-input v-model="editor.linkName" maxlength="20" show-word-limit :placeholder="t('comments.links.namePlaceholder')" />
        </a-form-item>
        <a-form-item field="linkAvatar" :label="t('comments.links.avatarAddress')" required>
          <a-input v-model="editor.linkAvatar" maxlength="255" :placeholder="t('comments.links.avatarPlaceholder')" />
        </a-form-item>
        <a-form-item field="linkAddress" :label="t('comments.links.address')" required>
          <a-input v-model="editor.linkAddress" maxlength="50" :placeholder="t('comments.links.addressPlaceholder')" />
        </a-form-item>
        <a-form-item field="linkIntro" :label="t('comments.links.intro')" required>
          <a-textarea
            v-model="editor.linkIntro"
            maxlength="100"
            show-word-limit
            :auto-size="{ minRows: 3, maxRows: 6 }"
            :placeholder="t('comments.links.introPlaceholder')" />
        </a-form-item>
      </a-form>
      <div v-if="isHttpUrl(editor.linkAvatar)" class="friend-link-preview">
        <AdminImagePreview :src="editor.linkAvatar" :alt="t('comments.links.avatarPreview')" :width="52" :height="52" />
        <span class="admin-field-hint">{{ t('comments.links.avatarHint') }}</span>
      </div>
    </a-modal>
  </section>
</template>

<script setup lang="ts">
import { computed, reactive, ref } from 'vue'
import { Message } from '@arco-design/web-vue'
import { IconDelete, IconLink, IconPlus, IconRefresh } from '@arco-design/web-vue/es/icon'

import { apiErrorMessage, deleteAdminFriendLinks, listAdminFriendLinks, reviewAdminFriendLinks, saveAdminFriendLink } from '@/api/http'
import AdminBatchBar from '@/components/AdminBatchBar.vue'
import AdminEmptyState from '@/components/AdminEmptyState.vue'
import AdminErrorState from '@/components/AdminErrorState.vue'
import AdminImagePreview from '@/components/AdminImagePreview.vue'
import AdminPageHeader from '@/components/AdminPageHeader.vue'
import { useAsyncList } from '@/composables/useAsyncList'
import { useQueryFilters } from '@/composables/useQueryFilters'
import { readStoredPageSize, useStoredPageSize } from '@/composables/useTablePrefs'
import { t } from '@/i18n'
import { formatDateTime, isHttpUrl } from '@/utils/format'
import { tablePagination } from '@/utils/pagination'
import type { AdminFriendLink } from '@stellar-beacon/api-contract'

const VIEW_KEY = 'friend-links'

// 列定义必须在 computed 里生成：它只在 setup 时求值一次，语言切换后不会再更新。
const columns = computed(() => [
  { title: t('comments.links.avatar'), dataIndex: 'linkAvatar', width: 84, slotName: 'avatar' },
  { title: t('common.name'), dataIndex: 'linkName', slotName: 'linkName', width: 160 },
  { title: t('comments.links.columnAddress'), dataIndex: 'linkAddress', slotName: 'address', ellipsis: true, tooltip: true, minWidth: 200 },
  { title: t('comments.links.columnIntro'), dataIndex: 'linkIntro', slotName: 'intro', ellipsis: true, tooltip: true, minWidth: 180 },
  { title: t('comments.common.createdAt'), dataIndex: 'createTime', slotName: 'createTime', width: 168 },
  { title: t('comments.links.reviewPending'), dataIndex: 'status', slotName: 'status', width: 110 },
  { title: t('common.actions'), dataIndex: 'actions', slotName: 'actions', width: 150 }
])

const selectedKeys = ref<Array<string | number>>([])
const keywords = ref('')
const saving = ref(false)
const editorVisible = ref(false)
const editor = reactive({ id: 0, linkName: '', linkAvatar: '', linkAddress: '', linkIntro: '' })

const {
  items: links,
  total,
  current,
  pageSize,
  loading,
  error: errorMessage,
  load,
  reload,
  changePage: gotoPage,
  changePageSize: applyPageSize
} = useAsyncList<AdminFriendLink>(
  async ({ current: page, pageSize: size, signal }) => {
    const result = await listAdminFriendLinks({
      current: page,
      size,
      keywords: keywords.value.trim()
    }, { signal })
    // 重新加载后丢掉已不在当前页的勾选，避免批量删除误伤其它页的数据。
    const available = new Set(result.items.map((link) => Number(link.id)))
    selectedKeys.value = selectedKeys.value.filter((key) => available.has(Number(key)))
    return result
  },
  // fallbackMessage 只在 setup 时取一次值（useAsyncList 的参数是普通字符串），
  // 因此这里保留当前语言的快照；错误块的标题会跟着语言切换重新渲染。
  { pageSize: readStoredPageSize(VIEW_KEY), fallbackMessage: t('comments.links.loadFailed') }
)

useStoredPageSize(VIEW_KEY, pageSize)
useQueryFilters([
  { key: 'keywords', ref: keywords, debounce: true },
  { key: 'page', ref: current }
], { onRestore: () => void load(), onSearch: () => void reload() })

const selectedIds = computed(() =>
  [...new Set(selectedKeys.value.map(Number).filter((id) => Number.isInteger(id) && id > 0))]
)
const pagination = computed(() => tablePagination(current.value, pageSize.value, total.value))

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

function openEditor(link?: AdminFriendLink): void {
  editor.id = Number(link?.id || 0)
  editor.linkName = String(link?.linkName || '')
  editor.linkAvatar = String(link?.linkAvatar || '')
  editor.linkAddress = String(link?.linkAddress || '')
  editor.linkIntro = String(link?.linkIntro || '')
  editorVisible.value = true
}

async function saveEditor(done: (closed: boolean) => void): Promise<void> {
  if (!editor.linkName.trim() || !editor.linkAvatar.trim() || !editor.linkAddress.trim() || !editor.linkIntro.trim()) {
    Message.error(t('comments.links.requiredFields'))
    done(false)
    return
  }
  saving.value = true
  try {
    await saveAdminFriendLink({
      id: editor.id || undefined,
      linkName: editor.linkName.trim(),
      linkAvatar: editor.linkAvatar.trim(),
      linkAddress: editor.linkAddress.trim(),
      linkIntro: editor.linkIntro.trim()
    })
    Message.success(editor.id ? t('comments.links.updated') : t('comments.links.created'))
    editorVisible.value = false
    await load()
    done(true)
  } catch (error) {
    Message.error(apiErrorMessage(error, t('comments.links.saveFailed')))
    done(false)
  } finally {
    saving.value = false
  }
}

const reviewLinks = async (ids: number[], status: number): Promise<void> => {
  try {
    await reviewAdminFriendLinks(ids, status)
    Message.success(status === 1 ? t('comments.links.approved') : t('comments.links.rejected'))
    await load()
  } catch (error) {
    Message.error(apiErrorMessage(error, t('comments.links.reviewFailed')))
  }
}

async function deleteLinks(ids: number[]): Promise<void> {
  const validIds = [...new Set(ids.map(Number).filter((id) => Number.isInteger(id) && id > 0))]
  if (validIds.length === 0) return
  try {
    await deleteAdminFriendLinks(validIds)
    selectedKeys.value = selectedKeys.value.filter((key) => !validIds.includes(Number(key)))
    if (links.value.length === validIds.length && current.value > 1) current.value -= 1
    Message.success(validIds.length > 1 ? t('comments.links.deletedCount', { count: validIds.length }) : t('comments.links.deleted'))
    await load()
  } catch (error) {
    Message.error(apiErrorMessage(error, t('comments.links.deleteFailed')))
  }
}
</script>

<style scoped>
.link-address,
.link-intro {
  display: block;
  max-width: 300px;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.friend-link-avatar-fallback {
  width: 44px;
  height: 44px;
  border-radius: 50%;
  font-size: 18px;
}

.friend-link-preview {
  display: flex;
  align-items: center;
  gap: 14px;
  margin-top: 4px;
  padding: 12px;
  border: 1px solid var(--admin-border);
  border-radius: var(--admin-radius-control);
  background: var(--admin-surface-soft);
}
</style>
