<template>
  <section class="admin-page review-policy">
    <AdminPageHeader title="Agent 审核策略" description="把生成边界、过期时间和敏感内容拦截规则放在清晰的审核流程里。" />
    <a-card class="admin-form-panel admin-form-card" :bordered="false" title="审核规则">
      <template #extra><a-tag color="green">人工审核强制开启</a-tag></template>
      <a-alert type="info" :show-icon="true" :closable="false">
        这里仅调整候选生成的边界和过期时间，模型不能通过配置直接发布内容。保存使用版本号防止覆盖其他管理员的修改。
      </a-alert>
      <a-spin :loading="loading" style="display: block">
        <a-form :model="policy" layout="vertical" @submit-success="save">
          <a-form-item label="当前版本">
            <a-input-number v-model="policy.version" disabled />
          </a-form-item>
          <a-form-item label="审核要求">
            <a-switch :model-value="policy.reviewRequired" disabled>
              <template #checked>所有生成物必须人工审核</template>
              <template #unchecked>不可关闭</template>
            </a-switch>
          </a-form-item>
          <div class="policy-grid">
            <a-form-item label="审核有效期（小时）">
              <a-input-number v-model="policy.reviewTtlHours" :min="1" :max="720" />
            </a-form-item>
            <a-form-item label="候选最大字符数">
              <a-input-number v-model="policy.maxCandidateRunes" :min="4" :max="10000" />
            </a-form-item>
            <a-form-item label="相似度拦截阈值">
              <a-input-number v-model="policy.similarityThreshold" :min="0.01" :max="1" :step="0.01" :precision="2" />
            </a-form-item>
            <a-form-item label="每日候选上限">
              <a-input-number v-model="policy.dailyLimit" :min="1" :max="100" />
            </a-form-item>
            <a-form-item label="单文章候选上限">
              <a-input-number v-model="policy.perArticleLimit" :min="1" :max="20" />
            </a-form-item>
            <a-form-item label="单行为候选上限">
              <a-input-number v-model="policy.perActionLimit" :min="1" :max="100" />
            </a-form-item>
          </div>
          <a-form-item label="允许的行为类型">
            <a-checkbox-group v-model="policy.allowedActions">
              <a-checkbox value="comment">评论候选</a-checkbox>
              <a-checkbox value="talk">说说候选</a-checkbox>
              <a-checkbox value="wake">唤醒建议</a-checkbox>
            </a-checkbox-group>
          </a-form-item>
          <a-form-item label="敏感内容拦截词（每行一个）">
            <a-textarea v-model="policy.sensitivePatternsText" :auto-size="{ minRows: 5, maxRows: 12 }" maxlength="8192" show-word-limit />
          </a-form-item>
          <a-space>
            <a-button type="primary" html-type="submit" :loading="saving">保存策略</a-button>
            <span class="hint">版本 {{ policy.version }}；行为 Worker 重启后读取最新策略，写作预览会立即使用最新有效期。</span>
          </a-space>
        </a-form>
      </a-spin>
    </a-card>
  </section>
</template>

<script setup lang="ts">
import { onMounted, reactive, ref } from 'vue'
import { Message } from '@arco-design/web-vue'

import { apiErrorMessage, getAdminAgentReviewPolicy, updateAdminAgentReviewPolicy } from '@/api/http'
import AdminPageHeader from '@/components/AdminPageHeader.vue'

const loading = ref(false)
const saving = ref(false)
const policy = reactive({
  version: 1,
  reviewRequired: true,
  reviewTtlHours: 168,
  maxCandidateRunes: 500,
  similarityThreshold: 0.82,
  dailyLimit: 3,
  perArticleLimit: 1,
  perActionLimit: 3,
  allowedActions: ['comment', 'talk', 'wake'],
  sensitivePatternsText: ''
})

onMounted(() => void load())

async function load(): Promise<void> {
  loading.value = true
  try {
    const value = await getAdminAgentReviewPolicy()
    policy.version = value.version
    policy.reviewRequired = value.reviewRequired
    policy.reviewTtlHours = Math.max(1, Math.round(value.reviewTtlSeconds / 3600))
    policy.maxCandidateRunes = value.maxCandidateRunes
    policy.similarityThreshold = value.similarityThreshold
    policy.dailyLimit = value.dailyLimit
    policy.perArticleLimit = value.perArticleLimit
    policy.perActionLimit = value.perActionLimit
    policy.allowedActions = [...value.allowedActions]
    policy.sensitivePatternsText = value.sensitivePatterns.join('\n')
  } catch (error) {
    Message.error(apiErrorMessage(error, '审核策略加载失败'))
  } finally {
    loading.value = false
  }
}

async function save(): Promise<void> {
  if (policy.allowedActions.length === 0) {
    Message.error('至少选择一种行为类型')
    return
  }
  saving.value = true
  try {
    const value = await updateAdminAgentReviewPolicy({
      version: policy.version,
      reviewTtlSeconds: Math.round(policy.reviewTtlHours * 3600),
      maxCandidateRunes: policy.maxCandidateRunes,
      similarityThreshold: policy.similarityThreshold,
      dailyLimit: policy.dailyLimit,
      perArticleLimit: policy.perArticleLimit,
      perActionLimit: policy.perActionLimit,
      allowedActions: [...policy.allowedActions],
      sensitivePatterns: policy.sensitivePatternsText.split(/\r?\n/).map((item) => item.trim()).filter(Boolean)
    })
    policy.version = value.version
    policy.reviewRequired = value.reviewRequired
    policy.reviewTtlHours = Math.max(1, Math.round(value.reviewTtlSeconds / 3600))
    policy.maxCandidateRunes = value.maxCandidateRunes
    policy.similarityThreshold = value.similarityThreshold
    policy.dailyLimit = value.dailyLimit
    policy.perArticleLimit = value.perArticleLimit
    policy.perActionLimit = value.perActionLimit
    policy.allowedActions = [...value.allowedActions]
    policy.sensitivePatternsText = value.sensitivePatterns.join('\n')
    Message.success('审核策略已保存')
  } catch (error) {
    Message.error(apiErrorMessage(error, '审核策略保存失败'))
  } finally {
    saving.value = false
  }
}
</script>

<style scoped>
.review-policy {
  display: grid;
  gap: 16px;
}

.policy-grid {
  display: grid;
  grid-template-columns: repeat(3, minmax(0, 1fr));
  gap: 16px;
}

.hint {
  color: var(--color-text-3);
  font-size: 12px;
}

@media (max-width: 900px) {
  .policy-grid { grid-template-columns: 1fr; }
}
</style>
