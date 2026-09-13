<template>
  <section class="admin-page">
    <AdminPageHeader title="用户管理" description="查看用户、角色及登录状态。">
      <template #actions>
        <a-input-search
          v-model="keywords"
          class="admin-filter-input"
          placeholder="搜索昵称"
          allow-clear
          @search="reload" />
        <a-button :loading="loading" @click="load">
          <template #icon><IconRefresh /></template>
          刷新
        </a-button>
      </template>
    </AdminPageHeader>

    <a-card class="admin-panel" :bordered="false">
      <div class="admin-table-toolbar">
        <div class="admin-table-toolbar-main">
          <a-select v-model="loginType" placeholder="全部登录方式" allow-clear style="width: 160px" @change="reload">
            <a-option :value="1">邮箱</a-option>
            <a-option :value="2">QQ</a-option>
          </a-select>
          <a-select v-model="disableFilter" placeholder="全部状态" allow-clear style="width: 140px" @change="reload">
            <a-option :value="0">正常</a-option>
            <a-option :value="1">已禁用</a-option>
          </a-select>
          <a-button v-if="hasFilters" type="text" size="small" @click="resetFilters">重置筛选</a-button>
        </div>
        <div class="admin-table-toolbar-actions">
          <span class="admin-toolbar-caption">共 {{ total }} 位用户</span>
        </div>
      </div>

      <AdminErrorState v-if="errorMessage" :error="errorMessage" title="用户列表加载失败" @retry="load" />

      <div class="admin-table-shell">
        <a-table
          :data="visibleUsers"
          :columns="columns"
          :loading="loading"
          :pagination="pagination"
          row-key="userInfoId"
          @page-change="changePage"
          @page-size-change="changePageSize">
          <template #nickname="{ record }">
            <div class="user-cell">
              <a-avatar :size="32" :image-url="record.avatar">{{ initialOf(record.nickname) }}</a-avatar>
              <span class="user-cell-copy">
                <strong :title="String(record.nickname || '')">{{ record.nickname || '未命名用户' }}</strong>
                <small
                  v-if="record.email || record.username"
                  :title="String(record.email || record.username || '')">{{ record.email || record.username }}</small>
                <small v-else class="admin-muted-cell">未绑定邮箱</small>
              </span>
            </div>
          </template>
          <template #roles="{ record }">
            <a-space v-if="roleNames(record).length" wrap :size="4">
              <a-tag v-for="role in roleNames(record).slice(0, 2)" :key="role" color="arcoblue">{{ role }}</a-tag>
              <a-tooltip v-if="roleNames(record).length > 2" :content="roleNames(record).join('、')">
                <a-tag>+{{ roleNames(record).length - 2 }}</a-tag>
              </a-tooltip>
            </a-space>
            <span v-else class="admin-muted-cell">未分配角色</span>
          </template>
          <template #loginType="{ record }">{{ loginTypeLabel(record.loginType) }}</template>
          <template #disable="{ record }">
            <div class="admin-status-switch">
              <a-tooltip :content="Number(record.isDisable) === 1 ? '点击启用该账号' : '点击禁用该账号'">
                <a-switch
                  :model-value="Number(record.isDisable) === 1"
                  :loading="pendingDisableId === userId(record)"
                  :disabled="pendingDisableId !== 0 && pendingDisableId !== userId(record)"
                  @change="(value) => toggleDisable(record, value)" />
              </a-tooltip>
              <span :class="['admin-status-switch-label', Number(record.isDisable) === 1 ? 'is-disabled' : 'is-active']">
                {{ Number(record.isDisable) === 1 ? '已禁用' : '正常' }}
              </span>
            </div>
          </template>
          <template #time="{ record }"><span class="admin-cell-nowrap">{{ formatDateTime(record.lastLoginTime) }}</span></template>
          <template #actions="{ record }">
            <a-button type="text" size="small" @click="openEditor(record)">编辑</a-button>
          </template>
          <template #empty>
            <AdminEmptyState
              :icon="IconUserGroup"
              :title="hasFilters ? '没有匹配的用户' : '暂无用户'"
              :description="hasFilters ? '换个关键词或重置筛选条件再试一次。' : '当访客注册成为站点用户后，会出现在这里。'">
              <a-button v-if="hasFilters" size="small" @click="resetFilters">重置筛选</a-button>
            </AdminEmptyState>
          </template>
        </a-table>
      </div>
    </a-card>

    <a-modal
      v-model:visible="editorVisible"
      title="修改用户"
      :ok-loading="saving"
      :mask-closable="false"
      width="560px"
      @ok="saveEditor">
      <a-form :model="editor" layout="vertical">
        <a-form-item field="nickname" label="昵称" required>
          <a-input v-model="editor.nickname" maxlength="50" show-word-limit placeholder="展示给其他用户的名字" />
        </a-form-item>
        <a-form-item field="roleIds" label="角色">
          <a-select v-model="editor.roleIds" multiple allow-clear placeholder="请选择角色" :loading="rolesLoading">
            <a-option v-for="role in roleOptions" :key="role.id" :value="role.id">{{ role.roleName }}</a-option>
          </a-select>
          <template #help>角色决定该账号可以访问的后台菜单与接口。留空表示不分配任何角色。</template>
        </a-form-item>
        <a-descriptions :column="2" size="small" bordered class="user-editor-meta">
          <a-descriptions-item label="用户 ID">{{ editor.userInfoId || '—' }}</a-descriptions-item>
          <a-descriptions-item label="登录方式">{{ loginTypeLabel(editor.loginType) }}</a-descriptions-item>
        </a-descriptions>
      </a-form>
    </a-modal>
  </section>
