<template>
  <section class="admin-page">
    <AdminPageHeader :title="title" :description="description">
      <template #actions>
        <a-input-search
          v-model="keywords"
          class="admin-filter-input"
          :placeholder="placeholder"
          allow-clear
          @search="load" />
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
          <span class="admin-toolbar-caption">
            {{ nodes.length }} 个{{ mode === 'menus' ? '顶级菜单' : '顶级资源组' }} · 共 {{ totalNodes }} 条记录
          </span>
        </div>
        <div class="admin-table-toolbar-actions">
          <a-button size="small" @click="toggleAll">{{ allExpanded ? '全部折叠' : '全部展开' }}</a-button>
          <a-button :loading="loading" size="small" @click="load">
            <template #icon><IconRefresh /></template>
            刷新
          </a-button>
        </div>
      </div>

      <a-alert v-if="errorMessage" type="error" closable @close="errorMessage = ''">{{ errorMessage }}</a-alert>

      <div class="admin-table-shell">
        <a-table :data="rows" :columns="columns" :loading="loading" :pagination="false" row-key="id">
          <template #name="{ record }">
            <span class="admin-tree-branch" :style="{ paddingLeft: `${record.depth * 22}px` }">
              <span v-if="record.depth > 0" class="admin-tree-indent" aria-hidden="true" />
              <button
                v-if="record.hasChildren"
                class="admin-tree-toggle"
                type="button"
                :aria-expanded="isExpanded(record.id)"
                :aria-label="isExpanded(record.id) ? `折叠 ${record.name}` : `展开 ${record.name}`"
                @click="toggle(record.id)">
                <IconCaretDown v-if="isExpanded(record.id)" />
                <IconCaretRight v-else />
              </button>
              <span v-else class="admin-tree-toggle-spacer" aria-hidden="true" />
              <span class="admin-tree-name">{{ record.name }}</span>
            </span>
          </template>
          <template #path="{ record }">
            <span class="admin-mono-cell" :title="record.path">{{ record.path || '—' }}</span>
          </template>
          <template #component="{ record }">
            <a-tooltip v-if="isUnknownComponent(record.component)" :content="`未注册的组件会回退到占位页：${record.component}`">
              <a-tag color="orange">{{ record.component }}</a-tag>
            </a-tooltip>
            <span v-else class="admin-mono-cell" :title="String(record.component || '')">{{ record.component || '—' }}</span>
          </template>
          <template #orderNum="{ record }">
            <span class="admin-num-cell">{{ record.orderNum ?? 0 }}</span>
          </template>
          <template #hidden="{ record }">
            <a-tooltip :content="Number(record.isHidden) === 1 ? '点击在侧边栏显示' : '点击在侧边栏隐藏'">
              <a-switch
                :model-value="Number(record.isHidden) === 1"
                :loading="pendingHiddenId === record.id"
                @change="(value) => toggleHidden(record, value)" />
            </a-tooltip>
          </template>
          <template #method="{ record }">
            <a-tag :color="methodColor(record.requestMethod)">{{ record.requestMethod || '—' }}</a-tag>
          </template>
          <template #anonymous="{ record }">
            <AdminStatusTag :kind="Number(record.isAnonymous) === 1 ? 'anonymous' : 'authenticated'" />
          </template>
          <template #actions="{ record }">
            <a-space class="admin-action-space">
              <a-button type="text" size="small" @click="openEditor(record)">编辑</a-button>
              <a-popconfirm :content="`确定删除${record.name}吗？此操作不可撤销。`" @ok="deleteItem(record.id)">
                <a-button type="text" status="danger" size="small">删除</a-button>
              </a-popconfirm>
            </a-space>
          </template>
          <template #empty>
            <AdminEmptyState
              :icon="props.mode === 'menus' ? IconMenu : IconCode"
              :title="keywords.trim() ? '没有匹配的记录' : `暂无${title}`"
              :description="keywords.trim() ? '换个关键词再试一次。' : `创建第一条记录，它会立即出现在这里。`">
              <a-button v-if="keywords.trim()" size="small" @click="clearKeywords">清空搜索</a-button>
              <a-button v-else type="primary" size="small" @click="openEditor()">新增{{ mode === 'menus' ? '菜单' : '资源' }}</a-button>
            </AdminEmptyState>
          </template>
        </a-table>
      </div>
    </a-card>

    <a-modal
      v-model:visible="editorVisible"
      :title="form.id ? `编辑${title}` : `新增${title}`"
      :ok-loading="saving"
      :mask-closable="false"
      width="660px"
      @ok="save">
      <a-form :model="form" layout="vertical">
        <template v-if="mode === 'menus'">
          <div class="admin-form-grid">
            <a-form-item field="name" label="菜单名称" required>
              <a-input v-model="form.name" maxlength="20" show-word-limit placeholder="例如：文章列表" />
            </a-form-item>
            <a-form-item field="orderNum" label="排序">
              <a-input-number v-model="form.orderNum" :min="0" :max="9999" placeholder="数字越小越靠前" />
            </a-form-item>
          </div>
          <a-form-item field="path" label="路径" required>
            <a-input v-model="form.path" maxlength="100" placeholder="例如 /article-list" />
            <template #help>路径必须唯一，并与前端路由保持一致。</template>
          </a-form-item>
          <a-form-item field="component" label="组件路径" required>
            <a-input v-model="form.component" maxlength="100" placeholder="例如 /article/ArticleList.vue" />
            <template #help>
              <span v-if="isUnknownComponent(form.component)" class="component-warning">
                该组件未在前端注册，页面会回退到占位视图。
              </span>
              <span v-else>使用前端已注册的视图文件路径，例如 /article/ArticleList.vue。</span>
            </template>
          </a-form-item>
          <a-form-item field="icon" label="图标">
            <a-input v-model="form.icon" maxlength="50" placeholder="可选，沿用后端图标标识" />
          </a-form-item>
        </template>

        <template v-else>
          <a-form-item field="resourceName" label="资源名称" required>
            <a-input v-model="form.resourceName" maxlength="50" placeholder="例如：文章读取" />
          </a-form-item>
          <a-form-item field="url" label="URL" required>
            <a-input v-model="form.url" maxlength="255" placeholder="例如 /admin/articles" />
          </a-form-item>
          <a-form-item field="requestMethod" label="请求方法" required>
            <a-select v-model="form.requestMethod">
              <a-option v-for="method in methods" :key="method" :value="method">{{ method }}</a-option>
            </a-select>
          </a-form-item>
        </template>

        <a-form-item field="parentId" :label="mode === 'menus' ? '父菜单' : '父资源'">
          <a-select v-model="form.parentId" placeholder="顶级">
            <a-option :value="0">顶级{{ mode === 'menus' ? '菜单' : '资源' }}</a-option>
            <a-option v-for="parent in parentOptions" :key="parent.id" :value="parent.id">{{ parent.label }}</a-option>
          </a-select>
          <template #help>不能选择自己或自己的子节点，否则会形成循环层级。</template>
        </a-form-item>

        <a-form-item v-if="mode === 'menus'" label="在侧边栏隐藏">
          <a-switch v-model="form.isHidden" :checked-value="1" :unchecked-value="0" />
          <template #help>隐藏后菜单仍然可访问，但不会出现在导航中。</template>
        </a-form-item>
        <a-form-item v-else label="允许匿名访问">
          <a-switch v-model="form.isAnonymous" :checked-value="1" :unchecked-value="0" />
          <template #help>开启后，未登录用户也可以调用该接口。</template>
        </a-form-item>
      </a-form>
    </a-modal>
  </section>
