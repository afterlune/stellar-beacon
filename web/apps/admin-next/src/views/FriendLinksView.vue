<template>
  <section class="admin-page">
    <AdminPageHeader title="友链管理" description="管理博客友情链接。">
      <template #actions>
        <a-input-search
          v-model="keywords"
          class="admin-filter-input"
          placeholder="搜索友链名称"
          allow-clear
          @search="reload" />
        <a-button type="primary" @click="openEditor()">
          <template #icon><IconPlus /></template>
          新增
        </a-button>
      </template>
    </AdminPageHeader>

    <a-card class="admin-panel" :bordered="false">
      <div class="admin-table-toolbar">
        <div class="admin-table-toolbar-main">
          <a-tag v-if="keywords.trim()" color="arcoblue">关键词：{{ keywords.trim() }}</a-tag>
          <a-button v-if="keywords.trim()" type="text" size="small" @click="clearKeywords">清空搜索</a-button>
        </div>
        <div class="admin-table-toolbar-actions">
          <span class="admin-toolbar-caption">
            共 {{ total }} 条友链<template v-if="selectedIds.length"> · 已选 {{ selectedIds.length }} 条</template>
          </span>
          <a-button :loading="loading" size="small" @click="load">
            <template #icon><IconRefresh /></template>
            刷新
          </a-button>
        </div>
      </div>

      <AdminErrorState v-if="errorMessage" :error="errorMessage" title="友链列表加载失败" @retry="load" />

      <AdminBatchBar :count="selectedIds.length" :hint="`本页 ${links.length} 条`" @clear="clearSelection">
        <a-popconfirm
          :content="`确定删除选中的 ${selectedIds.length} 条友链吗？`"
          :disabled="selectedIds.length === 0"
          @ok="deleteLinks(selectedIds)">
          <a-button status="danger" :disabled="selectedIds.length === 0">
            <template #icon><IconDelete /></template>
            批量删除
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
              :alt="`${String(record.linkName || '友链')} 头像`"
              :width="44"
              :height="44" />
            <span v-else class="admin-cover-cell friend-link-avatar-fallback" aria-hidden="true"><IconLink /></span>
          </template>
          <template #linkName="{ record }">
            <span class="admin-title-cell">{{ record.linkName || '未命名友链' }}</span>
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
          <template #actions="{ record }">
            <a-space class="admin-action-space">
              <a-button type="text" size="small" @click="openEditor(record)">编辑</a-button>
              <a-popconfirm content="确定删除该友链吗？" @ok="deleteLinks([Number(record.id)])">
                <a-button type="text" status="danger" size="small">删除</a-button>
              </a-popconfirm>
            </a-space>
          </template>
          <template #empty>
            <AdminEmptyState
              :icon="IconLink"
              :title="keywords.trim() ? '没有匹配的友链' : '还没有友链'"
              :description="keywords.trim() ? '换个关键词再试一次。' : '添加友情链接后，会显示在博客前台。'">
              <a-button v-if="keywords.trim()" size="small" @click="clearKeywords">清空搜索</a-button>
              <a-button v-else type="primary" size="small" @click="openEditor()">新增友链</a-button>
            </AdminEmptyState>
          </template>
        </a-table>
      </div>
    </a-card>

    <a-modal
      v-model:visible="editorVisible"
      :title="editor.id ? '编辑友链' : '新增友链'"
      :ok-loading="saving"
      :mask-closable="false"
      width="620px"
      @before-ok="saveEditor">
      <a-form :model="editor" layout="vertical">
        <a-form-item field="linkName" label="友链名称" required>
          <a-input v-model="editor.linkName" maxlength="20" show-word-limit placeholder="对方站点的名称" />
        </a-form-item>
        <a-form-item field="linkAvatar" label="头像地址" required>
          <a-input v-model="editor.linkAvatar" maxlength="255" placeholder="HTTPS 图片地址" />
        </a-form-item>
        <a-form-item field="linkAddress" label="链接地址" required>
          <a-input v-model="editor.linkAddress" maxlength="50" placeholder="友链主页地址" />
        </a-form-item>
        <a-form-item field="linkIntro" label="友链介绍" required>
          <a-textarea
            v-model="editor.linkIntro"
            maxlength="100"
            show-word-limit
            :auto-size="{ minRows: 3, maxRows: 6 }"
            placeholder="一句话介绍对方站点" />
        </a-form-item>
      </a-form>
      <div v-if="isHttpUrl(editor.linkAvatar)" class="friend-link-preview">
        <AdminImagePreview :src="editor.linkAvatar" alt="头像预览" :width="52" :height="52" />
        <span class="admin-field-hint">头像会以圆形显示在友链列表中。</span>
      </div>
    </a-modal>
  </section>
