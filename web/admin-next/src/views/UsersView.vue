<template>
  <section class="admin-page">
    <AdminPageHeader title="用户管理" description="查看用户、角色与最近一次登录状态。">
      <template #actions>
        <a-space>
          <a-select v-model="loginType" allow-clear placeholder="登录方式" style="width: 130px" @change="reload">
            <a-option :value="1">邮箱</a-option>
            <a-option :value="2">QQ</a-option>
          </a-select>
          <a-input-search v-model="keywords" class="admin-filter-input" placeholder="搜索昵称" allow-clear @search="reload" />
        </a-space>
      </template>
    </AdminPageHeader>
    <a-card class="admin-panel" :bordered="false">
      <a-alert v-if="errorMessage" type="error" closable @close="errorMessage = ''">{{ errorMessage }}</a-alert>
      <div class="admin-table-shell">
        <a-table
          :data="users"
          :columns="columns"
          :loading="loading"
          :pagination="pagination"
          row-key="userInfoId"
          @page-change="changePage"
          @page-size-change="changePageSize">
          <template #roles="{ record }">
            <a-space wrap>
              <a-tag v-for="role in roleNames(record)" :key="role" color="arcoblue">{{ role }}</a-tag>
              <span v-if="roleNames(record).length === 0">—</span>
            </a-space>
          </template>
          <template #loginType="{ record }">{{ loginTypeLabel(record.loginType) }}</template>
          <template #disable="{ record }">
            <a-switch
              :model-value="Number(record.isDisable) === 1"
              :loading="pendingDisableId === userId(record)"
              @change="(value) => toggleDisable(record, value)" />
          </template>
          <template #actions="{ record }">
            <a-button type="text" size="small" @click="openEditor(record)">编辑</a-button>
          </template>
          <template #empty><div class="admin-table-empty"><a-empty description="暂无用户" /></div></template>
        </a-table>
      </div>
    </a-card>

    <a-modal v-model:visible="editorVisible" title="修改用户" :ok-loading="saving" @ok="saveEditor">
      <a-form :model="editor">
        <a-form-item field="nickname" label="昵称" required>
          <a-input v-model="editor.nickname" maxlength="50" show-word-limit />
        </a-form-item>
        <a-form-item field="roleIds" label="角色">
          <a-select v-model="editor.roleIds" multiple allow-clear placeholder="请选择角色">
            <a-option v-for="role in roleOptions" :key="role.id" :value="role.id">{{ role.roleName }}</a-option>
          </a-select>
        </a-form-item>
      </a-form>
    </a-modal>
  </section>
</template>

<script setup lang="ts">
import { computed, onMounted, reactive, ref } from 'vue'
import { Message } from '@arco-design/web-vue'

import {
  apiErrorMessage,
  listAdminPage,
  listUserRoles,
  updateAdminUser,
  updateAdminUserDisable
} from '@/api/http'
import AdminPageHeader from '@/components/AdminPageHeader.vue'
import { tablePagination } from '@/utils/pagination'
import type { AdminUser, UserRole } from '@shared/api-contract'

const columns = [
  { title: '昵称', dataIndex: 'nickname', ellipsis: true, tooltip: true },
  { title: '登录方式', dataIndex: 'loginType', width: 110, slotName: 'loginType' },
  { title: '角色', dataIndex: 'roles', ellipsis: true, slotName: 'roles' },
  { title: '登录 IP', dataIndex: 'ipAddress', width: 150 },
  { title: '禁用', dataIndex: 'isDisable', width: 100, slotName: 'disable' },
  { title: '最后登录', dataIndex: 'lastLoginTime', width: 180 },
  { title: '操作', dataIndex: 'actions', width: 90, slotName: 'actions' }
]

const users = ref<AdminUser[]>([])
const roleOptions = ref<UserRole[]>([])
const keywords = ref('')
const loginType = ref<number | undefined>(undefined)
const current = ref(1)
const pageSize = ref(10)
const total = ref(0)
const loading = ref(false)
const saving = ref(false)
const pendingDisableId = ref(0)
const errorMessage = ref('')
const editorVisible = ref(false)
const editor = reactive({ userInfoId: 0, nickname: '', roleIds: [] as number[] })

const pagination = computed(() => tablePagination(current.value, pageSize.value, total.value))

onMounted(() => {
  void Promise.all([loadUsers(), loadRoles()])
})

async function reload(): Promise<void> {
  current.value = 1
  await loadUsers()
}

async function loadUsers(): Promise<void> {
  loading.value = true
  errorMessage.value = ''
  try {
    const page = await listAdminPage<AdminUser>('admin/users', {
      current: current.value,
      size: pageSize.value,
      keywords: keywords.value.trim(),
      loginType: loginType.value ?? 0
    })
    users.value = page.records
    total.value = page.count
  } catch (error) {
    errorMessage.value = apiErrorMessage(error, '用户列表加载失败')
    Message.error(errorMessage.value)
  } finally {
    loading.value = false
  }
}

async function loadRoles(): Promise<void> {
  try {
    roleOptions.value = await listUserRoles()
  } catch (error) {
    errorMessage.value = apiErrorMessage(error, '角色选项加载失败')
  }
}

function changePage(page: number): void {
  current.value = page
  void loadUsers()
}

function changePageSize(size: number): void {
  pageSize.value = size
  current.value = 1
  void loadUsers()
}

function openEditor(user: AdminUser): void {
  editor.userInfoId = userId(user)
  editor.nickname = String(user.nickname || '')
  editor.roleIds = roleIds(user)
  editorVisible.value = true
}

async function saveEditor(): Promise<void> {
  if (!editor.userInfoId || !editor.nickname.trim()) {
    Message.error('用户 ID 和昵称不能为空')
    return
  }
  saving.value = true
  try {
    await updateAdminUser({
      userInfoId: editor.userInfoId,
      nickname: editor.nickname.trim(),
      roleIds: editor.roleIds
    })
    Message.success('用户信息已保存')
    editorVisible.value = false
    await loadUsers()
  } catch (error) {
    Message.error(apiErrorMessage(error, '用户信息保存失败'))
  } finally {
    saving.value = false
  }
}

async function toggleDisable(user: AdminUser, value: boolean | string | number): Promise<void> {
  const id = userId(user)
  if (!id) return
  const next = Boolean(value) ? 1 : 0
  pendingDisableId.value = id
  try {
    await updateAdminUserDisable(id, next)
    user.isDisable = next
    Message.success(next === 1 ? '用户已禁用' : '用户已启用')
  } catch (error) {
    Message.error(apiErrorMessage(error, '用户状态更新失败'))
  } finally {
    pendingDisableId.value = 0
  }
}

function userId(user: AdminUser): number {
  return Number(user.userInfoId || user.id || 0)
}

function roleIds(user: AdminUser): number[] {
  if (!Array.isArray(user.roles)) return []
  return user.roles.flatMap((role) => {
    if (typeof role === 'string') {
      const found = roleOptions.value.find((option) => option.roleName === role)
      return found ? [found.id] : []
    }
    return Number.isFinite(Number(role.id)) ? [Number(role.id)] : []
  })
}

function roleNames(user: AdminUser): string[] {
  if (!Array.isArray(user.roles)) return []
  return user.roles.map((role) => typeof role === 'string' ? role : role.roleName).filter(Boolean)
}

function loginTypeLabel(value: unknown): string {
  if (Number(value) === 1) return '邮箱'
  if (Number(value) === 2) return 'QQ'
  return '其他'
}


</script>
