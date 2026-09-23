<template>
  <div class="comment-governance-actions">
    <template v-if="isHidden">
      <span class="comment-governance-actions__status">已被书单作者删除</span>
      <span v-if="appeal" class="comment-governance-actions__appeal-state">{{ appealStatusLabel }}</span>
      <button v-if="!appeal" type="button" class="comment-governance-actions__link" data-testid="comment-appeal-action" @click="appealOpen = !appealOpen">申诉</button>
      <button v-if="canEscalate" type="button" class="comment-governance-actions__link" data-testid="comment-appeal-escalate" :disabled="busy" @click="escalateAppeal">升级给管理员</button>
      <form v-if="appealOpen && !appeal" class="comment-governance-actions__form" @submit.prevent="submitAppeal">
        <textarea v-model.trim="appealReason" maxlength="500" rows="2" placeholder="说明为什么希望恢复这条评论（必填）" data-testid="comment-appeal-reason" />
        <div>
          <button type="submit" :disabled="busy || !appealReason" data-testid="comment-appeal-submit">提交申诉</button>
          <button type="button" :disabled="busy" @click="appealOpen = false">取消</button>
        </div>
      </form>
    </template>
    <template v-else-if="canReport">
      <button type="button" class="comment-governance-actions__link" data-testid="comment-report-action" @click="reportOpen = !reportOpen">举报</button>
      <form v-if="reportOpen" class="comment-governance-actions__form" @submit.prevent="submitReport">
        <select v-model="reportReason" aria-label="举报原因">
          <option value="spam">垃圾信息</option>
          <option value="harassment">骚扰或攻击</option>
          <option value="porn">色情低俗</option>
          <option value="illegal">违法内容</option>
          <option value="privacy">侵犯隐私</option>
          <option value="other">其他</option>
        </select>
        <textarea v-model.trim="reportDetail" maxlength="200" rows="2" placeholder="补充说明（选填）" data-testid="comment-report-detail" />
        <div>
          <button type="submit" :disabled="busy" data-testid="comment-report-submit">提交举报</button>
          <button type="button" :disabled="busy" @click="reportOpen = false">取消</button>
        </div>
      </form>
    </template>
    <span v-if="message" class="comment-governance-actions__message" :class="{ 'is-error': messageIsError }">{{ message }}</span>
  </div>
</template>

<script lang="ts">
import { computed, defineComponent, inject, ref } from 'vue'
import api from '@/api/api'
import emitter from '@/utils/mitt'

