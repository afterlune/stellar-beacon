<template>
  <section>
    <a-card :title="title">
      <template #extra>
        <a-space>
          <a-input-search v-model="keywords" :placeholder="placeholder" allow-clear style="width: 240px" @search="load" />
          <a-button type="primary" @click="openEditor()">新增</a-button>
        </a-space>
      </template>
      <a-alert v-if="errorMessage" type="error" closable @close="errorMessage = ''">{{ errorMessage }}</a-alert>
      <a-table :data="rows" :columns="columns" :loading="loading" :pagination="false" row-key="id">
        <template #name="{ record }">
          <span :style="{ paddingLeft: `${record.depth * 20}px` }">{{ record.name }}</span>
        </template>
        <template #hidden="{ record }">
          <a-switch :model-value="Number(record.isHidden) === 1" @change="(value) => toggleHidden(record, value)" />
        </template>
        <template #anonymous="{ record }">{{ Number(record.isAnonymous) === 1 ? '是' : '否' }}</template>
        <template #actions="{ record }">
          <a-space>
            <a-button type="text" size="small" @click="openEditor(record)">编辑</a-button>
            <a-popconfirm :content="`确定删除${record.name}吗？`" @ok="deleteItem(record.id)">
              <a-button type="text" status="danger" size="small">删除</a-button>
            </a-popconfirm>
          </a-space>
        </template>
        <template #empty><a-empty :description="`暂无${title}`" /></template>
      </a-table>
    </a-card>

    <a-modal v-model:visible="editorVisible" :title="form.id ? `编辑${title}` : `新增${title}`" :ok-loading="saving" width="640px" @ok="save">
      <a-form :model="form" layout="vertical">
        <template v-if="mode === 'menus'">
          <a-form-item field="name" label="菜单名称" required><a-input v-model="form.name" maxlength="20" /></a-form-item>
          <a-form-item field="path" label="路径" required><a-input v-model="form.path" maxlength="100" placeholder="例如 /article-list" /></a-form-item>
          <a-form-item field="component" label="组件路径" required><a-input v-model="form.component" maxlength="100" placeholder="例如 /article/ArticleList.vue" /></a-form-item>
          <a-form-item field="icon" label="图标"><a-input v-model="form.icon" maxlength="50" /></a-form-item>
          <a-form-item field="orderNum" label="排序"><a-input-number v-model="form.orderNum" :min="0" :max="9999" /></a-form-item>
          <a-form-item field="parentId" label="父菜单">
            <a-select v-model="form.parentId" allow-clear placeholder="顶级菜单">
              <a-option :value="0">顶级菜单</a-option>
              <a-option v-for="parent in parentOptions" :key="parent.id" :value="parent.id">{{ parent.name }}</a-option>
            </a-select>
          </a-form-item>
          <a-form-item label="是否隐藏"><a-switch v-model="form.isHidden" :checked-value="1" :unchecked-value="0" /></a-form-item>
        </template>
        <template v-else>
          <a-form-item field="resourceName" label="资源名称" required><a-input v-model="form.resourceName" maxlength="50" /></a-form-item>
          <a-form-item field="url" label="URL" required><a-input v-model="form.url" maxlength="255" placeholder="例如 /admin/articles" /></a-form-item>
          <a-form-item field="requestMethod" label="请求方法" required>
            <a-select v-model="form.requestMethod"><a-option v-for="method in methods" :key="method" :value="method">{{ method }}</a-option></a-select>
          </a-form-item>
          <a-form-item field="parentId" label="父资源">
            <a-select v-model="form.parentId" allow-clear placeholder="顶级资源">
              <a-option :value="0">顶级资源</a-option>
              <a-option v-for="parent in parentOptions" :key="parent.id" :value="parent.id">{{ parent.name }}</a-option>
            </a-select>
          </a-form-item>
          <a-form-item label="匿名访问"><a-switch v-model="form.isAnonymous" :checked-value="1" :unchecked-value="0" /></a-form-item>
        </template>
      </a-form>
    </a-modal>
  </section>
</template>

<script setup lang="ts">
import { computed, onMounted, reactive, ref } from 'vue'
import { Message } from '@arco-design/web-vue'

import {
  apiErrorMessage,
  deleteAdminMenu,
  deleteAdminResource,
  listAdminMenus,
  listAdminResources,
  saveAdminMenu,
  saveAdminResource,
  updateAdminMenuHidden
} from '@/api/http'

type PermissionMode = 'menus' | 'resources'
interface PermissionNode {
  id: number
  name?: string
  resourceName?: string
  path?: string
  url?: string
  component?: string
  icon?: string
  orderNum?: number
  parentId?: number
  isHidden?: number
  isAnonymous?: number
  requestMethod?: string
  children?: PermissionNode[]
  depth?: number
  [key: string]: unknown
}

