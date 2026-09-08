<template>
  <section class="admin-page memory-page">
    <AdminPageHeader title="Agent 记忆" description="仅由管理员审核长期记忆，所有撤回和冲突处理都会保留历史。" />
    <a-alert type="warning" :show-icon="true" :closable="false">
      长期记忆只面向管理员审核。撤回、解决和驳回都会保留不可变历史，公共 Agent 对话不会直接访问这里的数据。
    </a-alert>

    <a-tabs v-model:active-key="activeTab">
      <a-tab-pane key="assertions" title="记忆断言">
        <a-card title="当前断言" class="admin-card">
          <template #extra>
            <a-button @click="loadAssertions">刷新</a-button>
          </template>
          <a-space wrap class="filters">
            <a-input v-model="assertionFilters.subjectKey" placeholder="主体，例如 user:42" allow-clear @press-enter="reloadAssertions" />
            <a-input v-model="assertionFilters.predicate" placeholder="谓词" allow-clear @press-enter="reloadAssertions" />
            <a-select v-model="assertionFilters.status" placeholder="状态" allow-clear style="width: 140px" @change="reloadAssertions">
              <a-option value="active">有效</a-option>
              <a-option value="stale">过期</a-option>
              <a-option value="conflicted">冲突</a-option>
              <a-option value="retracted">已撤回</a-option>
            </a-select>
            <a-button type="primary" @click="reloadAssertions">查询</a-button>
          </a-space>
          <a-alert v-if="assertionError" class="inline-alert" type="error" closable @close="assertionError = ''">{{ assertionError }}</a-alert>
          <a-table
            class="memory-table"
            :data="assertions"
            :columns="assertionColumns"
            :loading="assertionLoading"
            :pagination="assertionPagination"
            row-key="id"
            @page-change="changeAssertionPage">
            <template #status="{ record }">
              <a-tag class="admin-status-tag" :color="statusColor(record.status)">{{ statusLabel(record.status) }}</a-tag>
            </template>
            <template #object="{ record }">
              <span class="object-cell">{{ record.object }}</span>
            </template>
            <template #confidence="{ record }">{{ formatConfidence(record.confidence) }}</template>
            <template #time="{ record }">{{ formatTime(record.updatedAt) }}</template>
            <template #actions="{ record }">
              <a-space class="admin-action-space">
                <a-button type="text" size="small" @click="openHistory(record.id)">历史</a-button>
                <a-popconfirm content="撤回后不能重新激活，确定继续吗？" @ok="revokeAssertion(record.id)">
                  <a-button type="text" status="danger" size="small" :disabled="record.status === 'retracted'">撤回</a-button>
                </a-popconfirm>
              </a-space>
            </template>
            <template #empty><a-empty description="暂无记忆断言" /></template>
          </a-table>
        </a-card>
      </a-tab-pane>

      <a-tab-pane key="conflicts" title="冲突审核">
        <a-card title="记忆冲突" class="admin-card">
          <template #extra>
            <a-button @click="loadConflicts">刷新</a-button>
          </template>
          <a-space wrap class="filters">
            <a-input v-model="conflictFilters.subjectKey" placeholder="主体，例如 user:42" allow-clear @press-enter="reloadConflicts" />
            <a-input v-model="conflictFilters.predicate" placeholder="谓词" allow-clear @press-enter="reloadConflicts" />
            <a-select v-model="conflictFilters.status" placeholder="状态" allow-clear style="width: 140px" @change="reloadConflicts">
              <a-option value="open">待处理</a-option>
              <a-option value="resolved">已解决</a-option>
              <a-option value="rejected">已驳回</a-option>
            </a-select>
            <a-button type="primary" @click="reloadConflicts">查询</a-button>
          </a-space>
          <a-alert v-if="conflictError" class="inline-alert" type="error" closable @close="conflictError = ''">{{ conflictError }}</a-alert>
          <a-table
            class="memory-table"
            :data="conflicts"
            :columns="conflictColumns"
            :loading="conflictLoading"
            :pagination="conflictPagination"
            row-key="id"
            @page-change="changeConflictPage">
            <template #status="{ record }">
              <a-tag class="admin-status-tag" :color="conflictStatusColor(record.status)">{{ conflictStatusLabel(record.status) }}</a-tag>
            </template>
            <template #members="{ record }">
              <a-space wrap>
                <a-tag v-for="member in record.members" :key="member.assertionId" :color="member.role === 'winner' ? 'green' : 'gray'">
                  {{ member.assertionId }} · {{ memberRoleLabel(member.role) }}
                </a-tag>
              </a-space>
            </template>
            <template #time="{ record }">{{ formatTime(record.updatedAt) }}</template>
            <template #actions="{ record }">
              <a-space class="admin-action-space">
                <a-button type="text" size="small" :disabled="record.status !== 'open' || record.members.length === 0" @click="openResolve(record)">选择赢家</a-button>
                <a-popconfirm content="驳回后会撤回全部冲突断言，确定继续吗？" @ok="rejectConflict(record.id)">
                  <a-button type="text" status="danger" size="small" :disabled="record.status !== 'open'">驳回</a-button>
                </a-popconfirm>
              </a-space>
            </template>
            <template #empty><a-empty description="暂无记忆冲突" /></template>
          </a-table>
        </a-card>
      </a-tab-pane>
    </a-tabs>

    <a-modal v-model:visible="historyVisible" title="断言修订历史" :footer="false" width="900px">
      <a-table :data="history" :columns="historyColumns" :pagination="false" row-key="revisionNo">
        <template #status="{ record }"><a-tag class="admin-status-tag" :color="statusColor(record.status)">{{ statusLabel(record.status) }}</a-tag></template>
        <template #object="{ record }"><span class="object-cell">{{ record.object }}</span></template>
        <template #time="{ record }">{{ formatTime(record.changedAt) }}</template>
        <template #empty><a-empty description="暂无修订历史" /></template>
      </a-table>
    </a-modal>

    <a-modal v-model:visible="resolveVisible" title="解决记忆冲突" :ok-loading="resolveLoading" @before-ok="submitResolve">
      <a-alert type="info" :show-icon="true" :closable="false" class="inline-alert">
        只能从冲突成员中选择赢家，其他成员会进入过期状态并保留历史。
      </a-alert>
      <a-form :model="resolveForm" layout="vertical">
        <a-form-item label="冲突主体">
          <a-input :model-value="selectedConflict?.subjectKey || ''" disabled />
        </a-form-item>
        <a-form-item label="赢家断言" required>
          <a-select v-model="resolveForm.winnerAssertionId" placeholder="选择冲突成员">
            <a-option v-for="member in selectedConflict?.members || []" :key="member.assertionId" :value="member.assertionId">
              {{ member.assertionId }} · {{ memberRoleLabel(member.role) }}
            </a-option>
          </a-select>
        </a-form-item>
        <a-form-item label="审核说明">
          <a-textarea v-model="resolveForm.resolution" maxlength="256" show-word-limit :auto-size="{ minRows: 3, maxRows: 6 }" />
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
  listAdminAgentMemoryAssertions,
  listAdminAgentMemoryConflicts,
  listAdminAgentMemoryHistory,
  rejectAdminAgentMemoryConflict,
  resolveAdminAgentMemoryConflict,
  revokeAdminAgentMemoryAssertion
} from '@/api/http'
import type { AgentMemoryAssertion, AgentMemoryConflict, AgentMemoryAssertionRevision } from '@shared/api-contract'
import AdminPageHeader from '@/components/AdminPageHeader.vue'
import { formatTime } from '@/utils/format'

