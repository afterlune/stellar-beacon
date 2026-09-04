<template>
  <section>
    <a-card title="网站配置">
      <a-alert type="info" :closable="false">配置保存会沿用后端完整配置对象，未展示的字段会从当前读取结果原样保留。</a-alert>
      <a-alert v-if="errorMessage" type="error" closable @close="errorMessage = ''">{{ errorMessage }}</a-alert>
      <a-form class="config-form" :model="form" layout="vertical" @submit-success="save">
        <div class="config-grid">
          <a-form-item field="name" label="网站名称"><a-input v-model="form.name" /></a-form-item>
          <a-form-item field="englishName" label="英文名称"><a-input v-model="form.englishName" /></a-form-item>
          <a-form-item field="author" label="作者"><a-input v-model="form.author" /></a-form-item>
          <a-form-item field="logo" label="Logo URL"><a-input v-model="form.logo" /></a-form-item>
          <a-form-item field="github" label="GitHub"><a-input v-model="form.github" /></a-form-item>
          <a-form-item field="gitee" label="Gitee"><a-input v-model="form.gitee" /></a-form-item>
        </div>
        <a-form-item field="notice" label="公告"><a-textarea v-model="form.notice" :auto-size="{ minRows: 3, maxRows: 8 }" /></a-form-item>
        <a-button type="primary" html-type="submit" :loading="saving">保存</a-button>
      </a-form>
    </a-card>
  </section>
</template>

<script setup lang="ts">
import { onMounted, reactive, ref } from 'vue'
import { Message } from '@arco-design/web-vue'

import { apiErrorMessage, getWebsiteConfig, updateWebsiteConfig } from '@/api/http'

const form = reactive<Record<string, string | number>>({ name: '', englishName: '', author: '', logo: '', github: '', gitee: '', notice: '' })
const saving = ref(false)
const errorMessage = ref('')

onMounted(() => void load())
async function load(): Promise<void> {
  try {
    const value = await getWebsiteConfig()
    for (const [key, item] of Object.entries(value)) {
      if (typeof item === 'string' || typeof item === 'number') form[key] = item
    }
  } catch (error) { errorMessage.value = apiErrorMessage(error, '网站配置加载失败') }
}
async function save(): Promise<void> {
  saving.value = true
  try { await updateWebsiteConfig({ ...form }); Message.success('网站配置已保存') }
  catch (error) { Message.error(apiErrorMessage(error, '网站配置保存失败')) }
  finally { saving.value = false }
}
</script>

<style scoped>
.config-form { margin-top: 18px; }
.config-grid { display: grid; grid-template-columns: repeat(2, minmax(0, 1fr)); gap: 16px; }
@media (max-width: 800px) { .config-grid { grid-template-columns: 1fr; } }
</style>
