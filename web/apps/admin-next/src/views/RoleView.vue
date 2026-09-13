<template>
  <section class="admin-page">
    <AdminPageHeader title="角色管理" description="管理角色，并配置菜单和接口权限。">
      <template #actions>
        <a-input-search
          v-model="keywords"
          class="admin-filter-input"
          placeholder="搜索角色名"
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
          <span class="admin-toolbar-caption">共 {{ total }} 个角色</span>
          <a-button :loading="loading" size="small" @click="load">
            <template #icon><IconRefresh /></template>
            刷新
          </a-button>
        </div>
      </div>

      <AdminErrorState v-if="errorMessage" :error="errorMessage" title="角色列表加载失败" @retry="load" />

      <AdminBatchBar :count="selectedIds.length" :hint="`本页 ${roles.length} 个`" @clear="clearSelection">
        <a-button size="small" status="danger" :loading="batchDeleting" @click="batchDelete">批量删除</a-button>
      </AdminBatchBar>

      <div class="admin-table-shell">
        <a-table
          v-model:selected-keys="selectedKeys"
          :row-selection="{ type: 'checkbox', showCheckedAll: true, onlyCurrent: true }"
          :data="roles"
          :columns="columns"
          :loading="loading"
          :pagination="pagination"
          row-key="id"
          @page-change="changePage"
          @page-size-change="changePageSize">
          <template #id="{ record }"><span class="admin-id-cell">#{{ record.id }}</span></template>
          <template #roleName="{ record }">
            <span class="admin-title-cell">{{ record.roleName || '未命名角色' }}</span>
          </template>
          <template #status="{ record }">
            <AdminStatusTag :kind="Number(record.isDisable) === 1 ? 'disabled' : 'enabled'" />
          </template>
          <template #permissions="{ record }">
            <a-space :size="4" wrap>
              <a-tag>{{ countOf(record.menuIds) }} 菜单</a-tag>
              <a-tag>{{ countOf(record.resourceIds) }} 接口</a-tag>
            </a-space>
          </template>
          <template #time="{ record }"><span class="admin-cell-nowrap">{{ formatDateTime(record.createTime) }}</span></template>
          <template #actions="{ record }">
            <a-space class="admin-action-space">
              <a-button type="text" size="small" @click="openEditor(record)">编辑权限</a-button>
              <a-popconfirm
                :content="`删除角色「${record.roleName}」后，拥有该角色的账号会立即失去对应权限，确认删除吗？`"
                @ok="deleteRole(record.id)">
                <a-button type="text" status="danger" size="small">删除</a-button>
              </a-popconfirm>
            </a-space>
          </template>
          <template #empty>
            <AdminEmptyState
              :icon="IconLock"
              :title="keywords.trim() ? '没有匹配的角色' : '还没有角色'"
              :description="keywords.trim() ? '换个关键词再试一次。' : '创建角色并分配菜单与接口权限，再把它授予用户。'">
              <a-button v-if="keywords.trim()" size="small" @click="clearKeywords">清空搜索</a-button>
              <a-button v-else type="primary" size="small" @click="openEditor()">新增角色</a-button>
            </AdminEmptyState>
          </template>
        </a-table>
      </div>
    </a-card>

    <a-modal
      v-model:visible="editorVisible"
      :title="editor.id ? '编辑角色' : '新增角色'"
      :ok-loading="saving"
      :mask-closable="false"
      width="760px"
      @ok="saveEditor">
      <a-form :model="editor" layout="vertical">
        <a-form-item field="roleName" label="角色名" required>
          <a-input v-model="editor.roleName" maxlength="20" show-word-limit placeholder="例如：内容编辑" />
          <template #help>角色名用于标识一组权限，建议使用岗位或职责命名。</template>
        </a-form-item>

        <a-form-item label="菜单权限">
          <div class="permission-panel">
            <div class="permission-panel-head">
              <span>已选 {{ editor.menuIds.length }} 项</span>
              <a-space :size="4">
                <a-button type="text" size="mini" :disabled="menuOptions.length === 0" @click="selectAll('menu')">全选</a-button>
                <a-button type="text" size="mini" :disabled="editor.menuIds.length === 0" @click="clearAll('menu')">清空</a-button>
              </a-space>
            </div>
            <div v-if="menuOptions.length" class="permission-groups">
              <div v-for="group in menuOptions" :key="`menu-group-${group.id}`" class="permission-group-block">
                <a-checkbox
                  :model-value="isGroupChecked('menu', group)"
                  :indeterminate="isGroupIndeterminate('menu', group)"
                  @change="(checked) => toggleGroup('menu', group, Boolean(checked))">
                  {{ group.label }}
                </a-checkbox>
                <div v-if="group.children?.length" class="permission-children">
                  <a-checkbox
                    v-for="child in group.children"
                    :key="`menu-${child.id}`"
                    :model-value="editor.menuIds.includes(child.id)"
                    @change="(checked) => toggleId('menu', child.id, Boolean(checked))">
                    {{ child.label }}
                  </a-checkbox>
                </div>
              </div>
            </div>
            <a-empty v-else description="暂无菜单权限选项" />
          </div>
        </a-form-item>

        <a-form-item label="接口权限">
          <div class="permission-panel">
            <div class="permission-panel-head">
              <span>已选 {{ editor.resourceIds.length }} 项</span>
              <a-space :size="4">
                <a-button type="text" size="mini" :disabled="resourceOptions.length === 0" @click="selectAll('resource')">全选</a-button>
                <a-button type="text" size="mini" :disabled="editor.resourceIds.length === 0" @click="clearAll('resource')">清空</a-button>
              </a-space>
            </div>
            <div v-if="resourceOptions.length" class="permission-groups">
              <div v-for="group in resourceOptions" :key="`resource-group-${group.id}`" class="permission-group-block">
                <a-checkbox
                  :model-value="isGroupChecked('resource', group)"
                  :indeterminate="isGroupIndeterminate('resource', group)"
                  @change="(checked) => toggleGroup('resource', group, Boolean(checked))">
                  {{ group.label }}
                </a-checkbox>
                <div v-if="group.children?.length" class="permission-children">
                  <a-checkbox
                    v-for="child in group.children"
                    :key="`resource-${child.id}`"
                    :model-value="editor.resourceIds.includes(child.id)"
                    @change="(checked) => toggleId('resource', child.id, Boolean(checked))">
                    {{ child.label }}
                  </a-checkbox>
                </div>
              </div>
            </div>
            <a-empty v-else description="暂无接口权限选项" />
          </div>
        </a-form-item>
      </a-form>
    </a-modal>
  </section>
