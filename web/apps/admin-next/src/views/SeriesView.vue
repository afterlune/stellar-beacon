<template>
  <section class="admin-page">
    <AdminPageHeader :title="t('series.title')" :description="t('series.description')">
      <template #actions>
        <a-input-search
          v-model="keywords"
          class="admin-filter-input"
          :placeholder="t('series.searchPlaceholder')"
          allow-clear
          @search="reload" />
        <a-button type="primary" data-testid="series-create" @click="openEditor()">
          <template #icon><IconPlus /></template>
          {{ t('common.create') }}
        </a-button>
      </template>
    </AdminPageHeader>

    <a-card class="admin-panel" :bordered="false">
      <div class="admin-table-toolbar">
        <div class="admin-table-toolbar-main">
          <span class="admin-toolbar-caption">{{ t('series.total', { total }) }}</span>
        </div>
        <div class="admin-table-toolbar-actions">
          <a-button :loading="loading" size="small" @click="load">
            <template #icon><IconRefresh /></template>
            {{ t('common.refresh') }}
          </a-button>
        </div>
      </div>

      <AdminErrorState v-if="errorMessage" :error="errorMessage" :title="t('series.loadFailed')" @retry="load" />

      <div class="admin-table-shell">
        <a-table
          :data="series"
          :columns="columns"
          :loading="loading"
          :pagination="pagination"
          row-key="id"
          @page-change="changePage"
          @page-size-change="changePageSize">
          <template #seriesName="{ record }">
            <span class="admin-title-cell" data-testid="series-name">{{ record.seriesName }}</span>
          </template>
          <template #seriesDesc="{ record }">
            <span class="admin-muted-cell">{{ record.seriesDesc || '—' }}</span>
          </template>
          <template #articleCount="{ record }">
            <span class="admin-num-cell" data-testid="series-count">{{ formatNumber(record.articleCount) }}</span>
          </template>
          <template #updatedAt="{ record }">
            <span class="admin-cell-nowrap">{{ formatDateTime(record.updateTime) }}</span>
          </template>
          <template #actions="{ record }">
            <a-space class="admin-action-space">
              <a-button v-if="isOwner(record)" type="text" size="small" @click="openEditor(record)">{{ t('common.edit') }}</a-button>
              <a-popconfirm v-if="isOwner(record)" :content="t('series.deleteConfirm', { name: String(record.seriesName) })" @ok="removeSeries(record.id)">
                <a-button type="text" size="small" status="danger">{{ t('common.delete') }}</a-button>
              </a-popconfirm>
            </a-space>
          </template>
        </a-table>
      </div>
    </a-card>

    <a-modal
      v-model:visible="editorVisible"
      :title="form.id ? t('series.edit') : t('series.create')"
      :ok-loading="saving"
      :mask-closable="false"
      @ok="save">
      <a-form :model="form" layout="vertical">
        <a-form-item field="seriesName" :label="t('series.name')" required>
          <a-input v-model="form.seriesName" maxlength="50" show-word-limit :placeholder="t('series.namePlaceholder')" />
        </a-form-item>
        <a-form-item field="seriesDesc" :label="t('series.desc')">
          <a-textarea v-model="form.seriesDesc" maxlength="255" :auto-size="{ minRows: 2, maxRows: 4 }" />
        </a-form-item>
        <a-form-item field="cover" :label="t('series.cover')">
          <a-input v-model="form.cover" maxlength="1024" :placeholder="t('series.coverPlaceholder')" />
        </a-form-item>
      </a-form>
    </a-modal>
  </section>
</template>

<script lang="ts">
import { computed, defineComponent, onMounted, reactive, ref } from 'vue'
import { Message } from '@arco-design/web-vue'
import { IconPlus, IconRefresh } from '@arco-design/web-vue/es/icon'
import AdminErrorState from '@/components/AdminErrorState.vue'
import AdminPageHeader from '@/components/AdminPageHeader.vue'
import { getAdminSeries, saveAdminSeries, deleteAdminSeries } from '@/api/http'
import { t } from '@/i18n'
import { useAuthStore } from '@/stores/auth'
import { formatDateTime, formatNumber } from '@/utils/format'

