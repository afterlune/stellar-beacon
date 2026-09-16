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
          {{ t('common.create') }}
        </a-button>
      </template>
    </AdminPageHeader>

    <a-card class="admin-panel" :bordered="false">
      <div class="admin-table-toolbar">
        <div class="admin-table-toolbar-main">
          <a-tag v-if="keywords.trim()" color="arcoblue">{{ t('rbac.shared.keywords', { keywords: keywords.trim() }) }}</a-tag>
          <a-button v-if="keywords.trim()" type="text" size="small" @click="clearKeywords">{{ t('rbac.shared.clearSearch') }}</a-button>
          <span class="admin-toolbar-caption">
            {{ mode === 'menus'
              ? t('rbac.tree.topLevelMenus', { count: nodes.length })
              : t('rbac.tree.topLevelResources', { count: nodes.length }) }}
            · {{ t('rbac.tree.totalRecords', { total: totalNodes }) }}
          </span>
        </div>
        <div class="admin-table-toolbar-actions">
          <a-button size="small" @click="toggleAll">{{ allExpanded ? t('rbac.shared.collapseAll') : t('rbac.shared.expandAll') }}</a-button>
          <a-button :loading="loading" size="small" @click="load">
            <template #icon><IconRefresh /></template>
            {{ t('common.refresh') }}
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
                :aria-label="isExpanded(record.id) ? t('rbac.tree.collapseNode', { name: record.name }) : t('rbac.tree.expandNode', { name: record.name })"
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
            <a-tooltip v-if="isUnknownComponent(record.component)" :content="t('rbac.tree.unknownComponent', { component: record.component })">
              <a-tag color="orange">{{ record.component }}</a-tag>
            </a-tooltip>
            <span v-else class="admin-mono-cell" :title="String(record.component || '')">{{ record.component || '—' }}</span>
          </template>
          <template #orderNum="{ record }">
            <span class="admin-num-cell">{{ record.orderNum ?? 0 }}</span>
          </template>
          <template #hidden="{ record }">
            <a-tooltip :content="Number(record.isHidden) === 1 ? t('rbac.tree.clickShow') : t('rbac.tree.clickHide')">
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
              <a-button type="text" size="small" @click="openEditor(record)">{{ t('common.edit') }}</a-button>
              <a-popconfirm :content="t('rbac.tree.deleteConfirm', { name: record.name })" @ok="deleteItem(record.id)">
                <a-button type="text" status="danger" size="small">{{ t('common.delete') }}</a-button>
              </a-popconfirm>
            </a-space>
          </template>
          <template #empty>
            <AdminEmptyState
              :icon="props.mode === 'menus' ? IconMenu : IconCode"
              :title="keywords.trim() ? t('rbac.tree.emptySearchTitle') : emptyTitle"
              :description="keywords.trim() ? t('rbac.shared.searchHint') : t('rbac.tree.emptyHint')">
              <a-button v-if="keywords.trim()" size="small" @click="clearKeywords">{{ t('rbac.shared.clearSearch') }}</a-button>
              <a-button v-else type="primary" size="small" @click="openEditor()">{{ t(props.mode === 'menus' ? 'rbac.tree.createMenu' : 'rbac.tree.createResource') }}</a-button>
            </AdminEmptyState>
          </template>
        </a-table>
      </div>
    </a-card>

    <a-modal
      v-model:visible="editorVisible"
      :title="editorTitle"
      :ok-loading="saving"
      :mask-closable="false"
      width="660px"
      @ok="save">
      <a-form :model="form" layout="vertical">
        <template v-if="mode === 'menus'">
          <div class="admin-form-grid">
            <a-form-item field="name" :label="t('rbac.tree.menuName')" required>
              <a-input v-model="form.name" maxlength="20" show-word-limit :placeholder="t('rbac.tree.menuNamePlaceholder')" />
            </a-form-item>
            <a-form-item field="orderNum" :label="t('common.sort')">
              <a-input-number v-model="form.orderNum" :min="0" :max="9999" :placeholder="t('rbac.tree.orderHint')" />
            </a-form-item>
          </div>
          <a-form-item field="path" :label="t('rbac.tree.path')" required>
            <a-input v-model="form.path" maxlength="100" :placeholder="t('rbac.tree.pathPlaceholder')" />
            <template #help>{{ t('rbac.tree.pathHelp') }}</template>
          </a-form-item>
          <a-form-item field="component" :label="t('rbac.tree.componentPath')" required>
            <a-input v-model="form.component" maxlength="100" :placeholder="t('rbac.tree.componentPlaceholder')" />
            <template #help>
              <span v-if="isUnknownComponent(form.component)" class="component-warning">
                {{ t('rbac.tree.componentUnknown') }}
              </span>
              <span v-else>{{ t('rbac.tree.componentHelp') }}</span>
            </template>
          </a-form-item>
          <a-form-item field="icon" :label="t('rbac.tree.icon')">
            <a-input v-model="form.icon" maxlength="50" :placeholder="t('rbac.tree.iconPlaceholder')" />
          </a-form-item>
        </template>

        <template v-else>
          <a-form-item field="resourceName" :label="t('rbac.tree.resourceName')" required>
            <a-input v-model="form.resourceName" maxlength="50" :placeholder="t('rbac.tree.resourceNamePlaceholder')" />
          </a-form-item>
          <a-form-item field="url" label="URL" required>
            <a-input v-model="form.url" maxlength="255" :placeholder="t('rbac.tree.urlPlaceholder')" />
          </a-form-item>
          <a-form-item field="requestMethod" :label="t('rbac.tree.requestMethod')" required>
            <a-select v-model="form.requestMethod">
              <a-option v-for="method in methods" :key="method" :value="method">{{ method }}</a-option>
            </a-select>
          </a-form-item>
        </template>

        <a-form-item field="parentId" :label="mode === 'menus' ? t('rbac.tree.parentMenu') : t('rbac.tree.parentResource')">
          <a-select v-model="form.parentId" :placeholder="t('rbac.tree.topLevel')">
            <a-option :value="0">{{ t(mode === 'menus' ? 'rbac.tree.topLevelMenu' : 'rbac.tree.topLevelResource') }}</a-option>
            <a-option v-for="parent in parentOptions" :key="parent.id" :value="parent.id">{{ parent.label }}</a-option>
          </a-select>
          <template #help>{{ t('rbac.tree.parentHelp') }}</template>
        </a-form-item>

        <a-form-item v-if="mode === 'menus'" :label="t('rbac.tree.hideInSidebar')">
          <a-switch v-model="form.isHidden" :checked-value="1" :unchecked-value="0" />
          <template #help>{{ t('rbac.tree.hideHelp') }}</template>
        </a-form-item>
        <a-form-item v-else :label="t('rbac.tree.allowAnonymous')">
          <a-switch v-model="form.isAnonymous" :checked-value="1" :unchecked-value="0" />
          <template #help>{{ t('rbac.tree.anonymousHelp') }}</template>
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
import { t } from '@/i18n'
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

