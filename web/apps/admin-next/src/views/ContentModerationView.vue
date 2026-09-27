<template>
  <section class="admin-page moderation-page">
    <AdminPageHeader :title="t('moderation.title')" :description="t('moderation.description')">
      <template #actions>
        <a-input-search
          v-model="keywords"
          class="admin-filter-input"
          :placeholder="t('moderation.searchPlaceholder')"
          allow-clear
          @search="reload" />
        <a-button :loading="loading" @click="load">
          <template #icon><IconRefresh /></template>
          {{ t('moderation.refresh') }}
        </a-button>
      </template>
    </AdminPageHeader>

    <a-card class="admin-panel" :bordered="false">
      <a-tabs v-model:active-key="activeKind" class="moderation-tabs" @change="reload">
        <a-tab-pane key="article" :title="t('moderation.tabs.articles')" />
        <a-tab-pane key="talk" :title="t('moderation.tabs.talks')" />
        <a-tab-pane key="series" :title="t('moderation.tabs.series')" />
        <a-tab-pane key="collection" :title="t('moderation.tabs.collections')" />
      </a-tabs>

      <div class="admin-table-toolbar">
        <div class="admin-table-toolbar-main">
          <a-select v-if="activeKind !== 'collection'" v-model="statusFilter" :placeholder="t('moderation.allStatuses')" allow-clear style="width: 150px" @change="reload">
            <a-option :value="1">{{ t('moderation.published') }}</a-option>
            <a-option :value="2">{{ t('moderation.private') }}</a-option>
            <a-option :value="3">{{ t('moderation.draft') }}</a-option>
            <a-option v-if="activeKind === 'article'" :value="4">{{ t('moderation.scheduled') }}</a-option>
          </a-select>
          <a-select v-model="moderationFilter" :placeholder="t('moderation.allModeration')" allow-clear style="width: 150px" @change="reload">
            <a-option value="visible">{{ t('moderation.visible') }}</a-option>
            <a-option value="hidden">{{ t('moderation.hidden') }}</a-option>
          </a-select>
        </div>
        <span class="admin-toolbar-caption">{{ t('moderation.total', { total }) }}</span>
      </div>

      <AdminErrorState v-if="errorMessage" :error="errorMessage" :title="t('moderation.actionFailed')" @retry="load" />

      <AdminBatchBar
        :count="selectedKeys.length"
        :hint="t('moderation.pageHint', { count: records.length })"
        @clear="clearSelection">
        <a-button size="small" :loading="batchPending" @click="openHide(selectedItems)">{{ t('moderation.batchHide') }}</a-button>
        <a-button size="small" :loading="batchPending" @click="restoreSelected">{{ t('moderation.batchRestore') }}</a-button>
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
          <template #title="{ record }">
            <div class="moderation-title-cell">
              <strong :title="contentTitle(record)">{{ contentTitle(record) }}</strong>
              <small>{{ contentExcerpt(record) }}</small>
            </div>
          </template>
          <template #author="{ record }">
            <div class="moderation-author-cell">
              <span>{{ authorName(record) }}</span>
              <small v-if="authorHandle(record)">@{{ authorHandle(record) }}</small>
            </div>
          </template>
          <template #status="{ record }">
            <a-tag :color="contentVisibilityColor(record)">{{ contentVisibilityLabel(record) }}</a-tag>
          </template>
          <template #moderation="{ record }">
            <a-tag :color="record.moderationStatus === 'hidden' ? 'red' : 'green'">
              {{ record.moderationStatus === 'hidden' ? t('moderation.hidden') : t('moderation.visible') }}
            </a-tag>
            <small v-if="record.moderationReason" class="moderation-reason" :title="String(record.moderationReason)">{{ record.moderationReason }}</small>
          </template>
          <template #recommendation="{ record }">
            <a-tag v-if="activeKind !== 'article'" color="gray">—</a-tag>
            <a-tag v-else :color="Number(record.isFeatured) === 1 ? 'arcoblue' : 'gray'">
              {{ Number(record.isFeatured) === 1 ? t('moderation.featured') : t('moderation.notFeatured') }}
            </a-tag>
          </template>
          <template #updatedAt="{ record }"><span class="admin-cell-nowrap">{{ formatDateTime(String(record.updatedAt || record.updateTime || record.createTime || '')) }}</span></template>
          <template #actions="{ record }">
            <a-dropdown trigger="click" position="br">
              <a-button type="text" size="small">{{ t('moderation.actions') }} <IconDown /></a-button>
              <template #content>
                <a-doption @click="record.moderationStatus === 'hidden' ? restoreItem(record) : openHide([record])">
                  {{ record.moderationStatus === 'hidden' ? t('moderation.restore') : t('moderation.hide') }}
                </a-doption>
                <template v-if="activeKind === 'article'">
                  <a-doption :disabled="record.moderationStatus === 'hidden' || Number(record.status) !== 1" @click="toggleRecommendation(record)">
                    {{ Number(record.isFeatured) === 1 ? t('moderation.unrecommend') : t('moderation.recommend') }}
                  </a-doption>
                  <a-doption :disabled="record.moderationStatus === 'hidden' || Number(record.status) !== 1" @click="sendNewsletter(record)">
                    {{ t('moderation.newsletter') }}
                  </a-doption>
                </template>
                <a-doption v-if="isOwner(record) && activeKind !== 'collection'" @click="editItem(record)">
                  {{ activeKind === 'series' ? t('moderation.manageSeries') : t('moderation.edit') }}
                </a-doption>
                <a-doption v-if="isOwner(record) && activeKind !== 'collection'" class="admin-danger-option" @click="deleteItem(record)">{{ t('moderation.delete') }}</a-doption>
              </template>
            </a-dropdown>
          </template>
          <template #empty>
            <AdminEmptyState :icon="IconSafe" :title="t('moderation.empty')" />
          </template>
        </a-table>
      </div>
    </a-card>

    <a-modal
      v-model:visible="hideVisible"
      :title="t('moderation.hideTitle')"
      :ok-loading="batchPending"
      :mask-closable="false"
      @ok="confirmHide">
      <p class="moderation-hide-hint">{{ t('moderation.hideHint') }}</p>
      <a-textarea v-model="hideReason" :placeholder="t('moderation.reasonPlaceholder')" :max-length="255" show-word-limit :auto-size="{ minRows: 3, maxRows: 5 }" />
    </a-modal>
  </section>