</template>

<script setup lang="ts">
import { computed, onMounted, reactive, ref } from 'vue'
import { Message } from '@arco-design/web-vue'
import {
  IconCaretDown,
  IconCaretRight,
  IconCode,
  IconMenu,
  IconPlus,
  IconRefresh
} from '@arco-design/web-vue/es/icon'

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
import AdminEmptyState from '@/components/AdminEmptyState.vue'
import AdminPageHeader from '@/components/AdminPageHeader.vue'
import AdminStatusTag from '@/components/AdminStatusTag.vue'
import { isRegisteredComponent } from '@/router/menu'

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

interface TreeRow extends PermissionNode {
  name: string
  depth: number
  hasChildren: boolean
}

const props = defineProps<{ mode: PermissionMode }>()

const methods = ['GET', 'POST', 'PUT', 'DELETE', 'PATCH']

const title = computed(() => (props.mode === 'menus' ? '菜单管理' : '接口资源管理'))
const description = computed(() => (props.mode === 'menus'
  ? '维护页面入口、层级和可见性，让每个角色只看到该看到的入口。'
  : '查看后台接口资源与访问边界，明确哪些接口允许匿名调用。'))
const placeholder = computed(() => (props.mode === 'menus' ? '搜索菜单名或路径' : '搜索资源名或 URL'))

