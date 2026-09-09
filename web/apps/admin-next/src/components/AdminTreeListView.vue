<template>
  <section>
    <a-card :title="config.title">
      <template #extra>
        <a-input-search v-model="keywords" :placeholder="config.placeholder" allow-clear style="width: 260px" @search="load" />
      </template>
      <a-alert v-if="errorMessage" type="error" closable @close="errorMessage = ''">{{ errorMessage }}</a-alert>
      <a-table :data="rows" :columns="config.columns" :loading="loading" :pagination="false" row-key="id">
        <template #name="{ record }">
          <span :style="{ paddingLeft: `${record.depth * 20}px` }">{{ record.name }}</span>
        </template>
        <template #hidden="{ record }">{{ formatFlag(record.hidden) }}</template>
        <template #anonymous="{ record }">{{ formatFlag(record.anonymous) }}</template>
        <template #cell="{ record, column }">{{ formatCell(record[column.dataIndex]) }}</template>
        <template #empty><a-empty description="暂无数据" /></template>
      </a-table>
    </a-card>
  </section>
</template>

<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { Message } from '@arco-design/web-vue'

import { apiErrorMessage, listAdminCollection } from '@/api/http'

type TreeMode = 'menus' | 'resources'
interface TreeNode {
  id?: number
  name?: string
  path?: string
  component?: string
  icon?: string
  orderNum?: number
  isHidden?: number
  resourceName?: string
  url?: string
  requestMethod?: string
  isDisable?: number
  isAnonymous?: number
  children?: TreeNode[]
  [key: string]: unknown
}
interface TreeRow extends TreeNode {
  name: string
  depth: number
  hidden?: number
  anonymous?: number
}

const props = defineProps<{ mode: TreeMode }>()
const keywords = ref('')
const loading = ref(false)
const errorMessage = ref('')
const nodes = ref<TreeNode[]>([])

const configs = {
  menus: {
    title: '菜单管理',
    placeholder: '搜索菜单名或路径',
    endpoint: 'admin/menus',
    columns: [
      { title: '菜单', dataIndex: 'name', slotName: 'name' },
      { title: '路径', dataIndex: 'path' },
      { title: '组件', dataIndex: 'component' },
      { title: '排序', dataIndex: 'orderNum' },
      { title: '隐藏', dataIndex: 'hidden', slotName: 'hidden' }
    ]
  },
  resources: {
    title: '接口资源管理',
    placeholder: '搜索资源名或 URL',
    endpoint: 'admin/permissions',
    columns: [
      { title: '资源', dataIndex: 'name', slotName: 'name' },
      { title: 'URL', dataIndex: 'url' },
      { title: '方法', dataIndex: 'requestMethod' },
      { title: '禁用', dataIndex: 'isDisable', slotName: 'hidden' },
      { title: '匿名', dataIndex: 'anonymous', slotName: 'anonymous' }
    ]
  }
} as const

const config = computed(() => configs[props.mode])
const rows = computed(() => flatten(nodes.value, keywords.value.trim().toLowerCase()))

onMounted(() => void load())

async function load(): Promise<void> {
  loading.value = true
  errorMessage.value = ''
  try {
    nodes.value = await listAdminCollection<TreeNode>(config.value.endpoint)
  } catch (error) {
    errorMessage.value = apiErrorMessage(error, '数据加载失败')
    Message.error(errorMessage.value)
  } finally {
    loading.value = false
  }
}

function flatten(items: TreeNode[], query: string, depth = 0): TreeRow[] {
  const result: TreeRow[] = []
  for (const item of items) {
    const name = props.mode === 'menus' ? item.name || '未命名菜单' : item.resourceName || item.name || '未命名资源'
    const searchable = [name, item.path, item.url, item.component].filter(Boolean).join(' ').toLowerCase()
    const children = Array.isArray(item.children) ? item.children : []
    const descendants = flatten(children, query, depth + 1)
    if (!query || searchable.includes(query) || descendants.length > 0) {
      result.push({
        ...item,
        name,
        depth,
        hidden: item.isHidden ?? item.isDisable,
        anonymous: item.isAnonymous
      })
      result.push(...descendants)
    }
  }
  return result
}

function formatFlag(value: unknown): string { return Number(value) === 1 ? '是' : '否' }
function formatCell(value: unknown): string { return value === null || value === undefined || value === '' ? '—' : String(value) }
</script>