</template>

<script setup lang="ts">
import { computed, reactive, ref } from 'vue'
import { Message } from '@arco-design/web-vue'
import { IconDelete, IconLink, IconPlus, IconRefresh } from '@arco-design/web-vue/es/icon'

import { apiErrorMessage, deleteAdminFriendLinks, listAdminFriendLinks, saveAdminFriendLink } from '@/api/http'
import AdminBatchBar from '@/components/AdminBatchBar.vue'
import AdminEmptyState from '@/components/AdminEmptyState.vue'
import AdminErrorState from '@/components/AdminErrorState.vue'
import AdminImagePreview from '@/components/AdminImagePreview.vue'
import AdminPageHeader from '@/components/AdminPageHeader.vue'
import { useAsyncList } from '@/composables/useAsyncList'
import { useQueryFilters } from '@/composables/useQueryFilters'
import { readStoredPageSize, useStoredPageSize } from '@/composables/useTablePrefs'
import { formatDateTime, isHttpUrl } from '@/utils/format'
import { tablePagination } from '@/utils/pagination'
import type { AdminFriendLink } from '@stellar-beacon/api-contract'

const VIEW_KEY = 'friend-links'

const columns = [
  { title: '头像', dataIndex: 'linkAvatar', width: 84, slotName: 'avatar' },
  { title: '名称', dataIndex: 'linkName', slotName: 'linkName', width: 160 },
  { title: '地址', dataIndex: 'linkAddress', slotName: 'address', ellipsis: true, tooltip: true, minWidth: 200 },
  { title: '介绍', dataIndex: 'linkIntro', slotName: 'intro', ellipsis: true, tooltip: true, minWidth: 180 },
  { title: '创建时间', dataIndex: 'createTime', slotName: 'createTime', width: 168 },
  { title: '操作', dataIndex: 'actions', slotName: 'actions', width: 150 }
]

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
  { pageSize: readStoredPageSize(VIEW_KEY), fallbackMessage: '友链列表加载失败' }
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
    Message.error('友链名称、头像、地址和介绍不能为空')
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
    Message.success(editor.id ? '友链已更新' : '友链已创建')
    editorVisible.value = false
    await load()
    done(true)
  } catch (error) {
    Message.error(apiErrorMessage(error, '友链保存失败'))
    done(false)
  } finally {
    saving.value = false
  }
}

async function deleteLinks(ids: number[]): Promise<void> {
  const validIds = [...new Set(ids.map(Number).filter((id) => Number.isInteger(id) && id > 0))]
  if (validIds.length === 0) return
  try {
    await deleteAdminFriendLinks(validIds)
    selectedKeys.value = selectedKeys.value.filter((key) => !validIds.includes(Number(key)))
    if (links.value.length === validIds.length && current.value > 1) current.value -= 1
    Message.success(validIds.length > 1 ? `已删除 ${validIds.length} 条友链` : '友链已删除')
    await load()
  } catch (error) {
    Message.error(apiErrorMessage(error, '友链删除失败'))
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