const activeTab = ref('assertions')
const assertions = ref<AgentMemoryAssertion[]>([])
const conflicts = ref<AgentMemoryConflict[]>([])
const history = ref<AgentMemoryAssertionRevision[]>([])
const assertionCurrent = ref(1)
const conflictCurrent = ref(1)
const pageSize = 20
const assertionHasMore = ref(false)
const conflictHasMore = ref(false)
const assertionLoading = ref(false)
const conflictLoading = ref(false)
const historyVisible = ref(false)
const resolveVisible = ref(false)
const resolveLoading = ref(false)
const assertionError = ref('')
const conflictError = ref('')
const selectedConflict = ref<AgentMemoryConflict | null>(null)
const assertionFilters = reactive<{ subjectKey: string; predicate: string; status?: string }>({ subjectKey: '', predicate: '', status: undefined })
const conflictFilters = reactive<{ subjectKey: string; predicate: string; status?: string }>({ subjectKey: '', predicate: '', status: undefined })
const resolveForm = reactive({ winnerAssertionId: '', resolution: '' })

const assertionColumns = [
  { title: '主体', dataIndex: 'subjectKey', width: 130, ellipsis: true, tooltip: true },
  { title: '谓词', dataIndex: 'predicate', width: 120 },
  { title: '对象', dataIndex: 'object', slotName: 'object', width: 220, ellipsis: true, tooltip: true },
  { title: '来源', dataIndex: 'sourceType', width: 100 },
  { title: '版本', dataIndex: 'version', width: 70 },
  { title: '置信度', dataIndex: 'confidence', slotName: 'confidence', width: 90 },
  { title: '状态', dataIndex: 'status', slotName: 'status', width: 90 },
  { title: '更新时间', dataIndex: 'updatedAt', slotName: 'time', width: 170 },
  { title: '操作', dataIndex: 'actions', slotName: 'actions', width: 130 }
]

