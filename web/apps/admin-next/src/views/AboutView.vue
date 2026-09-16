<template>
  <section class="admin-page">
    <AdminPageHeader :title="t('site.about.title')" :description="t('site.about.description')">
      <template #actions>
        <a-button :disabled="!dirty" @click="reset">{{ t('site.revert') }}</a-button>
      </template>
    </AdminPageHeader>

    <a-card class="admin-form-panel admin-form-card" :bordered="false">
      <a-alert v-if="errorMessage" type="error" closable @close="errorMessage = ''">{{ errorMessage }}</a-alert>
      <a-spin v-if="!ready" class="about-loading" :tip="t('site.about.loading')" />
      <a-form v-else class="config-form" :model="form" layout="vertical" @submit-success="save">
        <a-form-item :label="t('site.about.content')">
          <a-textarea
            v-model="form.content"
            :max-length="100000"
            show-word-limit
            :auto-size="{ minRows: 16, maxRows: 32 }"
            :placeholder="t('site.about.contentPlaceholder')" />
          <template #help>{{ t('site.about.contentHelp') }}</template>
        </a-form-item>
        <div class="admin-form-actions">
          <a-button type="primary" html-type="submit" :loading="saving">{{ t('common.save') }}</a-button>
          <span class="about-status">
            <template v-if="saving">{{ t('site.about.saving') }}</template>
            <template v-else-if="dirty">{{ t('site.unsavedChanges') }}</template>
            <template v-else>{{ t('site.about.syncedAt', { time: lastSavedLabel }) }}</template>
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
import { t } from '@/i18n'
import { formatDateTime } from '@/utils/format'

const form = reactive({ content: '' })
const original = ref('')
const saving = ref(false)
const errorMessage = ref('')
const ready = ref(false)
const lastSaved = ref('')

const dirty = computed(() => form.content !== original.value)
const lastSavedLabel = computed(() => (lastSaved.value ? formatDateTime(lastSaved.value) : t('site.about.neverSaved')))

onMounted(() => void load())

async function load(): Promise<void> {
  try {
    const value = await getAbout()
    form.content = String(value.content || '')
    original.value = form.content
  } catch (error) {
    errorMessage.value = apiErrorMessage(error, t('site.about.loadFailed'))
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
    Message.success(t('site.about.saveSuccess'))
  } catch (error) {
    errorMessage.value = apiErrorMessage(error, t('site.about.saveFailed'))
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