const props = defineProps<{ mode: PermissionMode }>()
const title = computed(() => props.mode === 'menus' ? '菜单管理' : '接口资源管理')
const placeholder = computed(() => props.mode === 'menus' ? '搜索菜单名或路径' : '搜索资源名或 URL')
const methods = ['GET', 'POST', 'PUT', 'DELETE', 'PATCH']
const columns = computed(() => props.mode === 'menus'
  ? [
      { title: '菜单', dataIndex: 'name', slotName: 'name' },
      { title: '路径', dataIndex: 'path' },
      { title: '组件', dataIndex: 'component' },
      { title: '排序', dataIndex: 'orderNum', width: 90 },
      { title: '隐藏', dataIndex: 'isHidden', slotName: 'hidden', width: 90 },
      { title: '操作', dataIndex: 'actions', slotName: 'actions', width: 150 }
    ]
  : [
      { title: '资源', dataIndex: 'name', slotName: 'name' },
      { title: 'URL', dataIndex: 'url' },
      { title: '方法', dataIndex: 'requestMethod', width: 100 },
      { title: '匿名', dataIndex: 'isAnonymous', slotName: 'anonymous', width: 90 },
      { title: '操作', dataIndex: 'actions', slotName: 'actions', width: 150 }
    ])

const nodes = ref<PermissionNode[]>([])
const keywords = ref('')
const loading = ref(false)
const saving = ref(false)
const editorVisible = ref(false)
const errorMessage = ref('')
const form = reactive({ id: 0, name: '', path: '', component: '', icon: '', orderNum: 0, parentId: 0, isHidden: 0, resourceName: '', url: '', requestMethod: 'GET', isAnonymous: 0 })

const rows = computed(() => flatten(nodes.value, keywords.value.trim().toLowerCase()))
const parentOptions = computed(() => nodes.value.map((node) => ({ id: node.id, name: node.name || node.resourceName || '未命名' })))

onMounted(() => void load())

async function load(): Promise<void> {
  loading.value = true
  errorMessage.value = ''
  try {
    nodes.value = normalizeNodes(props.mode === 'menus' ? await listAdminMenus({ keywords: keywords.value.trim() }) : await listAdminResources({ keywords: keywords.value.trim() }))
  } catch (error) {
    errorMessage.value = apiErrorMessage(error, '权限数据加载失败')
    Message.error(errorMessage.value)
  } finally {
    loading.value = false
  }
}

function openEditor(node?: PermissionNode): void {
  form.id = Number(node?.id || 0)
  form.name = String(node?.name || '')
  form.path = String(node?.path || '')
  form.component = String(node?.component || '')
  form.icon = String(node?.icon || '')
  form.orderNum = Number(node?.orderNum || 0)
  form.parentId = Number(node?.parentId || 0)
  form.isHidden = Number(node?.isHidden || 0)
  form.resourceName = String(node?.resourceName || '')
  form.url = String(node?.url || '')
  form.requestMethod = String(node?.requestMethod || 'GET')
  form.isAnonymous = Number(node?.isAnonymous || 0)
  editorVisible.value = true
}

async function save(): Promise<void> {
  if (props.mode === 'menus') {
    if (!form.name.trim() || !form.path.trim() || !form.component.trim()) {
      Message.error('菜单名称、路径和组件不能为空')
      return
    }
  } else if (!form.resourceName.trim() || !form.url.trim() || !form.requestMethod) {
    Message.error('资源名称、URL 和请求方法不能为空')
    return
  }
  saving.value = true
  try {
    if (props.mode === 'menus') {
      await saveAdminMenu({ id: form.id || undefined, name: form.name.trim(), path: form.path.trim(), component: form.component.trim(), icon: form.icon.trim(), orderNum: form.orderNum, parentId: form.parentId, IsHidden: form.isHidden })
    } else {
      await saveAdminResource({ id: form.id || undefined, resourceName: form.resourceName.trim(), url: form.url.trim(), requestMethod: form.requestMethod, parentId: form.parentId, isAnonymous: form.isAnonymous })
    }
    editorVisible.value = false
    Message.success(`${title.value}已保存`)
    await load()
  } catch (error) {
    Message.error(apiErrorMessage(error, `${title.value}保存失败`))
  } finally {
    saving.value = false
  }
}

async function deleteItem(id: unknown): Promise<void> {
  const itemId = Number(id)
  if (!itemId) return
  try {
    if (props.mode === 'menus') await deleteAdminMenu(itemId)
    else await deleteAdminResource(itemId)
    Message.success(`${title.value}已删除`)
    await load()
  } catch (error) {
    Message.error(apiErrorMessage(error, `${title.value}删除失败`))
  }
}

async function toggleHidden(node: PermissionNode, value: boolean | string | number): Promise<void> {
  if (props.mode !== 'menus') return
  try {
    await updateAdminMenuHidden(node.id, Boolean(value) ? 1 : 0)
    node.isHidden = Boolean(value) ? 1 : 0
  } catch (error) {
    Message.error(apiErrorMessage(error, '菜单状态更新失败'))
  }
}

function normalizeNodes(value: unknown): PermissionNode[] {
  if (!Array.isArray(value)) return []
  return value.flatMap((item) => {
    if (!item || typeof item !== 'object') return []
    const source = item as Record<string, unknown>
    const id = Number(source.id)
    if (!Number.isFinite(id) || id <= 0) return []
    return [{ ...source, id, children: normalizeNodes(source.children) } as PermissionNode]
  })
}

function flatten(items: PermissionNode[], query: string, depth = 0): PermissionNode[] {
  const result: PermissionNode[] = []
  for (const item of items) {
    const name = props.mode === 'menus' ? item.name || '未命名菜单' : item.resourceName || '未命名资源'
    const searchable = [name, item.path, item.url, item.component].filter(Boolean).join(' ').toLowerCase()
    const children = flatten(item.children || [], query, depth + 1)
    if (!query || searchable.includes(query) || children.length > 0) result.push({ ...item, name, depth }, ...children)
  }
  return result
}
</script>