const historyColumns = [
  { title: '修订号', dataIndex: 'revisionNo', width: 80 },
  { title: '对象', dataIndex: 'object', slotName: 'object', ellipsis: true, tooltip: true },
  { title: '版本', dataIndex: 'version', width: 70 },
  { title: '状态', dataIndex: 'status', slotName: 'status', width: 90 },
  { title: '原因', dataIndex: 'reason', ellipsis: true, tooltip: true },
  { title: '变更时间', dataIndex: 'changedAt', slotName: 'time', width: 170 }
]

const conflictColumns = [
  { title: '主体', dataIndex: 'subjectKey', width: 130 },
  { title: '谓词', dataIndex: 'predicate', width: 120 },
  { title: '状态', dataIndex: 'status', slotName: 'status', width: 90 },
  { title: '冲突成员', dataIndex: 'members', slotName: 'members', ellipsis: true, tooltip: true },
  { title: '更新时间', dataIndex: 'updatedAt', slotName: 'time', width: 170 },
  { title: '操作', dataIndex: 'actions', slotName: 'actions', width: 150 }
]

const assertionPagination = computed(() => pagination(assertionCurrent.value, assertions.value.length, assertionHasMore.value))
const conflictPagination = computed(() => pagination(conflictCurrent.value, conflicts.value.length, conflictHasMore.value))

onMounted(() => {
  void Promise.all([loadAssertions(), loadConflicts()])
})

async function loadAssertions(): Promise<void> {
  assertionLoading.value = true
  assertionError.value = ''
  try {
    const page = await listAdminAgentMemoryAssertions({
      current: assertionCurrent.value,
      size: pageSize,
      subjectKey: assertionFilters.subjectKey.trim(),
      predicate: assertionFilters.predicate.trim(),
      ...(assertionFilters.status ? { status: assertionFilters.status } : {})
    })
    assertions.value = Array.isArray(page.items) ? page.items : []
    assertionHasMore.value = Boolean(page.hasMore)
  } catch (error) {
    assertionError.value = apiErrorMessage(error, '记忆断言加载失败')
    Message.error(assertionError.value)
  } finally {
    assertionLoading.value = false
  }
}

async function loadConflicts(): Promise<void> {
  conflictLoading.value = true
  conflictError.value = ''
  try {
    const page = await listAdminAgentMemoryConflicts({
      current: conflictCurrent.value,
      size: pageSize,
      subjectKey: conflictFilters.subjectKey.trim(),
      predicate: conflictFilters.predicate.trim(),
      ...(conflictFilters.status ? { status: conflictFilters.status } : {})
    })
    conflicts.value = Array.isArray(page.items) ? page.items : []
    conflictHasMore.value = Boolean(page.hasMore)
  } catch (error) {
    conflictError.value = apiErrorMessage(error, '记忆冲突加载失败')
    Message.error(conflictError.value)
  } finally {
    conflictLoading.value = false
  }
}