const columns = computed(() => (props.mode === 'menus'
  ? [
      { title: '菜单', dataIndex: 'name', slotName: 'name', minWidth: 240 },
      { title: '路径', dataIndex: 'path', slotName: 'path', width: 180 },
      { title: '组件', dataIndex: 'component', slotName: 'component', width: 210 },
      { title: '排序', dataIndex: 'orderNum', slotName: 'orderNum', width: 80 },
      { title: '隐藏', dataIndex: 'isHidden', slotName: 'hidden', width: 84 },
      { title: '操作', dataIndex: 'actions', slotName: 'actions', width: 140 }
    ]
  : [
      { title: '资源', dataIndex: 'name', slotName: 'name', minWidth: 240 },
      { title: 'URL', dataIndex: 'url', slotName: 'path', width: 250 },
      { title: '方法', dataIndex: 'requestMethod', slotName: 'method', width: 100 },
      { title: '访问', dataIndex: 'isAnonymous', slotName: 'anonymous', width: 100 },
      { title: '操作', dataIndex: 'actions', slotName: 'actions', width: 140 }
    ]))

const nodes = ref<PermissionNode[]>([])
const expanded = ref<Set<number>>(new Set())
const keywords = ref('')
const loading = ref(false)
const saving = ref(false)
const pendingHiddenId = ref(0)
const editorVisible = ref(false)
const errorMessage = ref('')
const form = reactive({
  id: 0,
  name: '',
  path: '',
  component: '',
  icon: '',
  orderNum: 0,
  parentId: 0,
  isHidden: 0,
  resourceName: '',
  url: '',
  requestMethod: 'GET',
  isAnonymous: 0
})

const totalNodes = computed(() => countNodes(nodes.value))
const allExpanded = computed(() => nodes.value.length > 0 && nodes.value.every((node) => expanded.value.has(node.id)))

/**
 * Flatten the tree honouring the expand state. When a search query is present
 * every ancestor of a match is force-expanded so results are always visible.
 */
const rows = computed<TreeRow[]>(() => flatten(nodes.value, 0, Boolean(keywords.value.trim())))

/** Exclude the edited node and its descendants so a cycle cannot be created. */
const parentOptions = computed(() => {
  const excluded = new Set<number>(form.id ? [form.id, ...descendantIds(form.id)] : [])
  const options: Array<{ id: number; label: string }> = []
  const walk = (items: PermissionNode[], depth: number): void => {
    for (const item of items) {
      if (excluded.has(item.id)) continue
      const label = item.name || item.resourceName || '未命名'
      options.push({ id: item.id, label: `${'　'.repeat(depth)}${depth > 0 ? '└ ' : ''}${label}` })
      walk(item.children || [], depth + 1)
    }
  }
  walk(nodes.value, 0)
  return options
})

onMounted(() => void load())

function isExpanded(id: number): boolean {
  return expanded.value.has(id)
}

function toggle(id: number): void {
  const next = new Set(expanded.value)
  if (next.has(id)) next.delete(id)
  else next.add(id)
  expanded.value = next
}

function toggleAll(): void {
  if (allExpanded.value) {
    expanded.value = new Set()
    return
  }
  expanded.value = new Set(collectIds(nodes.value))
}