</template>

<script setup lang="ts">
import { computed, onMounted, reactive, ref } from 'vue'
import { Message, Modal } from '@arco-design/web-vue'
import { IconLock, IconPlus, IconRefresh } from '@arco-design/web-vue/es/icon'

import {
  apiErrorMessage,
  deleteAdminRoles,
  listAdminRoles,
  listRoleMenus,
  listRoleResources,
  saveAdminRole
} from '@/api/http'
import AdminBatchBar from '@/components/AdminBatchBar.vue'
import AdminEmptyState from '@/components/AdminEmptyState.vue'
import AdminErrorState from '@/components/AdminErrorState.vue'
import AdminPageHeader from '@/components/AdminPageHeader.vue'
import AdminStatusTag from '@/components/AdminStatusTag.vue'
import { useAsyncList } from '@/composables/useAsyncList'
import { useQueryFilters } from '@/composables/useQueryFilters'
import { readStoredPageSize, useStoredPageSize } from '@/composables/useTablePrefs'
import { formatDateTime } from '@/utils/format'
import { tablePagination } from '@/utils/pagination'
import type { AdminRole } from '@stellar-beacon/api-contract'

interface PermissionOption {
  id: number
  label: string
  children: PermissionOption[]
}

type Scope = 'menu' | 'resource'

const VIEW_KEY = 'roles'