interface SeriesRow {
  id: number
  userId?: number
  seriesName: string
  seriesDesc: string
  cover: string
  articleCount: number
  updateTime: string
}

export default defineComponent({
  name: 'SeriesView',
  components: { AdminErrorState, AdminPageHeader, IconPlus, IconRefresh },
  setup() {
    const auth = useAuthStore()
    const series = ref<SeriesRow[]>([])
    const loading = ref(false)
    const saving = ref(false)
    const errorMessage = ref('')
    const keywords = ref('')
    const total = ref(0)
    const current = ref(1)
    const pageSize = ref(10)
    const editorVisible = ref(false)
    const form = reactive({ id: 0, seriesName: '', seriesDesc: '', cover: '' })

    const columns = computed(() => [
      { title: t('series.name'), dataIndex: 'seriesName', slotName: 'seriesName' },
      { title: t('series.desc'), dataIndex: 'seriesDesc', slotName: 'seriesDesc' },
      { title: t('series.articleCount'), dataIndex: 'articleCount', slotName: 'articleCount', width: 110 },
      { title: t('series.updatedAt'), dataIndex: 'updateTime', slotName: 'updatedAt', width: 180 },
      { title: t('common.actions'), dataIndex: 'actions', slotName: 'actions', width: 150 }
    ])

    const pagination = computed(() => ({
      current: current.value,
      pageSize: pageSize.value,
      total: total.value,
      showTotal: true,
      showPageSize: true
    }))

    const load = () => {
      loading.value = true
      errorMessage.value = ''
      getAdminSeries({ current: current.value, size: pageSize.value, keywords: keywords.value.trim() })
        .then((payload) => {
          series.value = Array.isArray(payload.items) ? (payload.items as unknown as SeriesRow[]) : []
          total.value = Number(payload.total || 0)
        })
        .catch((error: unknown) => {
          series.value = []
          total.value = 0
          errorMessage.value = error instanceof Error ? error.message : t('series.loadFailed')
        })
        .finally(() => {
          loading.value = false
        })
    }

    const reload = () => {
      current.value = 1
      load()
    }

    const changePage = (page: number) => {
      current.value = page
      load()
    }

    const changePageSize = (size: number) => {
      pageSize.value = size
      current.value = 1
      load()
    }

    const openEditor = (record?: SeriesRow) => {
      form.id = record?.id || 0
      form.seriesName = record?.seriesName || ''
      form.seriesDesc = record?.seriesDesc || ''
      form.cover = record?.cover || ''
      editorVisible.value = true
    }

    const save = () => {
      if (!form.seriesName.trim()) {
        Message.error(t('series.nameRequired'))
        return
      }
      saving.value = true
      saveAdminSeries({ ...form, seriesName: form.seriesName.trim() })
        .then(() => {
          editorVisible.value = false
          Message.success(t('series.saved'))
          load()
        })
        .catch((error: unknown) => {
          Message.error(error instanceof Error ? error.message : t('series.saveFailed'))
        })
        .finally(() => {
          saving.value = false
        })
    }

    const isOwner = (record?: SeriesRow) => {
      const currentUserId = Number(auth.user?.userInfoId || auth.user?.id || 0)
      return Boolean(record && currentUserId > 0 && Number(record.userId) === currentUserId)
    }

    const removeSeries = (id: number) => {
      deleteAdminSeries([id])
        .then(() => {
          Message.success(t('series.deleted'))
          load()
        })
        .catch((error: unknown) => {
          Message.error(error instanceof Error ? error.message : t('series.deleteFailed'))
        })
    }

    onMounted(load)

    return { t, series, loading, saving, errorMessage, keywords, total, pagination, columns, editorVisible, form, load, reload, changePage, changePageSize, openEditor, save, removeSeries, isOwner, formatDateTime, formatNumber }
  }
})
</script>