</template>

<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import { useRouter } from 'vue-router'
import { Message, Modal } from '@arco-design/web-vue'
import { IconDown, IconRefresh, IconSafe } from '@arco-design/web-vue/es/icon'
import type { AdminArticle, AdminTalk } from '@stellar-beacon/api-contract'

import {
  apiErrorMessage,
  deleteAdminArticles,
  deleteAdminSeries,
  deleteAdminTalks,
  distributeAdminArticle,
  getAdminCollections,
  getAdminSeries,
  listAdminPage,
  moderateAdminContent,
} from '@/api/http'
import AdminBatchBar from '@/components/AdminBatchBar.vue'
import AdminEmptyState from '@/components/AdminEmptyState.vue'
import AdminErrorState from '@/components/AdminErrorState.vue'
import AdminPageHeader from '@/components/AdminPageHeader.vue'
import { t } from '@/i18n'
import { useAuthStore } from '@/stores/auth'
import { formatDateTime } from '@/utils/format'
import { tablePagination } from '@/utils/pagination'

type ContentKind = 'article' | 'talk' | 'series' | 'collection'
type ModerationItem = Record<string, unknown> & { id: number; moderationStatus?: string; status?: number }

const router = useRouter()
const auth = useAuthStore()
const activeKind = ref<ContentKind>('article')
const records = ref<ModerationItem[]>([])
const loading = ref(false)
const batchPending = ref(false)
const errorMessage = ref('')
const keywords = ref('')
const statusFilter = ref<number | undefined>(undefined)
const moderationFilter = ref<string | undefined>(undefined)
const current = ref(1)
const pageSize = ref(12)
const total = ref(0)
const selectedKeys = ref<Array<string | number>>([])
const hideVisible = ref(false)
const hideReason = ref('')
const hideTargets = ref<ModerationItem[]>([])

const pagination = computed(() => tablePagination(current.value, pageSize.value, total.value))
const selectedItems = computed(() => records.value.filter((item) => selectedKeys.value.map(Number).includes(Number(item.id))))

const columns = computed(() => [
  { title: t('common.title'), dataIndex: 'title', slotName: 'title', width: 300 },
  { title: t('moderation.author'), dataIndex: 'author', slotName: 'author', width: 150 },
  { title: t('moderation.visibility'), dataIndex: 'status', slotName: 'status', width: 94 },
  { title: t('moderation.moderation'), dataIndex: 'moderationStatus', slotName: 'moderation', width: 190 },
  { title: t('moderation.recommendation'), dataIndex: 'isFeatured', slotName: 'recommendation', width: 110 },
  { title: t('common.updatedAt'), dataIndex: 'updateTime', slotName: 'updatedAt', width: 170 },
  { title: t('moderation.actions'), dataIndex: 'actions', slotName: 'actions', width: 100 }
])