</template>

<script setup lang="ts">
import { computed, onMounted, reactive, ref } from 'vue'
import { Message } from '@arco-design/web-vue'
import { IconRefresh, IconUserGroup } from '@arco-design/web-vue/es/icon'

import {
  apiErrorMessage,
  listAdminPage,
  listUserRoles,
  updateAdminUser,
  updateAdminUserDisable
} from '@/api/http'
import AdminEmptyState from '@/components/AdminEmptyState.vue'
import AdminErrorState from '@/components/AdminErrorState.vue'
import AdminPageHeader from '@/components/AdminPageHeader.vue'
import { useAsyncList } from '@/composables/useAsyncList'
import { useQueryFilters } from '@/composables/useQueryFilters'
import { readStoredPageSize, useStoredPageSize } from '@/composables/useTablePrefs'
import { formatDateTime, initialOf } from '@/utils/format'
import { tablePagination } from '@/utils/pagination'
import type { AdminUser, UserRole } from '@stellar-beacon/api-contract'

const VIEW_KEY = 'users'

const columns = [
  { title: '用户', dataIndex: 'nickname', slotName: 'nickname', minWidth: 210 },
  { title: '登录方式', dataIndex: 'loginType', slotName: 'loginType', width: 108 },
  { title: '角色', dataIndex: 'roles', slotName: 'roles', width: 190 },
  { title: '登录 IP', dataIndex: 'ipAddress', width: 148, ellipsis: true, tooltip: true },
  { title: '状态', dataIndex: 'isDisable', width: 132, slotName: 'disable' },
  { title: '最后登录', dataIndex: 'lastLoginTime', width: 184, slotName: 'time' },
  { title: '操作', dataIndex: 'actions', width: 88, slotName: 'actions' }
]

const roleOptions = ref<UserRole[]>([])
const rolesLoading = ref(false)
const keywords = ref('')
const loginType = ref<number | undefined>(undefined)
const disableFilter = ref<number | undefined>(undefined)
const saving = ref(false)
const pendingDisableId = ref(0)
const editorVisible = ref(false)
const editor = reactive({ userInfoId: 0, nickname: '', roleIds: [] as number[], loginType: 0 })

