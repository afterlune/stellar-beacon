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
          新增
        </a-button>
      </template>
    </AdminPageHeader>

    <a-card class="admin-panel" :bordered="false">
      <div class="admin-table-toolbar">
        <div class="admin-table-toolbar-main">
          <span class="admin-toolbar-caption">共 {{ total }} 个{{ config.unit }}</span>
          <a-tag v-if="keywords.trim()" color="arcoblue">关键词：{{ keywords.trim() }}</a-tag>
        </div>
        <a-button :loading="loading" size="small" @click="load">
          <template #icon><IconRefresh /></template>
          刷新
        </a-button>
      </div>

      <a-alert v-if="errorMessage" type="error" closable @close="errorMessage = ''">{{ errorMessage }}</a-alert>

      <div class="admin-table-shell">
        <a-table
          :data="records"
          :columns="columns"
          :loading="loading"
          :pagination="pagination"
          row-key="id"
          @page-change="changePage"
          @page-size-change="changePageSize">
          <template #id="{ record }"><span class="admin-id-cell">#{{ record.id }}</span></template>
          <template #name="{ record }">
            <span class="admin-title-cell">{{ record[config.nameKey] || '未命名' }}</span>
          </template>
          <template #articleCount="{ record }">
            <span class="admin-num-cell">{{ formatNumber(record.articleCount ?? 0) }}</span>
          </template>
          <template #createTime="{ record }"><span class="admin-cell-nowrap">{{ formatDateTime(record.createTime) }}</span></template>
          <template #actions="{ record }">
            <a-space class="admin-action-space">
              <a-button type="text" size="small" @click="openEdit(record)">编辑</a-button>
              <a-popconfirm
                :content="`删除「${record[config.nameKey] || '该条目'}」后，已关联的文章会失去这个${config.unit}，确认删除吗？`"
                @ok="remove(record.id)">
                <a-button type="text" status="danger" size="small">删除</a-button>
              </a-popconfirm>
            </a-space>
          </template>
          <template #empty>
            <AdminEmptyState
              :icon="props.kind === 'categories' ? IconFolder : IconTags"
              :title="keywords.trim() ? `没有匹配的${config.unit}` : `还没有${config.unit}`"
              :description="keywords.trim() ? '换个关键词再试一次。' : `先创建第一个${config.unit}，再回到文章编辑器里使用它。`">
              <a-button v-if="keywords.trim()" size="small" @click="clearKeywords">清空搜索</a-button>
              <a-button v-else type="primary" size="small" @click="openCreate">新增{{ config.unit }}</a-button>
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
          :rules="[{ required: true, message: `请输入${config.fieldLabel}` }]">
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
import { computed, onMounted, reactive, ref } from 'vue'
import { Message } from '@arco-design/web-vue'
import { IconFolder, IconPlus, IconRefresh, IconTags } from '@arco-design/web-vue/es/icon'

import { apiErrorMessage, deleteTaxonomy, listAdminPage, saveTaxonomy } from '@/api/http'
import AdminEmptyState from '@/components/AdminEmptyState.vue'
import AdminPageHeader from '@/components/AdminPageHeader.vue'
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

const configs = {
  categories: {
    title: '分类管理',
    description: '为文章建立稳定的归属，让读者更容易找到同一主题下的内容。',
    placeholder: '搜索分类名',
    fieldLabel: '分类名',
    unit: '分类',
    createTitle: '新增分类',
    editTitle: '编辑分类',
    inputPlaceholder: '例如：工程化',
    maxLength: 20,
    help: '分类名会出现在文章详情页与归档列表中，建议保持在 2–10 个字。',
    nameKey: 'categoryName' as const,
    endpoint: 'admin/categories'
  },
  tags: {
    title: '标签管理',
    description: '用轻量的关键词连接文章之间的脉络，让相关内容彼此可达。',
    placeholder: '搜索标签名',
    fieldLabel: '标签名',
    unit: '标签',
    createTitle: '新增标签',
    editTitle: '编辑标签',
    inputPlaceholder: '例如：性能优化',
    maxLength: 20,
    help: '标签用于跨分类串联内容，可以多篇文章共享同一个标签。',
    nameKey: 'tagName' as const,
    endpoint: 'admin/tags'
  }
}

const config = computed(() => configs[props.kind])
const columns = computed(() => [
  { title: 'ID', dataIndex: 'id', width: 84, slotName: 'id' },
  { title: config.value.fieldLabel, dataIndex: config.value.nameKey, slotName: 'name' },
  { title: '文章量', dataIndex: 'articleCount', width: 110, slotName: 'articleCount' },
  { title: '创建时间', dataIndex: 'createTime', width: 180, slotName: 'createTime' },
  { title: '操作', dataIndex: 'actions', width: 150, slotName: 'actions' }
])

const keywords = ref('')
const current = ref(1)
const size = ref(10)
const total = ref(0)
const records = ref<Row[]>([])
const loading = ref(false)
const saving = ref(false)
const errorMessage = ref('')
const dialogError = ref('')
const dialogVisible = ref(false)
const editing = ref(false)
const form = reactive({ id: 0, name: '' })
const pagination = computed(() => tablePagination(current.value, size.value, total.value))

onMounted(() => void reload())

async function reload(): Promise<void> {
  current.value = 1
  await load()
}

function clearKeywords(): void {
  keywords.value = ''
  void reload()
}

async function load(): Promise<void> {
  loading.value = true
  errorMessage.value = ''
  try {
    const page = await listAdminPage<Row>(config.value.endpoint, {
      current: current.value,
      size: size.value,
      keywords: keywords.value.trim()
    })
    records.value = page.items
    total.value = page.total
  } catch (error) {
    errorMessage.value = apiErrorMessage(error, '列表加载失败')
    Message.error(errorMessage.value)
  } finally {
    loading.value = false
  }
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
    dialogError.value = `请输入${config.value.fieldLabel}`
    return false
  }
  saving.value = true
  dialogError.value = ''
  try {
    await saveTaxonomy(props.kind, { id: form.id, [config.value.nameKey]: name })
    await load()
    Message.success(editing.value ? `${config.value.unit}已更新` : `${config.value.unit}已创建`)
    return true
  } catch (error) {
    dialogError.value = apiErrorMessage(error, '保存失败')
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
    Message.success(`${config.value.unit}已删除`)
  } catch (error) {
    Message.error(apiErrorMessage(error, '删除失败'))
  }
}

function changePage(page: number): void {
  current.value = page
  void load()
}

function changePageSize(pageSize: number): void {
  size.value = pageSize
  current.value = 1
  void load()
}
</script>