const title = computed(() => t(props.mode === 'menus' ? 'rbac.tree.menusTitle' : 'rbac.tree.resourcesTitle'))
const description = computed(() => t(props.mode === 'menus'
  ? 'rbac.tree.menusDescription'
  : 'rbac.tree.resourcesDescription'))
const placeholder = computed(() => t(props.mode === 'menus' ? 'rbac.tree.searchMenus' : 'rbac.tree.searchResources'))
const emptyTitle = computed(() => t(props.mode === 'menus' ? 'rbac.tree.emptyMenus' : 'rbac.tree.emptyResources'))

const columns = computed(() => (props.mode === 'menus'
  ? [
      { title: t('rbac.tree.menu'), dataIndex: 'name', slotName: 'name', minWidth: 240 },
      { title: t('rbac.tree.path'), dataIndex: 'path', slotName: 'path', width: 180 },
      { title: t('rbac.tree.component'), dataIndex: 'component', slotName: 'component', width: 210 },
      { title: t('common.sort'), dataIndex: 'orderNum', slotName: 'orderNum', width: 80 },
      { title: t('rbac.tree.hidden'), dataIndex: 'isHidden', slotName: 'hidden', width: 84 },
      { title: t('common.actions'), dataIndex: 'actions', slotName: 'actions', width: 140 }
    ]
  : [
      { title: t('rbac.tree.resource'), dataIndex: 'name', slotName: 'name', minWidth: 240 },
      { title: 'URL', dataIndex: 'url', slotName: 'path', width: 250 },
      { title: t('rbac.tree.method'), dataIndex: 'requestMethod', slotName: 'method', width: 100 },
      { title: t('rbac.tree.access'), dataIndex: 'isAnonymous', slotName: 'anonymous', width: 100 },
      { title: t('common.actions'), dataIndex: 'actions', slotName: 'actions', width: 140 }
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

/** 新增/编辑弹窗标题随语言与模式变化，且要跟着表单里的 id 走。 */
const editorTitle = computed(() => {
  if (props.mode === 'menus') return t(form.id ? 'rbac.tree.editMenu' : 'rbac.tree.createMenu')
  return t(form.id ? 'rbac.tree.editResource' : 'rbac.tree.createResource')
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
      const label = item.name || item.resourceName || t('rbac.shared.unnamed')
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
    errorMessage.value = apiErrorMessage(error, t('rbac.tree.loadFailed'))
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
      Message.error(t('rbac.tree.menuRequired'))
      return
    }
  } else if (!form.resourceName.trim() || !form.url.trim() || !form.requestMethod) {
    Message.error(t('rbac.tree.resourceRequired'))
    return
  }
  if (form.id && form.parentId === form.id) {
    Message.error(t('rbac.tree.selfParent'))
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
    Message.success(t('rbac.tree.saved', { title: title.value }))
    await load()
  } catch (error) {
    Message.error(apiErrorMessage(error, t('rbac.tree.saveFailed', { title: title.value })))
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
    Message.success(t('rbac.tree.deleted', { title: title.value }))
    await load()
  } catch (error) {
    Message.error(apiErrorMessage(error, t('rbac.tree.deleteFailed', { title: title.value })))
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
    Message.success(next === 1 ? t('rbac.tree.menuHidden') : t('rbac.tree.menuShown'))
  } catch (error) {
    node.isHidden = previous
    Message.error(apiErrorMessage(error, t('rbac.tree.visibilityFailed')))
  } finally {
    pendingHiddenId.value = 0
  }
}

function flatten(items: PermissionNode[], depth: number, forceExpand: boolean): TreeRow[] {
  const result: TreeRow[] = []
  for (const item of items) {
    const children = Array.isArray(item.children) ? item.children : []
    const name = props.mode === 'menus'
      ? item.name || t('rbac.tree.unnamedMenu')
      : item.resourceName || item.name || t('rbac.tree.unnamedResource')
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
