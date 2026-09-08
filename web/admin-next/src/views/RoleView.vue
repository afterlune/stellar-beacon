<template>
  <section class="admin-page">
    <AdminPageHeader title="角色管理" description="用清晰的角色边界保护内容与后台操作。">
      <template #actions>
        <a-space>
          <a-input-search v-model="keywords" class="admin-filter-input" placeholder="搜索角色名" allow-clear @search="reload" />
          <a-button type="primary" @click="openEditor()">
            <template #icon><IconPlus /></template>
            新增
          </a-button>
        </a-space>
      </template>
    </AdminPageHeader>
    <a-card class="admin-panel" :bordered="false">
      <a-alert v-if="errorMessage" type="error" closable @close="errorMessage = ''">{{ errorMessage }}</a-alert>
      <div class="admin-table-shell">
        <a-table
          :data="roles"
          :columns="columns"
          :loading="loading"
          :pagination="pagination"
          row-key="id"
          @page-change="changePage"
          @page-size-change="changePageSize">
          <template #status="{ record }"><a-tag class="admin-status-tag" :color="Number(record.isDisable) === 1 ? 'orange' : 'green'">{{ Number(record.isDisable) === 1 ? '禁用' : '启用' }}</a-tag></template>
          <template #time="{ record }">{{ formatCell(record.createTime) }}</template>
          <template #actions="{ record }">
            <a-space class="admin-action-space">
              <a-button type="text" size="small" @click="openEditor(record)">编辑权限</a-button>
              <a-popconfirm content="确定删除该角色吗？" @ok="deleteRole(record.id)">
                <a-button type="text" status="danger" size="small">删除</a-button>
              </a-popconfirm>
            </a-space>
          </template>
          <template #empty><div class="admin-table-empty"><a-empty description="暂无角色" /></div></template>
        </a-table>
      </div>
    </a-card>

    <a-modal v-model:visible="editorVisible" :title="editor.id ? '编辑角色' : '新增角色'" :ok-loading="saving" width="720px" @ok="saveEditor">
      <a-form :model="editor" layout="vertical">
        <a-form-item field="roleName" label="角色名" required>
          <a-input v-model="editor.roleName" maxlength="20" show-word-limit />
        </a-form-item>
        <a-form-item label="菜单权限">
          <a-checkbox-group v-model="editor.menuIds" class="permission-grid">
            <a-checkbox v-for="option in menuChoices" :key="`menu-${option.id}`" :value="option.id">{{ option.label }}</a-checkbox>
          </a-checkbox-group>
          <a-empty v-if="menuChoices.length === 0" description="暂无菜单权限选项" />
        </a-form-item>
        <a-form-item label="资源权限">
          <a-checkbox-group v-model="editor.resourceIds" class="permission-grid">
            <a-checkbox v-for="option in resourceChoices" :key="`resource-${option.id}`" :value="option.id">{{ option.label }}</a-checkbox>
          </a-checkbox-group>
          <a-empty v-if="resourceChoices.length === 0" description="暂无资源权限选项" />
        </a-form-item>
      </a-form>
    </a-modal>
  </section>
</template>

<script setup lang="ts">
import { computed, onMounted, reactive, ref } from 'vue'
import { Message } from '@arco-design/web-vue'
import { IconPlus } from '@arco-design/web-vue/es/icon'

import {
  apiErrorMessage,
  deleteAdminRoles,
  listAdminRoles,
  listRoleMenus,
  listRoleResources,
  saveAdminRole
} from '@/api/http'
import AdminPageHeader from '@/components/AdminPageHeader.vue'
import { formatCell } from '@/utils/format'
import { tablePagination } from '@/utils/pagination'
import type { AdminRole } from '@shared/api-contract'

interface PermissionOption {
  id: number
  label: string
  children?: PermissionOption[]
}

const columns = [
  { title: 'ID', dataIndex: 'id', width: 90 },
  { title: '角色名', dataIndex: 'roleName' },
  { title: '状态', dataIndex: 'isDisable', width: 90, slotName: 'status' },
  { title: '创建时间', dataIndex: 'createTime', width: 200, slotName: 'time' },
  { title: '操作', dataIndex: 'actions', width: 170, slotName: 'actions' }
]

