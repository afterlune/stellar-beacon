<template>
  <section class="admin-page">
    <AdminPageHeader title="关于我" description="写下希望访客了解的你，保存后会同步到博客展示页。" />
    <a-card class="admin-form-panel admin-form-card" :bordered="false">
      <a-alert v-if="errorMessage" type="error" closable @close="errorMessage = ''">{{ errorMessage }}</a-alert>
      <a-spin v-if="!ready" class="about-loading" tip="正在加载关于内容..." />
      <a-form v-else class="config-form" layout="vertical" @submit-success="save">
        <a-form-item label="内容"><a-textarea v-model="content" :max-length="100000" show-word-limit :auto-size="{ minRows: 16, maxRows: 32 }" /></a-form-item>
        <div class="admin-form-actions"><a-button type="primary" html-type="submit" :loading="saving">保存</a-button></div>
      </a-form>
    </a-card>
  </section>
</template>

<script setup lang="ts">
import { onMounted, ref } from 'vue'
import { Message } from '@arco-design/web-vue'

import { apiErrorMessage, getAbout, updateAbout } from '@/api/http'
import AdminPageHeader from '@/components/AdminPageHeader.vue'

const content = ref('')
const saving = ref(false)
const errorMessage = ref('')
const ready = ref(false)
onMounted(() => void load())
async function load(): Promise<void> {
  try { content.value = String((await getAbout()).content || '') }
  catch (error) { errorMessage.value = apiErrorMessage(error, '关于内容加载失败') }
  finally { ready.value = true }
}
async function save(): Promise<void> {
  saving.value = true
  try { await updateAbout(content.value); Message.success('关于内容已保存') }
  catch (error) { Message.error(apiErrorMessage(error, '关于内容保存失败')) }
  finally { saving.value = false }
}
</script>

<style scoped>
.about-loading { display: flex; justify-content: center; padding: 56px 0; }
</style>