const columns = [
  { title: 'ID', dataIndex: 'id', width: 84, slotName: 'id' },
  { title: '角色名', dataIndex: 'roleName', slotName: 'roleName', minWidth: 180 },
  { title: '状态', dataIndex: 'isDisable', width: 100, slotName: 'status' },
  { title: '权限范围', dataIndex: 'permissions', width: 180, slotName: 'permissions' },
  { title: '创建时间', dataIndex: 'createTime', width: 180, slotName: 'time' },
  { title: '操作', dataIndex: 'actions', width: 176, slotName: 'actions' }
]

const menuOptions = ref<PermissionOption[]>([])
const resourceOptions = ref<PermissionOption[]>([])
const keywords = ref('')
const saving = ref(false)
const selectedKeys = ref<number[]>([])
const batchDeleting = ref(false)
const editorVisible = ref(false)
const editor = reactive({ id: 0, roleName: '', menuIds: [] as number[], resourceIds: [] as number[] })

const {
  items: roles,
  total,
  current,
  pageSize,
  loading,
  error: errorMessage,
  load,
  reload,
  changePage: gotoPage,
  changePageSize: applyPageSize
} = useAsyncList<AdminRole>(
  ({ current: page, pageSize: size, signal }) => listAdminRoles({
    current: page,
    size,
    keywords: keywords.value.trim()
  }, { signal }),
  { pageSize: readStoredPageSize(VIEW_KEY), fallbackMessage: '角色列表加载失败' }
)

useStoredPageSize(VIEW_KEY, pageSize)
useQueryFilters([
  { key: 'keywords', ref: keywords, debounce: true },
  { key: 'page', ref: current }
], { onRestore: () => void load(), onSearch: () => void reload() })

const pagination = computed(() => tablePagination(current.value, pageSize.value, total.value))
const selectedIds = computed(() =>
  [...new Set(selectedKeys.value.map(Number).filter((id) => Number.isInteger(id) && id > 0))]
)

onMounted(() => {
  void loadPermissionOptions()
})

function clearKeywords(): void {
  keywords.value = ''
  void reload()
}

async function loadPermissionOptions(): Promise<void> {
  try {
    const [menus, resources] = await Promise.all([listRoleMenus(), listRoleResources()])
    menuOptions.value = normalizeOptions(menus)
    resourceOptions.value = normalizeOptions(resources)
  } catch (error) {
    Message.error(apiErrorMessage(error, '权限选项加载失败'))
  }
}

function changePage(page: number): void {
  clearSelection()
  gotoPage(page)
}

function changePageSize(size: number): void {
  clearSelection()
  applyPageSize(size)
}

function clearSelection(): void {
  selectedKeys.value = []
}

function batchDelete(): void {
  const ids = selectedIds.value
  if (ids.length === 0) return
  Modal.confirm({
    title: '批量删除',
    content: `删除选中的 ${ids.length} 个角色后，拥有这些角色的账号会立即失去对应权限，确定继续吗？`,
    okText: '批量删除',
    cancelText: '取消',
    okButtonProps: { status: 'danger' },
    onOk: async () => {
      batchDeleting.value = true
      try {
        await deleteAdminRoles(ids)
        Message.success(`已删除 ${ids.length} 个角色`)
        clearSelection()
        await load()
      } catch (error) {
        Message.error(apiErrorMessage(error, '批量删除失败'))
      } finally {
        batchDeleting.value = false
      }
    }
  })
}

function openEditor(role?: AdminRole): void {
  editor.id = Number(role?.id || 0)
  editor.roleName = String(role?.roleName || '')
  editor.menuIds = idsFrom(role?.menuIds)
  editor.resourceIds = idsFrom(role?.resourceIds)
  editorVisible.value = true
}

