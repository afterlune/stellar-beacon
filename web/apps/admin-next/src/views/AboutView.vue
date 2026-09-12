<template>
  <section class="admin-page">
    <AdminPageHeader title="关于我" description="写下希望访客了解的你，保存后会同步到博客展示页。">
      <template #actions>
        <a-button :disabled="!dirty" @click="reset">还原修改</a-button>
      </template>
    </AdminPageHeader>

    <a-card class="admin-form-panel admin-form-card" :bordered="false">
      <a-alert v-if="errorMessage" type="error" closable @close="errorMessage = ''">{{ errorMessage }}</a-alert>
      <a-spin v-if="!ready" class="about-loading" tip="正在加载关于内容…" />
      <a-form v-else class="config-form" :model="form" layout="vertical" @submit-success="save">
        <a-form-item label="内容">
          <a-textarea
            v-model="form.content"
            :max-length="100000"
            show-word-limit
            :auto-size="{ minRows: 16, maxRows: 32 }"
            placeholder="支持 Markdown / HTML，保存后由前台渲染到「关于我」页面。" />
          <template #help>这段内容会直接展示在博客的关于页面上。</template>
        </a-form-item>
        <div class="admin-form-actions">
          <a-button type="primary" html-type="submit" :loading="saving">保存</a-button>
          <span class="about-status">
            <template v-if="saving">正在保存…</template>
            <template v-else-if="dirty">有未保存的修改</template>
            <template v-else>已与服务器同步 · {{ lastSavedLabel }}</template>
          </span>
        </div>
      </a-form>
    </a-card>
  </section>
</template>

<script setup lang="ts">
import { computed, onMounted, reactive, ref } from 'vue'
import { Message } from '@arco-design/web-vue'

import { apiErrorMessage, getAbout, updateAbout } from '@/api/http'
import AdminPageHeader from '@/components/AdminPageHeader.vue'
import { formatDateTime } from '@/utils/format'

const form = reactive({ content: '' })
const original = ref('')
const saving = ref(false)
const errorMessage = ref('')
const ready = ref(false)
const lastSaved = ref('')

const dirty = computed(() => form.content !== original.value)
const lastSavedLabel = computed(() => (lastSaved.value ? formatDateTime(lastSaved.value) : '尚未保存'))

onMounted(() => void load())

async function load(): Promise<void> {
  try {
    const value = await getAbout()
    form.content = String(value.content || '')
    original.value = form.content
  } catch (error) {
    errorMessage.value = apiErrorMessage(error, '关于内容加载失败')
  } finally {
    ready.value = true
  }
}

function reset(): void {
  form.content = original.value
}

async function save(): Promise<void> {
  saving.value = true
  errorMessage.value = ''
  try {
    await updateAbout(form.content)
    original.value = form.content
    lastSaved.value = new Date().toISOString()
    Message.success('关于内容已保存')
  } catch (error) {
    errorMessage.value = apiErrorMessage(error, '关于内容保存失败')
    Message.error(errorMessage.value)
  } finally {
    saving.value = false
  }
}
</script>

<style scoped>
.about-loading {
  display: flex;
  justify-content: center;
  padding: 56px 0;
}

.about-status {
  color: var(--admin-muted);
  font-size: 12px;
}
</style>
