<template>
  <section>
    <a-card title="友链管理">
      <template #extra>
        <a-space>
          <a-input-search v-model="keywords" placeholder="搜索友链名称" allow-clear style="width: 220px" @search="reload" />
          <a-popconfirm
            content="确定删除选中的友链吗？"
            :disabled="selectedIds.length === 0"
            @ok="deleteLinks(selectedIds)">
            <a-button status="danger" :disabled="selectedIds.length === 0">批量删除</a-button>
          </a-popconfirm>
          <a-button type="primary" @click="openEditor()">新增</a-button>
        </a-space>
      </template>
      <a-alert v-if="errorMessage" type="error" closable @close="errorMessage = ''">{{ errorMessage }}</a-alert>
      <a-table
        v-model:selected-keys="selectedKeys"
        :row-selection="{ type: 'checkbox', showCheckedAll: true, onlyCurrent: true }"
        :data="links"
        :columns="columns"
        :loading="loading"
        :pagination="pagination"
        row-key="id"
        @page-change="changePage"
        @page-size-change="changePageSize">
        <template #avatar="{ record }">
          <img v-if="isHttpUrl(record.linkAvatar)" class="link-avatar" :src="record.linkAvatar" alt="友链头像" />
          <span v-else>—</span>
        </template>
        <template #address="{ record }">
          <span class="link-address" :title="record.linkAddress">{{ record.linkAddress || '—' }}</span>
        </template>
        <template #intro="{ record }">
          <span class="link-intro" :title="record.linkIntro">{{ record.linkIntro || '—' }}</span>
        </template>
        <template #createTime="{ record }">{{ formatTime(record.createTime) }}</template>
        <template #actions="{ record }">
          <a-space>
            <a-button type="text" size="small" @click="openEditor(record)">编辑</a-button>
            <a-popconfirm content="确定删除该友链吗？" @ok="deleteLinks([record.id])">
              <a-button type="text" status="danger" size="small">删除</a-button>
            </a-popconfirm>
          </a-space>
        </template>
        <template #empty><a-empty description="暂无友链" /></template>
      </a-table>
    </a-card>

    <a-modal
      v-model:visible="editorVisible"
      :title="editor.id ? '编辑友链' : '新增友链'"
      :ok-loading="saving"
      width="620px"
      @before-ok="saveEditor">
      <a-form :model="editor" layout="vertical">
        <a-form-item field="linkName" label="友链名称" required>
          <a-input v-model="editor.linkName" maxlength="20" show-word-limit />
        </a-form-item>
        <a-form-item field="linkAvatar" label="头像地址" required>
          <a-input v-model="editor.linkAvatar" maxlength="255" placeholder="HTTPS 图片地址" />
        </a-form-item>
        <a-form-item field="linkAddress" label="链接地址" required>
          <a-input v-model="editor.linkAddress" maxlength="50" placeholder="友链主页地址" />
        </a-form-item>
        <a-form-item field="linkIntro" label="友链介绍" required>
          <a-textarea v-model="editor.linkIntro" maxlength="100" show-word-limit :auto-size="{ minRows: 3, maxRows: 6 }" />
        </a-form-item>
      </a-form>
    </a-modal>
  </section>
</template>

<script setup lang="ts">
import { computed, onMounted, reactive, ref } from 'vue'
import { Message } from '@arco-design/web-vue'

import { apiErrorMessage, deleteAdminFriendLinks, listAdminFriendLinks, saveAdminFriendLink } from '@/api/http'
import type { AdminFriendLink } from '@shared/api-contract'

const columns = [
  { title: '头像', dataIndex: 'linkAvatar', width: 90, slotName: 'avatar' },
  { title: '名称', dataIndex: 'linkName', width: 150 },
  { title: '地址', dataIndex: 'linkAddress', slotName: 'address', ellipsis: true, tooltip: true },
  { title: '介绍', dataIndex: 'linkIntro', slotName: 'intro', ellipsis: true, tooltip: true },
  { title: '创建时间', dataIndex: 'createTime', slotName: 'createTime', width: 180 },
  { title: '操作', dataIndex: 'actions', slotName: 'actions', width: 150 }
]