async function load(): Promise<void> {
  loading.value = true
  errorMessage.value = ''
  try {
    const params = {
      current: current.value,
      size: pageSize.value,
      keywords: keywords.value.trim(),
      status: statusFilter.value ?? 0,
      moderationStatus: moderationFilter.value || ''
    }
    const page = activeKind.value === 'article'
      ? await listAdminPage<AdminArticle>('admin/articles', params)
      : activeKind.value === 'talk'
        ? await listAdminPage<AdminTalk>('admin/talks', params)
        : activeKind.value === 'series'
          ? await getAdminSeries(params)
          : await getAdminCollections(params)
    records.value = page.items as ModerationItem[]
    total.value = page.total
    clearSelection()
  } catch (error) {
    records.value = []
    total.value = 0
    errorMessage.value = apiErrorMessage(error, t('moderation.actionFailed'))
  } finally {
    loading.value = false
  }
}

function reload(): void {
  current.value = 1
  void load()
}
watch(activeKind, reload)

function changePage(page: number): void {
  current.value = page
  void load()
}
function changePageSize(size: number): void {
  pageSize.value = size
  current.value = 1
  void load()
}
function clearSelection(): void {
  selectedKeys.value = []
}
function isOwner(record: ModerationItem): boolean {
  return Number(record.userId) > 0 && Number(record.userId) === Number(auth.user?.userInfoId || auth.user?.id || 0)
}
function contentTitle(record: ModerationItem): string {
  if (activeKind.value === 'collection') return String(record.title || '')
  if (activeKind.value === 'talk') return String(record.content || '')
  if (activeKind.value === 'series') return String(record.seriesName || '')
  return String(record.articleTitle || '')
}
function contentExcerpt(record: ModerationItem): string {
  if (activeKind.value === 'collection') return String(record.description || '')
  if (activeKind.value === 'article') return String(record.categoryName || '')
  if (activeKind.value === 'series') return String(record.seriesDesc || '')
  return String(record.createTime || '')
}
function nestedOwner(record: ModerationItem): Record<string, unknown> {
  return (record.owner || {}) as Record<string, unknown>
}
function authorName(record: ModerationItem): string {
  const owner = nestedOwner(record)
  return String(record.authorNickname || record.nickname || record.authorHandle || owner.nickname || owner.handle || '—')
}
function authorHandle(record: ModerationItem): string {
  const owner = nestedOwner(record)
  return String(record.authorHandle || owner.handle || '')
}
function contentVisibilityLabel(record: ModerationItem): string {
  if (activeKind.value !== 'collection') return visibilityLabel(record.status)
  const visibility = String(record.visibility || 'private')
  if (visibility === 'public') return t('moderation.collectionPublic')
  if (visibility === 'unlisted') return t('moderation.collectionUnlisted')
  return t('moderation.collectionPrivate')
}
function contentVisibilityColor(record: ModerationItem): string {
  if (activeKind.value !== 'collection') return visibilityColor(record.status)
  const visibility = String(record.visibility || 'private')
  if (visibility === 'public') return 'green'
  if (visibility === 'unlisted') return 'orange'
  return 'arcoblue'
}
function visibilityLabel(value: unknown): string {
  const status = Number(value)
  if (status === 1) return t('moderation.published')
  if (status === 2) return t('moderation.private')
  if (status === 3) return t('moderation.draft')
  if (status === 4) return t('moderation.scheduled')
  return '—'
}
function visibilityColor(value: unknown): string {
  const status = Number(value)
  if (status === 1) return 'green'
  if (status === 2) return 'orange'
  if (status === 3) return 'arcoblue'
  return 'purple'
}
function openHide(items: ModerationItem[]): void {
  if (items.length === 0) return
  hideTargets.value = items
  hideReason.value = ''
  hideVisible.value = true
}
async function confirmHide(): Promise<void> {
  const reason = hideReason.value.trim()
  if (!reason) {
    Message.warning(t('moderation.reasonRequired'))
    return
  }
  batchPending.value = true
  const results = await Promise.allSettled(hideTargets.value.map((item) => moderateAdminContent({
    contentType: activeKind.value,
    id: Number(item.id),
    hidden: true,
    reason
  })))
  const failed = results.filter((result) => result.status === 'rejected').length
  batchPending.value = false
  hideVisible.value = false
  if (failed > 0) Message.warning(t('moderation.partialFailure', { failed }))
  else Message.success(t('moderation.hiddenSuccess'))
  await load()
}
async function restoreItem(item: ModerationItem): Promise<void> {
  Modal.confirm({
    title: t('moderation.restore'),
    content: t('moderation.confirmRestore'),
    okText: t('moderation.restore'),
    cancelText: t('common.cancel'),
    onOk: async () => {
      try {
        await moderateAdminContent({ contentType: activeKind.value, id: Number(item.id), hidden: false, reason: '' })
        Message.success(t('moderation.restoredSuccess'))
        await load()
      } catch (error) {
        Message.error(apiErrorMessage(error, t('moderation.actionFailed')))
      }
    }
  })
}
async function restoreSelected(): Promise<void> {
  const items = selectedItems.value.filter((item) => item.moderationStatus === 'hidden')
  if (items.length === 0) return
  batchPending.value = true
  const results = await Promise.allSettled(items.map((item) => moderateAdminContent({ contentType: activeKind.value, id: Number(item.id), hidden: false, reason: '' })))
  const failed = results.filter((result) => result.status === 'rejected').length
  batchPending.value = false
  if (failed > 0) Message.warning(t('moderation.partialFailure', { failed }))
  else Message.success(t('moderation.restoredSuccess'))
  await load()
}
async function toggleRecommendation(item: ModerationItem): Promise<void> {
  try {
    const featured = Number(item.isFeatured) !== 1
    await distributeAdminArticle(Number(item.id), { featured, newsletter: false })
    Message.success(featured ? t('moderation.recommended') : t('moderation.recommendationRemoved'))
    await load()
  } catch (error) {
    Message.error(apiErrorMessage(error, t('moderation.actionFailed')))
  }
}
function sendNewsletter(item: ModerationItem): void {
  Modal.confirm({
    title: t('moderation.newsletter'),
    content: `${t('moderation.newsletterConfirm', { title: contentTitle(item) })}\n${t('moderation.newsletterHint')}`,
    okText: t('moderation.newsletter'),
    cancelText: t('common.cancel'),
    onOk: async () => {
      try {
        await distributeAdminArticle(Number(item.id), { featured: Number(item.isFeatured) === 1, newsletter: true })
        Message.success(t('moderation.newsletterSent'))
      } catch (error) {
        Message.error(apiErrorMessage(error, t('moderation.actionFailed')))
      }
    }
  })
}
function editItem(item: ModerationItem): void {
  if (activeKind.value === 'article') void router.push(`/articles/${item.id}`)
  else if (activeKind.value === 'talk') void router.push(`/talks/${item.id}`)
  else void router.push('/series')
}
function deleteItem(item: ModerationItem): void {
  Modal.confirm({
    title: t('moderation.delete'),
    content: t('moderation.deleteConfirm', { title: contentTitle(item) }),
    okText: t('moderation.delete'),
    cancelText: t('common.cancel'),
    okButtonProps: { status: 'danger' },
    onOk: async () => {
      try {
        if (activeKind.value === 'article') await deleteAdminArticles([Number(item.id)])
        else if (activeKind.value === 'talk') await deleteAdminTalks([Number(item.id)])
        else await deleteAdminSeries([Number(item.id)])
        Message.success(t('moderation.deleted'))
        await load()
      } catch (error) {
        Message.error(apiErrorMessage(error, t('moderation.actionFailed')))
      }
    }
  })
}

void load()
</script>

<style scoped>
.moderation-tabs {
  margin-top: -8px;
  margin-bottom: 12px;
}
.moderation-title-cell {
  display: flex;
  min-width: 0;
  flex-direction: column;
  gap: 4px;
}
.moderation-title-cell strong,
.moderation-title-cell small {
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
.moderation-title-cell small,
.moderation-author-cell small,
.moderation-reason {
  display: block;
  color: var(--admin-muted);
  font-size: 12px;
}
.moderation-author-cell span,
.moderation-author-cell small {
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
.moderation-reason {
  max-width: 180px;
  margin-top: 4px;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
.moderation-hide-hint {
  margin: 0 0 12px;
  color: var(--admin-muted);
  line-height: 1.6;
}
</style>