const roles = ref<AdminRole[]>([])
const menuOptions = ref<PermissionOption[]>([])
const resourceOptions = ref<PermissionOption[]>([])
const keywords = ref('')
const current = ref(1)
const pageSize = ref(10)
const total = ref(0)
const loading = ref(false)
const saving = ref(false)
const editorVisible = ref(false)
const errorMessage = ref('')
const editor = reactive({ id: 0, roleName: '', menuIds: [] as number[], resourceIds: [] as number[] })

const pagination = computed(() => tablePagination(current.value, pageSize.value, total.value))
const menuChoices = computed(() => flattenOptions(menuOptions.value))
const resourceChoices = computed(() => flattenOptions(resourceOptions.value))

onMounted(() => {
  void Promise.all([loadRoles(), loadPermissionOptions()])
})

async function reload(): Promise<void> {
  current.value = 1
  await loadRoles()
}

async function loadRoles(): Promise<void> {
  loading.value = true
  errorMessage.value = ''
  try {
    const page = await listAdminRoles({ current: current.value, size: pageSize.value, keywords: keywords.value.trim() })
    roles.value = page.records
    total.value = page.count
  } catch (error) {
    errorMessage.value = apiErrorMessage(error, '角色列表加载失败')
    Message.error(errorMessage.value)
  } finally {
    loading.value = false
  }
}

async function loadPermissionOptions(): Promise<void> {
  try {
    const [menus, resources] = await Promise.all([listRoleMenus(), listRoleResources()])
    menuOptions.value = normalizeOptions(menus)
    resourceOptions.value = normalizeOptions(resources)
  } catch (error) {
    errorMessage.value = apiErrorMessage(error, '权限选项加载失败')
  }
}

function changePage(page: number): void {
  current.value = page
  void loadRoles()
}

function changePageSize(size: number): void {
  pageSize.value = size
  current.value = 1
  void loadRoles()
}

function openEditor(role?: AdminRole): void {
  editor.id = Number(role?.id || 0)
  editor.roleName = String(role?.roleName || '')
  editor.menuIds = idsFrom(role?.['menuIds'])
  editor.resourceIds = idsFrom(role?.['resourceIds'])
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
    Message.success('角色已保存')
    await loadRoles()
  } catch (error) {
    Message.error(apiErrorMessage(error, '角色保存失败'))
  } finally {
    saving.value = false
  }
}

async function deleteRole(id: unknown): Promise<void> {
  const roleId = Number(id)
  if (!roleId) return
  try {
    await deleteAdminRoles([roleId])
    Message.success('角色已删除')
    await loadRoles()
  } catch (error) {
    Message.error(apiErrorMessage(error, '角色删除失败'))
  }
}

function normalizeOptions(value: unknown): PermissionOption[] {
  if (!Array.isArray(value)) return []
  return value.flatMap((item) => {
    if (!item || typeof item !== 'object') return []
    const source = item as Record<string, unknown>
    const id = Number(source.id)
    if (!Number.isFinite(id) || id <= 0) return []
    const children = normalizeOptions(source.children)
    return [{ id, label: String(source.label || source.name || '未命名权限'), children }]
  })
}

function flattenOptions(options: PermissionOption[]): PermissionOption[] {
  return options.flatMap((option) => [
    { id: option.id, label: option.label },
    ...flattenOptions(option.children || [])
  ])
}

function idsFrom(value: unknown): number[] {
  return Array.isArray(value)
    ? value.map(Number).filter((id) => Number.isFinite(id) && id > 0)
    : []
}


</script>

<style scoped>
.permission-grid {
  display: grid;
  grid-template-columns: repeat(3, minmax(0, 1fr));
  gap: 10px 16px;
  max-height: 220px;
  overflow: auto;
  padding: 8px 4px;
}

@media (max-width: 800px) {
  .permission-grid { grid-template-columns: repeat(2, minmax(0, 1fr)); }
}
</style>