const links = ref<AdminFriendLink[]>([])
const selectedKeys = ref<Array<string | number>>([])
const keywords = ref('')
const current = ref(1)
const pageSize = ref(10)
const total = ref(0)
const loading = ref(false)
const saving = ref(false)
const errorMessage = ref('')
const editorVisible = ref(false)
const editor = reactive({ id: 0, linkName: '', linkAvatar: '', linkAddress: '', linkIntro: '' })

const selectedIds = computed(() => [...new Set(selectedKeys.value.map(Number).filter((id) => Number.isInteger(id) && id > 0))])
const pagination = computed(() => ({
  current: current.value,
  pageSize: pageSize.value,
  total: total.value,
  showTotal: true,
  showJumper: true,
  showPageSize: true
}))

onMounted(() => void load())

async function reload(): Promise<void> {
  current.value = 1
  await load()
}

async function load(): Promise<void> {
  loading.value = true
  errorMessage.value = ''
  try {
    const page = await listAdminFriendLinks({ current: current.value, size: pageSize.value, keywords: keywords.value.trim() })
    links.value = page.records
    total.value = page.count
    const available = new Set(links.value.map((link) => Number(link.id)))
    selectedKeys.value = selectedKeys.value.filter((key) => available.has(Number(key)))
  } catch (error) {
    errorMessage.value = apiErrorMessage(error, '友链列表加载失败')
    Message.error(errorMessage.value)
  } finally {
    loading.value = false
  }
}

function changePage(page: number): void {
  current.value = page
  void load()
}

function changePageSize(size: number): void {
  pageSize.value = size
  current.value = 1
  void load()
}

function openEditor(link?: AdminFriendLink): void {
  editor.id = Number(link?.id || 0)
  editor.linkName = String(link?.linkName || '')
  editor.linkAvatar = String(link?.linkAvatar || '')
  editor.linkAddress = String(link?.linkAddress || '')
  editor.linkIntro = String(link?.linkIntro || '')
  editorVisible.value = true
}

async function saveEditor(done: (closed: boolean) => void): Promise<void> {
  if (!editor.linkName.trim() || !editor.linkAvatar.trim() || !editor.linkAddress.trim() || !editor.linkIntro.trim()) {
    Message.error('友链名称、头像、地址和介绍不能为空')
    done(false)
    return
  }
  saving.value = true
  try {
    await saveAdminFriendLink({
      id: editor.id || undefined,
      linkName: editor.linkName.trim(),
      linkAvatar: editor.linkAvatar.trim(),
      linkAddress: editor.linkAddress.trim(),
      linkIntro: editor.linkIntro.trim()
    })
    Message.success('友链已保存')
    editorVisible.value = false
    await load()
    done(true)
  } catch (error) {
    Message.error(apiErrorMessage(error, '友链保存失败'))
    done(false)
  } finally {
    saving.value = false
  }
}

async function deleteLinks(ids: number[]): Promise<void> {
  const validIds = [...new Set(ids.map(Number).filter((id) => Number.isInteger(id) && id > 0))]
  if (validIds.length === 0) return
  try {
    await deleteAdminFriendLinks(validIds)
    selectedKeys.value = selectedKeys.value.filter((key) => !validIds.includes(Number(key)))
    Message.success('友链已删除')
    await load()
  } catch (error) {
    Message.error(apiErrorMessage(error, '友链删除失败'))
  }
}

function isHttpUrl(value: unknown): value is string {
  return typeof value === 'string' && /^https?:\/\//i.test(value)
}

function formatTime(value: unknown): string {
  return typeof value === 'string' ? value.replace('T', ' ').replace(/\.\d+Z$/, '') : '—'
}
</script>

<style scoped>
.link-avatar {
  width: 42px;
  height: 42px;
  border-radius: 50%;
  object-fit: cover;
}

.link-address,
.link-intro {
  display: block;
  max-width: 280px;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
</style>