async function saveEditor(): Promise<void> {
  if (!editor.roleName.trim()) {
    Message.error('角色名不能为空')
    return
  }
  saving.value = true
  try {
    await saveAdminRole({
      id: editor.id || undefined,
      roleName: editor.roleName.trim(),
      menuIds: editor.menuIds,
      resourceIds: editor.resourceIds
    })
    editorVisible.value = false
    Message.success(editor.id ? '角色已更新' : '角色已创建')
    await load()
  } catch (error) {
    Message.error(apiErrorMessage(error, '角色保存失败'))
  } finally {
    saving.value = false
  }
}

async function deleteRole(id: unknown): Promise<void> {
  const roleId = Number(id)
  if (!Number.isInteger(roleId) || roleId <= 0) return
  try {
    await deleteAdminRoles([roleId])
    if (roles.value.length === 1 && current.value > 1) current.value -= 1
    Message.success('角色已删除')
    await load()
  } catch (error) {
    Message.error(apiErrorMessage(error, '角色删除失败'))
  }
}

function selectionOf(scope: Scope): number[] {
  return scope === 'menu' ? editor.menuIds : editor.resourceIds
}

function setSelection(scope: Scope, value: number[]): void {
  if (scope === 'menu') editor.menuIds = value
  else editor.resourceIds = value
}

function groupIds(group: PermissionOption): number[] {
  return [group.id, ...(group.children || []).map((child) => child.id)]
}

function isGroupChecked(scope: Scope, group: PermissionOption): boolean {
  const selection = selectionOf(scope)
  const ids = groupIds(group)
  return ids.length > 0 && ids.every((id) => selection.includes(id))
}

function isGroupIndeterminate(scope: Scope, group: PermissionOption): boolean {
  const selection = selectionOf(scope)
  return groupIds(group).some((id) => selection.includes(id)) && !isGroupChecked(scope, group)
}

function toggleGroup(scope: Scope, group: PermissionOption, checked: boolean): void {
  const ids = groupIds(group)
  const selection = new Set(selectionOf(scope))
  for (const id of ids) {
    if (checked) selection.add(id)
    else selection.delete(id)
  }
  setSelection(scope, [...selection])
}

function toggleId(scope: Scope, id: number, checked: boolean): void {
  const selection = new Set(selectionOf(scope))
  if (checked) selection.add(id)
  else selection.delete(id)
  setSelection(scope, [...selection])
}

function selectAll(scope: Scope): void {
  const groups = scope === 'menu' ? menuOptions.value : resourceOptions.value
  setSelection(scope, [...new Set(groups.flatMap(groupIds))])
}

function clearAll(scope: Scope): void {
  setSelection(scope, [])
}

function countOf(value: unknown): number {
  return Array.isArray(value) ? value.length : 0
}

function normalizeOptions(value: unknown): PermissionOption[] {
  if (!Array.isArray(value)) return []
  return value.flatMap((item) => {
    if (!item || typeof item !== 'object') return []
    const source = item as Record<string, unknown>
    const id = Number(source.id)
    if (!Number.isFinite(id) || id <= 0) return []
    return [{
      id,
      label: String(source.label || source.name || '未命名权限'),
      children: normalizeOptions(source.children)
    }]
  })
}

function idsFrom(value: unknown): number[] {
  return Array.isArray(value)
    ? value.map(Number).filter((id) => Number.isFinite(id) && id > 0)
    : []
}
</script>

<style scoped>
.permission-panel {
  overflow: hidden;
  border: 1px solid var(--admin-border);
  border-radius: var(--admin-radius-control);
  background: var(--admin-surface-soft);
}

.permission-panel-head {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
  padding: 6px 12px;
  border-bottom: 1px solid var(--admin-border);
  color: var(--admin-muted);
  background: var(--admin-surface);
  font-size: 12px;
  font-weight: 650;
}

.permission-groups {
  max-height: 240px;
  overflow: auto;
  padding: 10px 12px;
  display: grid;
  gap: 10px;
}

.permission-group-block {
  display: grid;
  gap: 6px;
}

.permission-group-block > .arco-checkbox {
  font-weight: 650;
}

.permission-children {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(150px, 1fr));
  gap: 4px 14px;
  padding-left: 22px;
}
</style>
