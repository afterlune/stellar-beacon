<template>
  <section class="comment-governance-queues" data-testid="comment-governance-queues">
    <a-card class="admin-panel" :bordered="false" data-testid="admin-comment-report-queue">
      <template #title>{{ t('comments.governance.reportsTitle') }}</template>
      <template #extra>
        <a-space>
          <span class="queue-total">{{ t('comments.governance.total', { total: reportTotal }) }}</span>
          <a-button size="small" :loading="loading" @click="load">{{ t('common.refresh') }}</a-button>
        </a-space>
      </template>
      <a-spin :loading="loading" style="width: 100%">
        <div v-if="reports.length" class="queue-list">
          <article v-for="item in reports" :key="item.commentId" class="queue-item">
            <div class="queue-item__main">
              <strong>#{{ item.commentId }} · {{ item.collectionTitle || t('comments.governance.unknownCollection') }}</strong>
              <p>{{ plainText(item.commentContent) || t('comments.governance.emptyContent') }}</p>
              <small>
                {{ item.commentNickname || t('comments.governance.unknownUser') }} ·
                {{ t('comments.governance.reportCount', { count: Number(item.reportCount || 0) }) }} ·
                {{ reasonLabel(item.reasons) }}
                <template v-if="item.latestDetail"> · {{ item.latestDetail }}</template>
              </small>
            </div>
            <a-space class="queue-item__actions">
              <a-button size="small" :loading="busyKey === `report-dismiss-${item.commentId}`" @click="resolveReport(item.commentId, 'dismiss')">{{ t('comments.governance.dismiss') }}</a-button>
              <a-button v-if="Number(item.commentIsDelete) === 0" size="small" status="danger" :loading="busyKey === `report-hide-${item.commentId}`" @click="resolveReport(item.commentId, 'hide')">{{ t('comments.governance.hide') }}</a-button>
              <a-button v-else size="small" status="success" :loading="busyKey === `report-restore-${item.commentId}`" @click="resolveReport(item.commentId, 'restore')">{{ t('comments.governance.restore') }}</a-button>
            </a-space>
          </article>
        </div>
        <div v-else-if="!loading" class="queue-empty">{{ t('comments.governance.noReports') }}</div>
      </a-spin>
    </a-card>

    <a-card class="admin-panel" :bordered="false" data-testid="admin-comment-appeal-queue">
      <template #title>{{ t('comments.governance.appealsTitle') }}</template>
      <template #extra><span class="queue-total">{{ t('comments.governance.total', { total: appealTotal }) }}</span></template>
      <a-spin :loading="loading" style="width: 100%">
        <div v-if="appeals.length" class="queue-list">
          <article v-for="item in appeals" :key="item.id" class="queue-item">
            <div class="queue-item__main">
              <strong>#{{ item.commentId }} · {{ item.collectionTitle || t('comments.governance.unknownCollection') }}</strong>
              <p>{{ plainText(item.commentContent) || t('comments.governance.emptyContent') }}</p>
              <small>{{ item.appellantName || t('comments.governance.unknownUser') }} · {{ t('comments.governance.appealReason') }}：{{ item.reason || t('comments.governance.noReason') }}</small>
            </div>
            <a-space class="queue-item__actions">
              <a-button size="small" status="success" :loading="busyKey === `appeal-restore-${item.id}`" @click="resolveAppeal(item.id, 'restore')">{{ t('comments.governance.restore') }}</a-button>
              <a-button size="small" status="danger" :loading="busyKey === `appeal-reject-${item.id}`" @click="resolveAppeal(item.id, 'reject')">{{ t('comments.governance.reject') }}</a-button>
            </a-space>
          </article>
        </div>
        <div v-else-if="!loading" class="queue-empty">{{ t('comments.governance.noAppeals') }}</div>
      </a-spin>
    </a-card>
  </section>
</template>