async function load(): Promise<void> {
  loading.value = true
  errorMessage.value = ''
  try {
    const raw = props.mode === 'menus'
      ? await listAdminMenus({ keywords: keywords.value.trim() })
      : await listAdminResources({ keywords: keywords.value.trim() })
    nodes.value = normalizeNodes(raw)
    // Default to fully expanded: the tree is small and seeing the hierarchy helps.
    if (expanded.value.size === 0) expanded.value = new Set(collectIds(nodes.value))
  } catch (error) {
    errorMessage.value = apiErrorMessage(error, '权限数据加载失败')
    Message.error(errorMessage.value)
  } finally {
    loading.value = false
  }
}

function clearKeywords(): void {
  keywords.value = ''
  void load()
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
      Message.error('菜单名称、路径和组件路径不能为空')
      return
    }
  } else if (!form.resourceName.trim() || !form.url.trim() || !form.requestMethod) {
    Message.error('资源名称、URL 和请求方法不能为空')
    return
  }
  if (form.id && form.parentId === form.id) {
    Message.error('不能把记录设为自己的父节点')
    return
  }
  saving.value = true
  try {
    if (props.mode === 'menus') {
      await saveAdminMenu({
        id: form.id || undefined,
        name: form.name.trim(),
        path: form.path.trim(),
        component: form.component.trim(),
        icon: form.icon.trim(),
        orderNum: form.orderNum,
        parentId: form.parentId,
        isHidden: form.isHidden
      })
    } else {
      await saveAdminResource({
        id: form.id || undefined,
        resourceName: form.resourceName.trim(),
        url: form.url.trim(),
        requestMethod: form.requestMethod,
        parentId: form.parentId,
        isAnonymous: form.isAnonymous
      })
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
  if (!Number.isInteger(itemId) || itemId <= 0) return
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
  const next = Boolean(value) ? 1 : 0
  const previous = Number(node.isHidden) === 1 ? 1 : 0
  node.isHidden = next
  pendingHiddenId.value = node.id
  try {
    await updateAdminMenuHidden(node.id, next)
    Message.success(next === 1 ? '菜单已在侧边栏隐藏' : '菜单已在侧边栏显示')
  } catch (error) {
    node.isHidden = previous
    Message.error(apiErrorMessage(error, '菜单可见性更新失败'))
  } finally {
    pendingHiddenId.value = 0
  }
}

function flatten(items: PermissionNode[], depth: number, forceExpand: boolean): TreeRow[] {
  const result: TreeRow[] = []
  for (const item of items) {
    const children = Array.isArray(item.children) ? item.children : []
    const name = props.mode === 'menus'
      ? item.name || '未命名菜单'
      : item.resourceName || item.name || '未命名资源'
    result.push({ ...item, name, depth, hasChildren: children.length > 0 })
    if (children.length > 0 && (forceExpand || isExpanded(item.id))) {
      result.push(...flatten(children, depth + 1, forceExpand))
    }
  }
  return result
}

function descendantIds(id: number): number[] {
  const found = findNode(nodes.value, id)
  return found ? collectIds(found.children || []) : []
}

function findNode(items: PermissionNode[], id: number): PermissionNode | null {
  for (const item of items) {
    if (item.id === id) return item
    const nested = findNode(item.children || [], id)
    if (nested) return nested
  }
  return null
}

function collectIds(items: PermissionNode[]): number[] {
  return items.flatMap((item) => [item.id, ...collectIds(item.children || [])])
}

function countNodes(items: PermissionNode[]): number {
  return items.reduce((total, item) => total + 1 + countNodes(item.children || []), 0)
}

function isUnknownComponent(component: unknown): boolean {
  const value = String(component || '').trim()
  return value.length > 0 && value !== 'Layout' && !isRegisteredComponent(value)
}

function methodColor(method: unknown): string {
  switch (String(method || '').toUpperCase()) {
    case 'GET': return 'green'
    case 'POST': return 'arcoblue'
    case 'PUT': return 'orange'
    case 'DELETE': return 'red'
    case 'PATCH': return 'purple'
    default: return 'gray'
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
</script>

<style scoped>
.admin-tree-indent {
  width: 1px;
  height: 16px;
  flex: 0 0 auto;
  margin-right: 9px;
  background: var(--admin-border);
}

.component-warning {
  color: var(--admin-warm);
  font-weight: 600;
}
</style>