export default defineComponent({
  name: 'CommentGovernanceActions',
  props: {
    comment: { type: Object, required: true }
  },
  setup(props) {
    const readCollectionID = inject<() => number>('collectionId', () => 0)
    const readCurrentUserID = inject<() => number>('currentUserId', () => 0)
    const readLoggedIn = inject<() => boolean>('isLoggedIn', () => false)
    const readAppeal = inject<(commentID: number) => any>('commentAppeal', () => null)
    const reportOpen = ref(false)
    const appealOpen = ref(false)
    const busy = ref(false)
    const reportReason = ref('spam')
    const reportDetail = ref('')
    const appealReason = ref('')
    const message = ref('')
    const messageIsError = ref(false)

    const itemID = computed(() => Number((props.comment as any).id || 0))
    const isHidden = computed(() => Number((props.comment as any).isDelete || 0) === 1)
    const appeal = computed(() => readAppeal(itemID.value))
    const canReport = computed(() => {
      const authorID = Number((props.comment as any).userId || 0)
      return !isHidden.value && readLoggedIn() && Number(readCollectionID()) > 0 && Number(readCurrentUserID()) > 0 && Number(readCurrentUserID()) !== authorID
    })
    const canEscalate = computed(() => appeal.value?.status === 'rejected' && appeal.value?.stage === 'owner')
    const appealStatusLabel = computed(() => {
      const current = appeal.value
      if (!current) return ''
      if (current.status === 'pending' && current.stage === 'admin') return '申诉待管理员终审'
      if (current.status === 'pending') return '申诉待书单作者处理'
      if (current.status === 'restored') return '申诉已通过，评论已恢复'
      if (current.status === 'rejected' && current.stage === 'owner') return '书单作者已驳回，可升级给管理员'
      if (current.status === 'rejected') return '管理员已驳回申诉'
      return '申诉处理中'
    })

    const errorMessage = (reason: any, fallback: string) => reason?.response?.data?.message || reason?.message || fallback
    const ensureSuccess = (response: any) => {
      if (!response?.data?.flag) throw new Error(response?.data?.message || '操作失败')
    }
    const submitReport = async () => {
      const collectionID = Number(readCollectionID())
      if (!collectionID || !itemID.value) return
      busy.value = true
      message.value = ''
      try {
        ensureSuccess(await api.reportComment({ commentId: itemID.value, reason: reportReason.value, detail: reportDetail.value || undefined }))
        message.value = '举报已提交，作者会尽快处理。'
        messageIsError.value = false
        reportOpen.value = false
        reportDetail.value = ''
      } catch (reason: any) {
        message.value = errorMessage(reason, '举报提交失败')
        messageIsError.value = true
      } finally {
        busy.value = false
      }
    }
    const submitAppeal = async () => {
      if (!itemID.value || !appealReason.value) return
      busy.value = true
      message.value = ''
      try {
        ensureSuccess(await api.appealComment({ commentId: itemID.value, reason: appealReason.value }))
        message.value = '申诉已提交。'
        messageIsError.value = false
        appealOpen.value = false
        appealReason.value = ''
        emitter.emit('collectionAppealsRefresh')
      } catch (reason: any) {
        message.value = errorMessage(reason, '申诉提交失败')
        messageIsError.value = true
      } finally {
        busy.value = false
      }
    }
    const escalateAppeal = async () => {
      const appealID = Number(appeal.value?.id || 0)
      if (!appealID) return
      busy.value = true
      message.value = ''
      try {
        ensureSuccess(await api.escalateCommentAppeal(appealID))
        message.value = '已升级给管理员终审。'
        messageIsError.value = false
        emitter.emit('collectionAppealsRefresh')
      } catch (reason: any) {
        message.value = errorMessage(reason, '升级失败')
        messageIsError.value = true
      } finally {
        busy.value = false
      }
    }

    return {
      isHidden,
      appeal,
      canReport,
      canEscalate,
      appealStatusLabel,
      reportOpen,
      appealOpen,
      busy,
      reportReason,
      reportDetail,
      appealReason,
      message,
      messageIsError,
      submitReport,
      submitAppeal,
      escalateAppeal
    }
  }
})
</script>

<style scoped>
.comment-governance-actions { display: flex; flex-wrap: wrap; align-items: center; gap: 8px; }
.comment-governance-actions__link { padding: 0; border: 0; background: transparent; color: var(--text-accent); font: inherit; cursor: pointer; }
.comment-governance-actions__link:disabled { opacity: .55; cursor: wait; }
.comment-governance-actions__status { color: #ef8c7f; }
.comment-governance-actions__appeal-state { color: var(--text-ob-dim); }
.comment-governance-actions__form { flex-basis: 100%; display: grid; gap: 7px; min-width: min(100%, 360px); margin-top: 6px; padding: 10px; border: 1px solid var(--border-hairline); border-radius: 10px; background: var(--background-primary); }
.comment-governance-actions__form select, .comment-governance-actions__form textarea { box-sizing: border-box; width: 100%; padding: 7px 9px; border: 1px solid var(--border-hairline); border-radius: 8px; background: var(--background-primary-alt); color: inherit; font: inherit; font-size: 12px; }
.comment-governance-actions__form div { display: flex; gap: 8px; }
.comment-governance-actions__form button { padding: 5px 10px; border: 1px solid var(--border-hairline); border-radius: 999px; background: transparent; color: inherit; cursor: pointer; }
.comment-governance-actions__form button[type='submit'] { color: var(--text-accent); }
.comment-governance-actions__form button:disabled { opacity: .55; cursor: wait; }
.comment-governance-actions__message { flex-basis: 100%; color: var(--color-ob); font-size: 11px; }
.comment-governance-actions__message.is-error { color: #ef8c7f; }
</style>