const {
  items: users,
  total,
  current,
  pageSize,
  loading,
  error: errorMessage,
  load,
  reload,
  changePage: gotoPage,
  changePageSize: applyPageSize
} = useAsyncList<AdminUser>(
  ({ current: page, pageSize: size, signal }) => listAdminPage<AdminUser>('admin/users', {
    current: page,
    size,
    keywords: keywords.value.trim(),
    loginType: loginType.value ?? 0
  }, { signal }),
  { pageSize: readStoredPageSize(VIEW_KEY), fallbackMessage: '用户列表加载失败' }
)

useStoredPageSize(VIEW_KEY, pageSize)
useQueryFilters([
  { key: 'keywords', ref: keywords, debounce: true },
  { key: 'loginType', ref: loginType },
  { key: 'disable', ref: disableFilter },
  { key: 'page', ref: current }
], { onRestore: () => void load(), onSearch: () => void reload() })

const pagination = computed(() => tablePagination(current.value, pageSize.value, total.value))
const hasFilters = computed(() =>
  Boolean(keywords.value.trim()) || loginType.value !== undefined || disableFilter.value !== undefined
)
/** Client-side refinement so the status filter works without a backend flag. */
const visibleUsers = computed(() => {
  if (disableFilter.value === undefined) return users.value
  // URL 还原回来的是字符串，比较前统一转成数字。
  const wantDisabled = Number(disableFilter.value) === 1
  return users.value.filter((user) => (Number(user.isDisable) === 1) === wantDisabled)
})

onMounted(() => {
  void loadRoles()
})

function resetFilters(): void {
  keywords.value = ''
  loginType.value = undefined
  disableFilter.value = undefined
  void reload()
}

async function loadRoles(): Promise<void> {
  rolesLoading.value = true
  try {
    roleOptions.value = await listUserRoles()
  } catch (error) {
    Message.error(apiErrorMessage(error, '角色选项加载失败'))
  } finally {
    rolesLoading.value = false
  }
}

function changePage(page: number): void {
  gotoPage(page)
}

function changePageSize(size: number): void {
  applyPageSize(size)
}

function openEditor(user: AdminUser): void {
  editor.userInfoId = userId(user)
  editor.nickname = String(user.nickname || '')
  editor.roleIds = roleIds(user)
  editor.loginType = Number(user.loginType || 0)
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
    await load()
  } catch (error) {
    Message.error(apiErrorMessage(error, '用户信息保存失败'))
  } finally {
    saving.value = false
  }
}

async function toggleDisable(user: AdminUser, value: boolean | string | number): Promise<void> {
  const id = userId(user)
  if (!id) return
  const previous = Number(user.isDisable) === 1 ? 1 : 0
  const next = Boolean(value) ? 1 : 0
  user.isDisable = next
  pendingDisableId.value = id
  try {
    await updateAdminUserDisable(id, next)
    Message.success(next === 1 ? '用户已禁用' : '用户已启用')
  } catch (error) {
    user.isDisable = previous
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
  return user.roles.map((role) => (typeof role === 'string' ? role : role.roleName)).filter(Boolean)
}

function loginTypeLabel(value: unknown): string {
  if (Number(value) === 1) return '邮箱'
  if (Number(value) === 2) return 'QQ'
  return '其他'
}
</script>

<style scoped>
.user-cell {
  display: flex;
  align-items: center;
  gap: 10px;
  min-width: 0;
}

/* 开关单独出现时状态含义不明确，补一个文字标签 */
.admin-status-switch {
  display: inline-flex;
  align-items: center;
  gap: 8px;
}

.admin-status-switch-label {
  font-size: 12px;
  font-weight: 600;
  white-space: nowrap;
}

.admin-status-switch-label.is-active {
  color: var(--admin-sage);
}

.admin-status-switch-label.is-disabled {
  color: var(--admin-danger);
}

.user-cell-copy {
  min-width: 0;
  display: grid;
  gap: 1px;
}

.user-cell-copy strong,
.user-cell-copy small {
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.user-cell-copy strong {
  color: var(--admin-ink-strong);
  font-size: 13px;
  font-weight: 650;
}

.user-cell-copy small {
  color: var(--admin-subtle);
  font-size: 11px;
}

.user-editor-meta {
  margin-top: 4px;
}
</style>