<script setup lang="ts">
import { onMounted, ref } from 'vue'
import { Message } from '@arco-design/web-vue'
import {
  apiErrorMessage,
  listAdminCommentAppeals,
  listAdminCommentReports,
  resolveAdminCommentAppeal,
  resolveAdminCommentReports,
  type CommentAppealRow,
  type CommentReportGroupRow
} from '@/api/http'
import { t } from '@/i18n'
import { plainText } from '@/utils/format'

const loading = ref(false)
const busyKey = ref('')
const reports = ref<CommentReportGroupRow[]>([])
const appeals = ref<CommentAppealRow[]>([])
const reportTotal = ref(0)
const appealTotal = ref(0)

const reasonLabel = (reasons: unknown) => {
  const labels: Record<string, string> = {
    spam: t('comments.governance.reasonSpam'),
    harassment: t('comments.governance.reasonHarassment'),
    porn: t('comments.governance.reasonPorn'),
    illegal: t('comments.governance.reasonIllegal'),
    privacy: t('comments.governance.reasonPrivacy'),
    other: t('comments.governance.reasonOther')
  }
  const values = String(reasons || '').split(',').filter(Boolean)
  return values.length ? values.map((value) => labels[value] || value).join('、') : t('comments.governance.noReason')
}

async function load(): Promise<void> {
  loading.value = true
  try {
    const [reportPage, appealPage] = await Promise.all([
      listAdminCommentReports({ current: 1, size: 50 }),
      listAdminCommentAppeals({ current: 1, size: 50 })
    ])
    reports.value = reportPage.items ?? []
    reportTotal.value = Number(reportPage.total || 0)
    appeals.value = appealPage.items ?? []
    appealTotal.value = Number(appealPage.total || 0)
  } catch (error) {
    Message.error(apiErrorMessage(error, t('comments.governance.loadFailed')))
    reports.value = []
    appeals.value = []
    reportTotal.value = 0
    appealTotal.value = 0
  } finally {
    loading.value = false
  }
}

async function resolveReport(commentId: number, decision: 'dismiss' | 'hide' | 'restore'): Promise<void> {
  busyKey.value = `report-${decision}-${commentId}`
  try {
    await resolveAdminCommentReports(Number(commentId), decision)
    Message.success(t('comments.governance.reportHandled'))
    await load()
  } catch (error) {
    Message.error(apiErrorMessage(error, t('comments.governance.reportHandleFailed')))
  } finally {
    busyKey.value = ''
  }
}

async function resolveAppeal(appealId: number, decision: 'restore' | 'reject'): Promise<void> {
  busyKey.value = `appeal-${decision}-${appealId}`
  try {
    await resolveAdminCommentAppeal(Number(appealId), decision)
    Message.success(t('comments.governance.appealHandled'))
    await load()
  } catch (error) {
    Message.error(apiErrorMessage(error, t('comments.governance.appealHandleFailed')))
  } finally {
    busyKey.value = ''
  }
}

onMounted(() => void load())
</script>

<style scoped>
.comment-governance-queues { display: grid; gap: 16px; margin-top: 16px; }
.queue-total { color: var(--color-text-3); font-size: 12px; }
.queue-list { display: grid; gap: 10px; }
.queue-item { display: flex; align-items: flex-start; justify-content: space-between; gap: 16px; padding: 14px; border: 1px solid var(--color-border-2); border-radius: 10px; }
.queue-item__main { min-width: 0; }
.queue-item__main strong, .queue-item__main small { display: block; }
.queue-item__main p { margin: 7px 0; color: var(--color-text-1); line-height: 1.6; word-break: break-word; }
.queue-item__main small { color: var(--color-text-3); }
.queue-item__actions { flex: none; }
.queue-empty { padding: 34px 0; color: var(--color-text-3); text-align: center; }
@media (max-width: 760px) { .queue-item { flex-direction: column; } .queue-item__actions { width: 100%; } }
</style>