function reloadAssertions(): void {
  assertionCurrent.value = 1
  void loadAssertions()
}

function reloadConflicts(): void {
  conflictCurrent.value = 1
  void loadConflicts()
}

function changeAssertionPage(page: number): void {
  assertionCurrent.value = page
  void loadAssertions()
}

function changeConflictPage(page: number): void {
  conflictCurrent.value = page
  void loadConflicts()
}

async function openHistory(id: string): Promise<void> {
  try {
    const result = await listAdminAgentMemoryHistory(id)
    history.value = Array.isArray(result.items) ? result.items : []
    historyVisible.value = true
  } catch (error) {
    Message.error(apiErrorMessage(error, '记忆历史加载失败'))
  }
}

async function revokeAssertion(id: string): Promise<void> {
  try {
    await revokeAdminAgentMemoryAssertion(id)
    Message.success('记忆断言已撤回')
    await loadAssertions()
  } catch (error) {
    Message.error(apiErrorMessage(error, '撤回记忆断言失败'))
  }
}

function openResolve(conflict: AgentMemoryConflict): void {
  selectedConflict.value = conflict
  resolveForm.winnerAssertionId = conflict.members[0]?.assertionId || ''
  resolveForm.resolution = ''
  resolveVisible.value = true
}

async function submitResolve(done: (closed: boolean) => void): Promise<void> {
  if (!selectedConflict.value || !resolveForm.winnerAssertionId.trim()) {
    Message.error('请选择冲突赢家')
    done(false)
    return
  }
  resolveLoading.value = true
  try {
    await resolveAdminAgentMemoryConflict(selectedConflict.value.id, resolveForm.winnerAssertionId.trim(), resolveForm.resolution.trim())
    Message.success('记忆冲突已解决')
    await loadConflicts()
    done(true)
  } catch (error) {
    Message.error(apiErrorMessage(error, '解决记忆冲突失败'))
    done(false)
  } finally {
    resolveLoading.value = false
  }
}

async function rejectConflict(id: string): Promise<void> {
  try {
    await rejectAdminAgentMemoryConflict(id)
    Message.success('记忆冲突已驳回')
    await loadConflicts()
  } catch (error) {
    Message.error(apiErrorMessage(error, '驳回记忆冲突失败'))
  }
}

function pagination(current: number, itemCount: number, hasMore: boolean) {
  const total = hasMore ? current * pageSize + 1 : (current - 1) * pageSize + itemCount
  return { current, pageSize, total, showTotal: false, showJumper: false, showPageSize: false }
}



function formatConfidence(value: unknown): string {
  const number = Number(value)
  return Number.isFinite(number) ? number.toFixed(2) : '—'
}

function statusLabel(value: string): string {
  return ({ active: '有效', stale: '过期', conflicted: '冲突', retracted: '已撤回' } as Record<string, string>)[value] || value
}

function statusColor(value: string): string {
  return ({ active: 'green', stale: 'gray', conflicted: 'orange', retracted: 'red' } as Record<string, string>)[value] || 'gray'
}

function conflictStatusLabel(value: string): string {
  return ({ open: '待处理', resolved: '已解决', rejected: '已驳回' } as Record<string, string>)[value] || value
}

function conflictStatusColor(value: string): string {
  return ({ open: 'orange', resolved: 'green', rejected: 'red' } as Record<string, string>)[value] || 'gray'
}

function memberRoleLabel(value: string): string {
  return ({ candidate: '候选', winner: '赢家', rejected: '已驳回' } as Record<string, string>)[value] || value
}
</script>

<style scoped>
.memory-page {
  display: grid;
  gap: 16px;
}

.filters {
  margin-bottom: 16px;
}

.filters .arco-input-wrapper {
  width: 220px;
}

.inline-alert {
  margin-bottom: 16px;
}

.memory-table {
  margin-top: 8px;
}

.object-cell {
  display: block;
  max-width: 360px;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
</style>
