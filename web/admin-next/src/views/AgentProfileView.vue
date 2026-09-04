<template>
  <section>
    <a-card title="Agent 人设配置">
      <template #extra><a-tag color="orange">不包含 Provider 密钥</a-tag></template>
      <a-alert v-if="errorMessage" type="error" closable @close="errorMessage = ''">{{ errorMessage }}</a-alert>
      <a-spin :loading="loading" style="display: block">
        <a-form :model="profile" layout="vertical" @submit-success="save">
          <a-form-item field="name" label="名称" required>
            <a-input v-model="profile.name" maxlength="64" show-word-limit />
          </a-form-item>
          <a-form-item field="promptVersion" label="Prompt 版本" required>
            <a-input v-model="profile.promptVersion" maxlength="128" show-word-limit />
          </a-form-item>
          <a-form-item field="systemPrompt" label="系统提示词" required>
            <a-textarea v-model="profile.systemPrompt" :auto-size="{ minRows: 5, maxRows: 12 }" maxlength="8000" show-word-limit />
            <template #help>文章检索内容仍会被视为不可信资料；修改版本号会让旧会话重新建立上下文。</template>
          </a-form-item>
          <a-form-item field="opening" label="开场白">
            <a-textarea v-model="profile.opening" :auto-size="{ minRows: 2, maxRows: 5 }" maxlength="1200" show-word-limit />
          </a-form-item>
          <a-divider orientation="left">昼夜节律提示</a-divider>
          <a-grid :cols="3" :col-gap="16">
            <a-grid-item>
              <a-form-item label="清醒">
                <a-textarea v-model="profile.rhythmPrompts.awake" :auto-size="{ minRows: 3, maxRows: 8 }" maxlength="1200" show-word-limit />
              </a-form-item>
            </a-grid-item>
            <a-grid-item>
              <a-form-item label="黄昏">
                <a-textarea v-model="profile.rhythmPrompts.dusk" :auto-size="{ minRows: 3, maxRows: 8 }" maxlength="1200" show-word-limit />
              </a-form-item>
            </a-grid-item>
            <a-grid-item>
              <a-form-item label="深夜">
                <a-textarea v-model="profile.rhythmPrompts.night" :auto-size="{ minRows: 3, maxRows: 8 }" maxlength="1200" show-word-limit />
              </a-form-item>
            </a-grid-item>
          </a-grid>
          <a-form-item field="enabled" label="人设状态">
            <a-switch v-model="profile.enabled">
              <template #checked>启用</template>
              <template #unchecked>停用</template>
            </a-switch>
          </a-form-item>
          <a-space>
            <a-button type="primary" html-type="submit" :loading="saving">保存人设</a-button>
            <span class="hint">当前修改保存到进程内配置存储；重启后的持久化控制面仍是后续任务。</span>
          </a-space>
        </a-form>
      </a-spin>
    </a-card>
  </section>
</template>

<script setup lang="ts">
import { onMounted, reactive, ref } from 'vue'
import { Message } from '@arco-design/web-vue'

import { apiErrorMessage, getAdminAgentProfile, updateAdminAgentProfile } from '@/api/http'

const loading = ref(false)
const saving = ref(false)
const errorMessage = ref('')
const profile = reactive({
  name: '',
  promptVersion: '',
  systemPrompt: '',
  opening: '',
  rhythmPrompts: { awake: '', dusk: '', night: '' } as Record<string, string>,
  enabled: true
})

onMounted(() => void load())

async function load(): Promise<void> {
  loading.value = true
  errorMessage.value = ''
  try {
    const value = await getAdminAgentProfile()
    profile.name = value.name
    profile.promptVersion = value.promptVersion
    profile.systemPrompt = value.systemPrompt
    profile.opening = value.opening || ''
    profile.rhythmPrompts = { awake: value.rhythmPrompts?.awake || '', dusk: value.rhythmPrompts?.dusk || '', night: value.rhythmPrompts?.night || '' }
    profile.enabled = value.enabled
  } catch (error) {
    errorMessage.value = apiErrorMessage(error, '人设配置加载失败')
    Message.error(errorMessage.value)
  } finally {
    loading.value = false
  }
}

async function save(): Promise<void> {
  if (!profile.name.trim() || !profile.promptVersion.trim() || !profile.systemPrompt.trim()) {
    Message.error('名称、Prompt 版本和系统提示词不能为空')
    return
  }
  saving.value = true
  try {
    const value = await updateAdminAgentProfile({
      name: profile.name.trim(),
      promptVersion: profile.promptVersion.trim(),
      systemPrompt: profile.systemPrompt.trim(),
      opening: profile.opening.trim(),
      awakePrompt: profile.rhythmPrompts.awake.trim(),
      duskPrompt: profile.rhythmPrompts.dusk.trim(),
      nightPrompt: profile.rhythmPrompts.night.trim(),
      enabled: profile.enabled
    })
    profile.name = value.name
    profile.promptVersion = value.promptVersion
    profile.systemPrompt = value.systemPrompt
    profile.opening = value.opening || ''
    profile.rhythmPrompts = { awake: value.rhythmPrompts?.awake || '', dusk: value.rhythmPrompts?.dusk || '', night: value.rhythmPrompts?.night || '' }
    profile.enabled = value.enabled
    Message.success('人设已保存')
  } catch (error) {
    Message.error(apiErrorMessage(error, '人设保存失败'))
  } finally {
    saving.value = false
  }
}
</script>

<style scoped>
.hint {
  color: var(--color-text-3);
  font-size: 12px;
}
</style>
