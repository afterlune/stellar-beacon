<template>
  <section class="admin-page">
    <AdminPageHeader :title="config.title" :description="config.description">
      <template #actions>
        <a-space>
          <a-input-search v-model="keywords" class="admin-filter-input" :placeholder="config.placeholder" allow-clear @search="reload" />
          <a-button type="primary" @click="openCreate">
            <template #icon><IconPlus /></template>
            新增
          </a-button>
        </a-space>
      </template>
    </AdminPageHeader>
    <a-card class="admin-panel" :bordered="false">
      <a-alert v-if="errorMessage" type="error" closable @close="errorMessage = ''">{{ errorMessage }}</a-alert>
      <div class="admin-table-shell">
        <a-table :data="records" :columns="columns" :loading="loading" :pagination="pagination" row-key="id" @page-change="changePage">
          <template #articleCount="{ record }">{{ record.articleCount ?? 0 }}</template>
          <template #createTime="{ record }">{{ formatTime(record.createTime) }}</template>
          <template #actions="{ record }">
            <a-space class="admin-action-space">
              <a-button type="text" size="small" @click="openEdit(record)">编辑</a-button>
              <a-popconfirm content="确认删除这条记录吗？" @ok="remove(record.id)">
                <a-button type="text" status="danger" size="small">删除</a-button>
              </a-popconfirm>
            </a-space>
          </template>
          <template #empty><div class="admin-table-empty"><a-empty description="暂无数据" /></div></template>
        </a-table>
      </div>
    </a-card>

    <a-modal v-model:visible="dialogVisible" :title="editing ? '编辑' : '新增'" @before-ok="save">
      <a-form :model="form" layout="vertical">
        <a-form-item field="name" :label="config.fieldLabel" :rules="[{ required: true, message: `请输入${config.fieldLabel}` }]">
          <a-input v-model="form.name" :placeholder="config.fieldLabel" />
        </a-form-item>
      </a-form>
    </a-modal>
  </section>
</template>

<script setup lang="ts">
import { computed, onMounted, reactive, ref } from 'vue'
import { Message } from '@arco-design/web-vue'
import { IconPlus } from '@arco-design/web-vue/es/icon'

import { apiErrorMessage, deleteTaxonomy, listAdminPage, saveTaxonomy } from '@/api/http'
import AdminPageHeader from '@/components/AdminPageHeader.vue'
import { formatTime } from '@/utils/format'
import { tablePagination } from '@/utils/pagination'

type Kind = 'categories' | 'tags'
interface Row extends Record<string, unknown> { id: number; categoryName?: string; tagName?: string; articleCount?: number | string; createTime?: string }

const props = defineProps<{ kind: Kind }>()
const config = computed(() => props.kind === 'categories'
  ? { title: '分类管理', description: '给文章一个容易被找到的位置。', placeholder: '搜索分类名', fieldLabel: '分类名', nameKey: 'categoryName' }
  : { title: '标签管理', description: '用轻量的关键词连接文章之间的脉络。', placeholder: '搜索标签名', fieldLabel: '标签名', nameKey: 'tagName' })
const columns = computed(() => [
  { title: 'ID', dataIndex: 'id', width: 80 },
  { title: config.value.fieldLabel, dataIndex: config.value.nameKey },
  { title: '文章量', dataIndex: 'articleCount', slotName: 'articleCount' },
  { title: '创建时间', dataIndex: 'createTime', slotName: 'createTime' },
  { title: '操作', slotName: 'actions', width: 150 }
])
const keywords = ref('')
const current = ref(1)
const size = ref(10)
const total = ref(0)
const records = ref<Row[]>([])
const loading = ref(false)
const errorMessage = ref('')
const dialogVisible = ref(false)
const editing = ref(false)
const form = reactive({ id: 0, name: '' })
const pagination = computed(() => tablePagination(current.value, size.value, total.value))

onMounted(() => void reload())

async function reload(): Promise<void> {
  current.value = 1
  await load()
}

async function load(): Promise<void> {
  loading.value = true
  errorMessage.value = ''
  try {
    const page = await listAdminPage<Row>(`admin/${props.kind}`, { current: current.value, size: size.value, keywords: keywords.value.trim() })
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
  dialogVisible.value = true
}

function openEdit(row: Row): void {
  editing.value = true
  form.id = row.id
  form.name = String(row[config.value.nameKey] || '')
  dialogVisible.value = true
}

async function save(done: (closed: boolean) => void): Promise<void> {
  if (!form.name.trim()) {
    done(false)
    return
  }
  try {
    await saveTaxonomy(props.kind, { id: form.id, [config.value.nameKey]: form.name.trim() })
    await load()
    Message.success('保存成功')
    done(true)
  } catch (error) {
    Message.error(apiErrorMessage(error, '保存失败'))
    done(false)
  }
}

async function remove(id: number): Promise<void> {
  try {
    await deleteTaxonomy(props.kind, [id])
    await load()
    Message.success('删除成功')
  } catch (error) {
    Message.error(apiErrorMessage(error, '删除失败'))
  }
}

function changePage(page: number): void {
  current.value = page
  void load()
}


</script>
