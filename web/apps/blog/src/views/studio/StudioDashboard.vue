<template>
  <div class="studio-dashboard">
    <header class="studio-page-head">
      <div><p>WORKSPACE / 01</p><h1>创作总览</h1><span>发布计划、运营结果和内容表现都集中在这里。</span></div>
      <router-link to="/studio/articles/new">写一篇文章 →</router-link>
    </header>

    <section v-if="showActivation" class="studio-activation" :class="{ 'is-collapsed': activationState.collapsed }" data-testid="studio-activation">
      <template v-if="!activationState.collapsed">
        <header>
          <div><p>ACTIVATION</p><h2>完成你的空间设置</h2><span>三步建立公开身份、写下第一条内容，并检查公开主页。</span></div>
          <div class="studio-activation__progress">
            <strong>{{ activationProgress.completed }}/{{ activationProgress.total }}</strong>
            <button type="button" @click="collapseActivation">稍后</button>
          </div>
        </header>
        <div class="studio-activation__steps">
          <article v-for="(step, index) in activationProgress.steps" :key="step.key" :class="{ done: step.done }" :data-step="step.key">
            <span class="studio-activation__index">{{ step.done ? '✓' : index + 1 }}</span>
            <div class="studio-activation__copy">
              <strong>{{ step.label }}</strong>
              <p>{{ step.description }}</p>
            </div>
            <div class="studio-activation__actions">
              <template v-if="step.key === 'identity'">
                <router-link to="/studio/profile">{{ step.done ? '查看资料' : '完善资料' }}</router-link>
              </template>
              <template v-else-if="step.key === 'content'">
                <router-link to="/studio/articles/new">写文章</router-link>
                <router-link to="/studio/talks/new">发随想</router-link>
              </template>
              <template v-else>
                <router-link v-if="validHandle" :to="`/u/${normalizedHandle}`" @click="markProfileVisited">预览主页</router-link>
                <router-link v-else to="/studio/profile">先设置 Handle</router-link>
              </template>
            </div>
          </article>
        </div>
      </template>
      <button v-else type="button" class="studio-activation__collapsed" @click="expandActivation">
        <span>继续空间设置 · {{ activationProgress.nextStep?.label || '完成最后一步' }}</span>
        <strong>{{ activationProgress.completed }}/{{ activationProgress.total }}</strong>
      </button>
    </section>

    <section class="studio-stats">
      <article v-for="stat in stats" :key="stat.key">
        <span>{{ stat.index }}</span>
        <strong>{{ dashboard[stat.key] || 0 }}</strong>
        <small>{{ stat.label }}</small>
      </article>
    </section>

    <section class="studio-panel studio-analytics-panel">
      <header>
        <div><p>OPERATIONS</p><h2>内容运营</h2></div>
        <div class="studio-range-tabs" aria-label="统计区间">
          <button v-for="item in ranges" :key="item.value" type="button" :class="{ active: range === item.value }" @click="changeRange(item.value)">{{ item.label }}</button>
        </div>
      </header>
      <p v-if="analyticsError" class="studio-inline-error">{{ analyticsError }}</p>
      <div class="studio-operation-cards">
        <article><span>区间发布</span><strong>{{ operations.publishedArticles || 0 }}</strong><small>已成功上架</small></article>
        <article><span>等待发布</span><strong>{{ operations.scheduledArticles || 0 }}</strong><small>当前定时队列</small></article>
        <article :class="{ 'is-danger': operations.failedNotifications > 0 }"><span>通知异常</span><strong>{{ operations.failedNotifications || 0 }}</strong><small>{{ operations.retryingNotifications || 0 }} 条自动重试中</small></article>
        <article><span>批量操作</span><strong>{{ operations.batchOperations || 0 }}</strong><small>区间内审计记录</small></article>
        <article><span>关注者</span><strong>{{ dashboard.followerCount || 0 }}</strong><small><router-link to="/following?tab=followers">查看关注关系</router-link></small></article>
        <article><span>关注中</span><strong>{{ dashboard.followingCount || 0 }}</strong><small><router-link to="/following?tab=following">管理关注</router-link></small></article>
      </div>
      <div class="studio-analytics-grid">
        <div class="studio-trend">
          <div class="studio-trend__head"><strong>阅读趋势</strong><span>{{ rangeLabel }}</span></div>
          <svg v-if="trend.length" viewBox="0 0 600 180" role="img" aria-label="阅读趋势图" preserveAspectRatio="none">
            <polyline class="studio-trend__area" :points="trendAreaPoints" />
            <polyline class="studio-trend__line" :points="trendPolyline" />
          </svg>
          <p v-else class="studio-schedule-empty">所选区间还没有阅读数据。</p>
          <div class="studio-trend__footer"><span>阅读 {{ performance.views || 0 }}</span><span>去重读者 {{ performance.uniqueReaders || 0 }}</span><span>完成率 {{ formatPercent(performance.completionRate) }}</span></div>
        </div>
        <div class="studio-ranking">
          <strong>内容表现 Top 5</strong>
          <ol v-if="topArticles.length">
            <li v-for="(item, index) in topArticles" :key="item.articleId">
              <router-link :to="`/studio/articles/${item.articleId}/preview`"><span>{{ index + 1 }}</span><strong>{{ item.title }}</strong><em>{{ item.views }} 阅读</em></router-link>
            </li>
          </ol>
          <p v-else class="studio-schedule-empty">暂无内容表现数据。</p>
        </div>
      </div>
      <div v-if="operations.lastRun" class="studio-job-status" :class="{ 'is-danger': operations.lastRun.status === 1 }">
        <span>发布任务最近运行</span>
        <strong>{{ formatDateTime(operations.lastRun.startedAt) }}</strong>
        <em>{{ operations.lastRun.status === 1 ? '失败' : '正常' }} · {{ operations.lastRun.message }}</em>
      </div>
    </section>

    <section class="studio-panel studio-calendar-panel">
      <header>
        <div><p>PUBLISH CALENDAR</p><h2>发布日历</h2></div>
        <div class="studio-calendar-nav">
          <button type="button" aria-label="上个月" @click="moveMonth(-1)">←</button>
          <strong>{{ monthLabel }}</strong>
          <button type="button" aria-label="下个月" @click="moveMonth(1)">→</button>
          <button type="button" @click="resetMonth">本月</button>
        </div>
      </header>
      <p v-if="calendarError" class="studio-inline-error">{{ calendarError }}</p>
      <div class="studio-calendar-week" aria-hidden="true">
        <span v-for="day in weekdays" :key="day">{{ day }}</span>
      </div>
      <div class="studio-calendar-grid">
        <div v-for="cell in calendarCells" :key="cell.key" class="studio-calendar-day" :class="{ 'is-outside': !cell.day, 'is-today': cell.isToday }">
          <span v-if="cell.day" class="studio-calendar-day__number">{{ cell.day.getDate() }}</span>
          <article v-for="event in cell.events" :key="`${event.articleId}-${event.scheduledAt}`" class="studio-calendar-event" :class="`state-${event.state}`" :title="event.lastError || event.title">
            <router-link :to="`/studio/articles/${event.articleId}/preview`">{{ event.title }}</router-link>
            <span>{{ eventStateLabel(event.state) }}</span>
            <button v-if="event.state === 'notification_failed'" type="button" @click="retryPublish(event)">重试</button>
          </article>
        </div>
      </div>
    </section>

    <section class="studio-panel studio-schedule-panel">
      <header>
        <div><p>PUBLISH QUEUE</p><h2>待发布队列</h2></div>
        <router-link v-if="scheduledTotal" to="/studio/articles?status=4">管理全部 {{ scheduledTotal }} 篇 →</router-link>
      </header>
      <div v-if="scheduledArticles.length" class="studio-schedule-list">
        <router-link v-for="item in scheduledArticles" :key="item.id" :to="`/studio/articles/${item.id}/edit`">
          <time>{{ formatDateTime(item.scheduledAt) }}</time>
          <strong>{{ item.articleTitle }}</strong>
          <span>编辑定时 →</span>
        </router-link>
      </div>
      <p v-else class="studio-schedule-empty">当前没有待发布的定时文章。</p>
    </section>

    <div class="studio-dashboard__grid">
      <section class="studio-panel studio-panel--profile">
        <header>
          <div><p>PUBLIC IDENTITY</p><h2>公开主页资料</h2></div>
          <span>{{ completion.completed }}/{{ completion.total }} 已完成</span>
        </header>
        <div class="studio-profile-summary">
          <img :src="profile.avatar || defaultAvatar" :alt="profile.nickname || '作者头像'" />
          <div class="studio-profile-summary__copy">
            <strong>{{ profile.nickname || '未设置昵称' }}</strong>
            <span>@{{ profile.handle || 'your-handle' }}</span>
            <p>{{ profile.intro || '还没有公开简介，补全后作者主页会更完整。' }}</p>
          </div>
          <div class="studio-profile-progress">
            <span><i :style="{ width: `${completion.completed * 25}%` }" /></span>
            <small>{{ completion.missing.length ? `还缺：${completion.missing.map((item) => item.label).join('、')}` : '公开身份已完整' }}</small>
          </div>
          <div class="studio-profile-actions">
            <router-link to="/studio/profile">{{ completion.missing.length ? '完善公开资料' : '编辑公开资料' }} →</router-link>
            <router-link v-if="validHandle" :to="`/u/${normalizedHandle}`">预览主页</router-link>
          </div>
        </div>
      </section>

      <section class="studio-panel">
        <header><div><p>QUICK START</p><h2>继续创作</h2></div></header>
        <div class="studio-quick">
          <router-link to="/studio/articles"><strong>文章工作台</strong><span>管理公开、私有、草稿与定时发布</span><em>01 →</em></router-link>
          <router-link to="/studio/talks"><strong>发布随想</strong><span>记录一条轻量的公开信号</span><em>02 →</em></router-link>
          <router-link to="/studio/series"><strong>组织系列</strong><span>把长期文章串成阅读路径</span><em>03 →</em></router-link>
        </div>
      </section>
    </div>
  </div>
</template>

<script lang="ts">
import { computed, defineComponent, onMounted, reactive, ref, watch } from 'vue'
import { notify } from '@/services/notifications'
import api from '@/api/api'
import type { StudioActivation, StudioActivationSync } from '@stellar-beacon/api-contract'
import { useAppStore } from '@/stores/app'
import { useUserStore } from '@/stores/user'
import {
  isValidStudioHandle,
  normalizeStudioHandle,
  readStudioActivationState,
  saveStudioActivationState,
  studioActivationProgress,
  studioProfileCompletion,
  type StudioActivationState,
  type StudioProfile
} from '@/utils/studioProfile'

type CalendarEvent = { articleId: number; title: string; scheduledAt: string; publishedAt?: string; state: string; lastError?: string }

function emptyStudioActivation(): StudioActivation {
  return {
    startedAt: '', collapsed: false, identityCompletedAt: '',
    contentCompletedAt: '', profileVisitedAt: '', completedAt: ''
  }
}

export default defineComponent({
  name: 'StudioDashboard',
  setup() {
    const userStore = useUserStore()
    const appStore = useAppStore()
    const dashboard = ref<Record<string, number>>({})
    const scheduledArticles = ref<any[]>([])
    const scheduledTotal = ref(0)
    const range = ref<'7d' | '30d' | '90d'>('30d')
    const analytics = ref<any>({})
    const analyticsError = ref('')
    const calendarEvents = ref<CalendarEvent[]>([])
    const calendarError = ref('')
    const monthCursor = ref(new Date(new Date().getFullYear(), new Date().getMonth(), 1))
    const profile = reactive<StudioProfile>({
      handle: normalizeStudioHandle(userStore.userInfo?.handle),
      nickname: userStore.userInfo?.nickname || '',
      avatar: userStore.userInfo?.avatar || '',
      intro: userStore.userInfo?.intro || '',
      website: userStore.userInfo?.website || '',
      about: '',
      links: []
    })
    const stats = [
      { key: 'articleCount', label: '全部文章', index: 'A' },
      { key: 'draftCount', label: '草稿', index: 'D' },
      { key: 'privateCount', label: '私有内容', index: 'P' },
      { key: 'talkCount', label: '随想', index: 'T' },
      { key: 'seriesCount', label: '系列', index: 'S' },
      { key: 'favoriteCount', label: '收藏', index: 'F' }
    ]
    const ranges = [
      { value: '7d' as const, label: '7 天' },
      { value: '30d' as const, label: '30 天' },
      { value: '90d' as const, label: '90 天' }
    ]
    const weekdays = ['一', '二', '三', '四', '五', '六', '日']
    const defaultAvatar = 'data:image/svg+xml,%3Csvg xmlns="http://www.w3.org/2000/svg" width="96" height="96"%3E%3Crect width="96" height="96" rx="48" fill="%23172554"/%3E%3Ccircle cx="48" cy="36" r="17" fill="%239bb8ff"/%3E%3Cpath d="M16 89c5-23 16-34 32-34s27 11 32 34" fill="%239bb8ff"/%3E%3C/svg%3E'
    const completion = computed(() => studioProfileCompletion(profile))
    const normalizedHandle = computed(() => normalizeStudioHandle(profile.handle))
    const validHandle = computed(() => isValidStudioHandle(normalizedHandle.value))
    const activationUserID = Number(userStore.userInfo?.userInfoId || userStore.userInfo?.id || 0) || 'current'
    const activationState = reactive<StudioActivationState>(readStudioActivationState(activationUserID))
    const serverActivation = ref<StudioActivation>(emptyStudioActivation())
    const dashboardLoaded = ref(false)
    const profileLoaded = ref(false)
    const activationProgress = computed(() => studioActivationProgress(profile, dashboard.value, activationState.profileVisited))
    const activationReady = computed(() => dashboardLoaded.value && profileLoaded.value)
    const showActivation = computed(() => activationReady.value && !activationState.completedAt && !activationProgress.value.isComplete)
    const persistActivation = () => saveStudioActivationState(activationUserID, activationState)
    const hasServerActivation = (value: StudioActivation) => Boolean(
      value.startedAt || value.identityCompletedAt || value.contentCompletedAt || value.profileVisitedAt || value.completedAt
    )
    const applyServerActivation = (value: StudioActivation) => {
      if (!hasServerActivation(value)) return
      serverActivation.value = value
      activationState.collapsed = value.collapsed
      activationState.profileVisited = Boolean(value.profileVisitedAt) || activationState.profileVisited
      activationState.startedAt = value.startedAt || activationState.startedAt
      activationState.completedAt = value.completedAt || activationState.completedAt
      persistActivation()
    }
    const activationSyncPayload = (): StudioActivationSync => {
      const progress = activationProgress.value
      const serverStarted = hasServerActivation(serverActivation.value)
      return {
        started: Boolean(activationState.startedAt || serverActivation.value.startedAt)
          || (!serverStarted && !activationState.completedAt && !progress.isComplete),
        collapsed: activationState.collapsed,
        identityComplete: Boolean(progress.steps.find((step) => step.key === 'identity')?.done),
        contentComplete: Boolean(progress.steps.find((step) => step.key === 'content')?.done),
        profileVisited: activationState.profileVisited,
        completed: Boolean(activationState.completedAt || progress.isComplete)
      }
    }
    let activationSyncQueue = Promise.resolve()
    let activationSyncVersion = 0
    const queueActivationSync = () => {
      if (!activationReady.value) return
      const version = ++activationSyncVersion
      const payload = activationSyncPayload()
      activationSyncQueue = activationSyncQueue.then(() => undefined, () => undefined).then(async () => {
        try {
          const response = await api.syncStudioActivation(payload)
          if (!response?.data?.flag || version !== activationSyncVersion) return
          applyServerActivation(response.data.data)
        } catch {
          // Local state remains usable while the next dashboard load retries sync.
        }
      })
    }
    const collapseActivation = () => { activationState.collapsed = true; persistActivation(); queueActivationSync() }
    const expandActivation = () => { activationState.collapsed = false; persistActivation(); queueActivationSync() }
    const markProfileVisited = () => { activationState.profileVisited = true; persistActivation(); queueActivationSync() }
    watch(activationProgress, (progress) => {
      if (!progress.isComplete || activationState.completedAt) return
      activationState.completedAt = new Date().toISOString()
      persistActivation()
      queueActivationSync()
    }, { immediate: true })
    watch(activationReady, (ready) => {
      if (!ready) return
      const progress = activationProgress.value
      if (!hasServerActivation(serverActivation.value) && !activationState.startedAt && !activationState.completedAt &&
        progress.profileComplete && progress.contentCount > 0) {
        activationState.completedAt = new Date().toISOString()
        persistActivation()
      }
      queueActivationSync()
    }, { immediate: true })

    const operations = computed(() => analytics.value.operations || {})
    const performance = computed(() => analytics.value.performance || {})
    const trend = computed<any[]>(() => Array.isArray(analytics.value.trend) ? analytics.value.trend : [])
    const topArticles = computed<any[]>(() => Array.isArray(analytics.value.topArticles) ? analytics.value.topArticles : [])
    const rangeLabel = computed(() => ranges.find((item) => item.value === range.value)?.label || '30 天')
    const monthLabel = computed(() => new Intl.DateTimeFormat('zh-CN', { year: 'numeric', month: 'long' }).format(monthCursor.value))
    const trendMax = computed(() => Math.max(1, ...trend.value.map((item) => Number(item.views || 0))))
    const trendPolyline = computed(() => trend.value.map((item, index) => `${trendPointX(index)},${165 - Number(item.views || 0) / trendMax.value * 130}`).join(' '))
    const trendAreaPoints = computed(() => trendPolyline.value ? `0,180 ${trendPolyline.value} 600,180` : '')
    const calendarCells = computed(() => {
      const year = monthCursor.value.getFullYear()
      const month = monthCursor.value.getMonth()
      const firstWeekday = (new Date(year, month, 1).getDay() + 6) % 7
      const days = new Date(year, month + 1, 0).getDate()
      const cells: Array<{ key: string; day: Date | null; isToday: boolean; events: CalendarEvent[] }> = []
      const today = new Date()
      for (let index = 0; index < firstWeekday; index += 1) cells.push({ key: `blank-${index}`, day: null, isToday: false, events: [] })
      for (let day = 1; day <= days; day += 1) {
        const current = new Date(year, month, day)
        const key = localDateKey(current)
        cells.push({ key, day: current, isToday: current.toDateString() === today.toDateString(), events: calendarEvents.value.filter((event) => localDateKey(new Date(event.publishedAt || event.scheduledAt)) === key) })
      }
      while (cells.length % 7 !== 0 || cells.length < 35) cells.push({ key: `tail-${cells.length}`, day: null, isToday: false, events: [] })
      return cells
    })

    function trendPointX(index: number): number {
      if (trend.value.length <= 1) return 300
      return index / (trend.value.length - 1) * 600
    }
    function localDateKey(value: Date): string {
      return `${value.getFullYear()}-${String(value.getMonth() + 1).padStart(2, '0')}-${String(value.getDate()).padStart(2, '0')}`
    }
    const formatDateTime = (value: string) => value ? new Intl.DateTimeFormat('zh-CN', { dateStyle: 'medium', timeStyle: 'short' }).format(new Date(value)) : '待定'
    const formatPercent = (value: number) => `${Number(value || 0).toFixed(Number(value || 0) % 1 ? 1 : 0)}%`
    const eventStateLabel = (state: string) => ({ scheduled: '待发布', published: '已发布', notification_failed: '通知失败', suppressed: '已抑制', overdue: '已逾期' } as Record<string, string>)[state] || state

    const loadDashboard = async () => {
      try {
        const response = await api.getStudioDashboard()
        const data = response?.data?.data || {}
        dashboard.value = data
        if (data.activation) applyServerActivation(data.activation)
        dashboardLoaded.value = true
      } catch {
        notify.error('创作数据加载失败')
      }
    }
    const loadAnalytics = async () => {
      analyticsError.value = ''
      try {
        const response = await api.getStudioAnalytics(range.value)
        if (!response?.data?.flag) throw new Error(response?.data?.message || '运营数据加载失败')
        analytics.value = response.data.data || {}
      } catch (reason: any) {
        analyticsError.value = reason?.response?.data?.message || reason?.message || '运营数据加载失败'
      }
    }
    const loadCalendar = async () => {
      calendarError.value = ''
      const start = new Date(monthCursor.value.getFullYear(), monthCursor.value.getMonth(), 1)
      const end = new Date(monthCursor.value.getFullYear(), monthCursor.value.getMonth() + 1, 1)
      try {
        const response = await api.getStudioCalendar(start.toISOString(), end.toISOString())
        if (!response?.data?.flag) throw new Error(response?.data?.message || '发布日历加载失败')
        calendarEvents.value = response.data.data?.events || []
      } catch (reason: any) {
        calendarError.value = reason?.response?.data?.message || reason?.message || '发布日历加载失败'
      }
    }
    const loadSchedule = async () => {
      try {
        const response = await api.getStudioArticles({ current: 1, size: 3, status: 4 })
        const data = response?.data?.data || {}
        scheduledArticles.value = Array.isArray(data.records) ? data.records : Array.isArray(data.items) ? data.items : []
        scheduledTotal.value = Number(data.count ?? data.total ?? 0)
      } catch {
        scheduledArticles.value = []
        scheduledTotal.value = 0
      }
    }
    const loadProfile = async () => {
      try {
        const response = await api.getStudioProfile()
        if (!response?.data?.flag) return
        Object.assign(profile, response.data.data || {})
        userStore.userInfo = { ...(userStore.userInfo || {}), ...response.data.data }
        profileLoaded.value = true
      } catch {
        // Keep the cached identity visible when the profile request is unavailable.
      }
    }
    const changeRange = (value: '7d' | '30d' | '90d') => { range.value = value; void loadAnalytics() }
    const moveMonth = (offset: number) => { monthCursor.value = new Date(monthCursor.value.getFullYear(), monthCursor.value.getMonth() + offset, 1); void loadCalendar() }
    const resetMonth = () => { const now = new Date(); monthCursor.value = new Date(now.getFullYear(), now.getMonth(), 1); void loadCalendar() }
    const retryPublish = async (event: CalendarEvent) => {
      try {
        const response = await api.retryStudioArticlePublication(event.articleId)
        if (!response?.data?.flag) throw new Error(response?.data?.message || '重试失败')
        notify.success('已重新加入通知重试队列')
        await Promise.all([loadCalendar(), loadAnalytics()])
      } catch (reason: any) {
        notify.error(reason?.response?.data?.message || reason?.message || '重试失败')
      }
    }

    onMounted(() => {
      void loadDashboard()
      void loadAnalytics()
      void loadCalendar()
      void loadSchedule()
      void loadProfile()
    })

    return {
      dashboard, scheduledArticles, scheduledTotal, range, ranges, rangeLabel, analytics, analyticsError,
      operations, performance, trend, topArticles, trendPolyline, trendAreaPoints,
      calendarEvents, calendarError, monthCursor, monthLabel, calendarCells, weekdays, moveMonth, resetMonth, retryPublish,
      formatDateTime, formatPercent, eventStateLabel, changeRange,
      profile, stats, defaultAvatar, completion, normalizedHandle, validHandle, activationState, activationProgress, showActivation, collapseActivation, expandActivation, markProfileVisited
    }
  }
})
</script>

<style lang="scss" scoped>
.studio-dashboard { min-width: 0; }
.studio-page-head { display: flex; align-items: flex-end; justify-content: space-between; gap: 20px; padding: 32px; border: 1px solid var(--border-hairline); border-radius: 20px; background: radial-gradient(circle at 85% 0, rgba(98, 76, 190, .22), transparent 38%), color-mix(in srgb, var(--background-primary-alt) 94%, transparent); }
.studio-page-head p, .studio-panel header p { margin: 0 0 8px; color: var(--color-ob); font-size: 10px; letter-spacing: .18em; }
.studio-page-head h1 { margin: 0 0 8px; font-size: clamp(2rem, 4vw, 3.4rem); letter-spacing: -.05em; }
.studio-page-head span, .studio-panel header > span { color: var(--text-ob-dim); font-size: 12px; }
.studio-page-head > a { padding: 10px 16px; border-radius: 999px; background: var(--color-ob); color: #081127; font-weight: 700; text-decoration: none; }
.studio-activation { margin-top: 18px; padding: clamp(20px, 3vw, 30px); border: 1px solid color-mix(in srgb, var(--color-ob) 32%, var(--border-hairline)); border-radius: 20px; background: radial-gradient(circle at 90% 0, color-mix(in srgb, var(--color-ob) 17%, transparent), transparent 36%), color-mix(in srgb, var(--background-primary-alt) 94%, transparent); }
.studio-activation > header { display: flex; align-items: flex-start; justify-content: space-between; gap: 18px; margin-bottom: 18px; }
.studio-activation h2 { margin: 0 0 7px; font-size: clamp(1.35rem, 2.5vw, 1.9rem); }
.studio-activation p { margin: 0; color: var(--text-ob-dim); font-size: 12px; line-height: 1.7; }
.studio-activation__progress { display: flex; align-items: center; gap: 12px; }
.studio-activation__progress strong { font-size: 1.5rem; }
.studio-activation__progress button { min-height: 34px; padding: 0 13px; border: 1px solid var(--border-hairline); border-radius: 999px; background: transparent; color: inherit; cursor: pointer; }
.studio-activation__steps { display: grid; grid-template-columns: repeat(3, minmax(0, 1fr)); gap: 12px; }
.studio-activation__steps article { position: relative; display: grid; grid-template-columns: 34px minmax(0, 1fr); gap: 11px; padding: 16px; border: 1px solid var(--border-hairline); border-radius: 15px; background: color-mix(in srgb, var(--background-primary) 72%, transparent); }
.studio-activation__steps article.done { border-color: color-mix(in srgb, #78d0bb 48%, var(--border-hairline)); }
.studio-activation__index { display: grid; width: 34px; height: 34px; place-items: center; border-radius: 50%; background: color-mix(in srgb, var(--color-ob) 15%, transparent); color: var(--color-ob); font-weight: 800; }
.studio-activation__steps article.done .studio-activation__index { background: color-mix(in srgb, #78d0bb 20%, transparent); color: #78d0bb; }
.studio-activation__copy strong { display: block; margin: 5px 0 6px; }
.studio-activation__actions { grid-column: 2; display: flex; flex-wrap: wrap; gap: 8px; margin-top: 4px; }
.studio-activation__actions a { display: inline-flex; min-height: 34px; align-items: center; padding: 0 12px; border: 1px solid color-mix(in srgb, var(--color-ob) 42%, transparent); border-radius: 999px; color: var(--color-ob); font-size: 11px; text-decoration: none; }
.studio-activation__actions a + a { border-color: var(--border-hairline); color: inherit; }
.studio-activation__collapsed { display: flex; width: 100%; min-height: 54px; align-items: center; justify-content: space-between; border: 0; background: transparent; color: inherit; font: inherit; cursor: pointer; }
.studio-activation__collapsed span { color: var(--text-ob-dim); }
.studio-activation__collapsed strong { color: var(--color-ob); }
@media (max-width: 900px) { .studio-activation__steps { grid-template-columns: 1fr; } }.studio-stats { display: grid; grid-template-columns: repeat(6, minmax(0, 1fr)); gap: 10px; margin: 18px 0; }
.studio-stats article { padding: 16px; border: 1px solid var(--border-hairline); border-radius: 14px; background: color-mix(in srgb, var(--background-primary-alt) 90%, transparent); }
.studio-stats span, .studio-stats small { display: block; color: var(--text-ob-dim); font-size: 10px; }
.studio-stats strong { display: block; margin: 8px 0 4px; font-size: 1.45rem; }
.studio-panel { padding: 24px; margin-bottom: 16px; border: 1px solid var(--border-hairline); border-radius: 18px; background: color-mix(in srgb, var(--background-primary-alt) 92%, transparent); }
.studio-panel > header { display: flex; align-items: flex-start; justify-content: space-between; gap: 16px; margin-bottom: 18px; }
.studio-panel h2 { margin: 0; }
.studio-range-tabs, .studio-calendar-nav { display: flex; gap: 7px; align-items: center; }
.studio-range-tabs button, .studio-calendar-nav button { padding: 7px 11px; border: 1px solid var(--border-hairline); border-radius: 999px; background: transparent; color: inherit; cursor: pointer; }
.studio-range-tabs button.active { border-color: var(--color-ob); color: var(--color-ob); }
.studio-operation-cards { display: grid; grid-template-columns: repeat(3, minmax(0, 1fr)); gap: 10px; }
.studio-operation-cards article { padding: 15px; border: 1px solid var(--border-hairline); border-radius: 13px; }
.studio-operation-cards span, .studio-operation-cards small { display: block; color: var(--text-ob-dim); font-size: 10px; } .studio-operation-cards small a { color: var(--color-ob); text-decoration: none; }
.studio-operation-cards strong { display: block; margin: 7px 0 4px; font-size: 1.5rem; }
.studio-operation-cards article.is-danger strong { color: #df8177; }
.studio-analytics-grid { display: grid; grid-template-columns: minmax(0, 1.35fr) minmax(260px, .65fr); gap: 16px; margin-top: 16px; }
.studio-trend, .studio-ranking { min-width: 0; padding: 16px; border: 1px solid var(--border-hairline); border-radius: 14px; }
.studio-trend__head, .studio-trend__footer { display: flex; justify-content: space-between; gap: 10px; color: var(--text-ob-dim); font-size: 11px; }
.studio-trend svg { width: 100%; height: 180px; margin-top: 8px; overflow: visible; }
.studio-trend__line { fill: none; stroke: #8ca9ff; stroke-width: 3; vector-effect: non-scaling-stroke; }
.studio-trend__area { fill: rgba(140, 169, 255, .13); stroke: none; }
.studio-trend__footer { flex-wrap: wrap; }
.studio-ranking > strong { display: block; margin-bottom: 10px; }
.studio-ranking ol { display: grid; gap: 7px; margin: 0; padding: 0; list-style: none; }
.studio-ranking a { display: grid; grid-template-columns: 22px minmax(0, 1fr) auto; gap: 8px; padding: 9px 0; border-bottom: 1px solid var(--border-hairline); color: inherit; text-decoration: none; }
.studio-ranking a span, .studio-ranking a em { color: var(--text-ob-dim); font-size: 10px; font-style: normal; }
.studio-ranking a strong { overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
.studio-job-status { display: flex; flex-wrap: wrap; gap: 10px; margin-top: 14px; padding: 11px 13px; border: 1px solid var(--border-hairline); border-radius: 11px; color: var(--text-ob-dim); font-size: 11px; }
.studio-job-status strong { color: #78d0bb; }
.studio-job-status.is-danger strong { color: #df8177; }
.studio-job-status em { margin-left: auto; font-style: normal; }
.studio-calendar-week, .studio-calendar-grid { display: grid; grid-template-columns: repeat(7, minmax(0, 1fr)); }
.studio-calendar-week { color: var(--text-ob-dim); font-size: 10px; text-align: center; }
.studio-calendar-week span { padding: 7px 0; }
.studio-calendar-grid { border-top: 1px solid var(--border-hairline); border-left: 1px solid var(--border-hairline); }
.studio-calendar-day { min-height: 116px; padding: 8px; border-right: 1px solid var(--border-hairline); border-bottom: 1px solid var(--border-hairline); }
.studio-calendar-day.is-outside { background: color-mix(in srgb, var(--background-primary) 60%, transparent); }
.studio-calendar-day.is-today { background: color-mix(in srgb, var(--color-ob) 7%, transparent); }
.studio-calendar-day__number { display: block; margin-bottom: 5px; color: var(--text-ob-dim); font-size: 10px; text-align: right; }
.studio-calendar-event { display: grid; gap: 2px; margin-bottom: 5px; padding: 5px 6px; border-left: 2px solid #78d0bb; border-radius: 6px; background: color-mix(in srgb, #78d0bb 8%, transparent); font-size: 9px; }
.studio-calendar-event a { overflow: hidden; color: inherit; text-decoration: none; text-overflow: ellipsis; white-space: nowrap; }
.studio-calendar-event span { color: var(--text-ob-dim); }
.studio-calendar-event button { justify-self: start; padding: 2px 5px; border: 1px solid currentColor; border-radius: 5px; background: transparent; color: inherit; font-size: 9px; cursor: pointer; }
.studio-calendar-event.state-notification_failed, .studio-calendar-event.state-overdue { border-color: #df8177; background: color-mix(in srgb, #df8177 8%, transparent); color: #df8177; }
.studio-calendar-event.state-suppressed { border-color: #bca0ef; background: color-mix(in srgb, #bca0ef 8%, transparent); }
.studio-calendar-event.state-scheduled { border-color: #e7a652; background: color-mix(in srgb, #e7a652 8%, transparent); }
.studio-inline-error { margin: 0 0 12px; color: #df8177; font-size: 12px; }
.studio-schedule-panel header a { color: var(--color-ob); font-size: 12px; text-decoration: none; }
.studio-schedule-list { display: grid; gap: 8px; }
.studio-schedule-list a { display: grid; grid-template-columns: 165px minmax(0, 1fr) auto; gap: 14px; align-items: center; padding: 12px 14px; border: 1px solid var(--border-hairline); border-radius: 12px; color: inherit; text-decoration: none; }
.studio-schedule-list time { color: var(--color-ob); font-size: 11px; }
.studio-schedule-list strong { overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
.studio-schedule-list span, .studio-schedule-empty { color: var(--text-ob-dim); font-size: 11px; }
.studio-dashboard__grid { display: grid; grid-template-columns: minmax(0, 1.25fr) minmax(280px, .75fr); gap: 16px; }
.studio-profile-summary { display: grid; grid-template-columns: 84px minmax(0, 1fr); gap: 16px; align-items: center; }
.studio-profile-summary > img { width: 84px; height: 84px; border: 1px solid color-mix(in srgb, var(--color-ob) 45%, transparent); border-radius: 50%; object-fit: cover; }
.studio-profile-summary__copy strong, .studio-profile-summary__copy span { display: block; }
.studio-profile-summary__copy strong { font-size: 1.2rem; }
.studio-profile-summary__copy span { margin-top: 3px; color: var(--color-ob); font-size: 11px; }
.studio-profile-summary__copy p { margin: 9px 0 0; color: var(--text-ob-dim); font-size: 12px; line-height: 1.65; }
.studio-profile-progress { grid-column: 1 / -1; display: grid; gap: 8px; }
.studio-profile-progress > span { height: 5px; overflow: hidden; border-radius: 999px; background: color-mix(in srgb, var(--text-ob-dim) 18%, transparent); }
.studio-profile-progress i { display: block; height: 100%; border-radius: inherit; background: var(--color-ob); transition: width .2s ease; }
.studio-profile-progress small { color: var(--text-ob-dim); font-size: 11px; }
.studio-profile-actions { grid-column: 1 / -1; display: flex; flex-wrap: wrap; gap: 10px; }
.studio-profile-actions a { color: var(--color-ob); font-size: 12px; text-decoration: none; }
.studio-quick { display: grid; gap: 10px; }
.studio-quick a { display: grid; grid-template-columns: 1fr auto; gap: 4px 12px; padding: 15px; border: 1px solid var(--border-hairline); border-radius: 13px; color: inherit; text-decoration: none; }
.studio-quick strong, .studio-quick span { display: block; }
.studio-quick span { grid-column: 1; color: var(--text-ob-dim); font-size: 11px; }
.studio-quick em { grid-column: 2; grid-row: 1 / span 2; align-self: center; color: var(--color-ob); font-style: normal; }
@media (max-width: 980px) { .studio-activation { margin-top: 18px; padding: clamp(20px, 3vw, 30px); border: 1px solid color-mix(in srgb, var(--color-ob) 32%, var(--border-hairline)); border-radius: 20px; background: radial-gradient(circle at 90% 0, color-mix(in srgb, var(--color-ob) 17%, transparent), transparent 36%), color-mix(in srgb, var(--background-primary-alt) 94%, transparent); }
.studio-activation > header { display: flex; align-items: flex-start; justify-content: space-between; gap: 18px; margin-bottom: 18px; }
.studio-activation h2 { margin: 0 0 7px; font-size: clamp(1.35rem, 2.5vw, 1.9rem); }
.studio-activation p { margin: 0; color: var(--text-ob-dim); font-size: 12px; line-height: 1.7; }
.studio-activation__progress { display: flex; align-items: center; gap: 12px; }
.studio-activation__progress strong { font-size: 1.5rem; }
.studio-activation__progress button { min-height: 34px; padding: 0 13px; border: 1px solid var(--border-hairline); border-radius: 999px; background: transparent; color: inherit; cursor: pointer; }
.studio-activation__steps { display: grid; grid-template-columns: repeat(3, minmax(0, 1fr)); gap: 12px; }
.studio-activation__steps article { position: relative; display: grid; grid-template-columns: 34px minmax(0, 1fr); gap: 11px; padding: 16px; border: 1px solid var(--border-hairline); border-radius: 15px; background: color-mix(in srgb, var(--background-primary) 72%, transparent); }
.studio-activation__steps article.done { border-color: color-mix(in srgb, #78d0bb 48%, var(--border-hairline)); }
.studio-activation__index { display: grid; width: 34px; height: 34px; place-items: center; border-radius: 50%; background: color-mix(in srgb, var(--color-ob) 15%, transparent); color: var(--color-ob); font-weight: 800; }
.studio-activation__steps article.done .studio-activation__index { background: color-mix(in srgb, #78d0bb 20%, transparent); color: #78d0bb; }
.studio-activation__copy strong { display: block; margin: 5px 0 6px; }
.studio-activation__actions { grid-column: 2; display: flex; flex-wrap: wrap; gap: 8px; margin-top: 4px; }
.studio-activation__actions a { display: inline-flex; min-height: 34px; align-items: center; padding: 0 12px; border: 1px solid color-mix(in srgb, var(--color-ob) 42%, transparent); border-radius: 999px; color: var(--color-ob); font-size: 11px; text-decoration: none; }
.studio-activation__actions a + a { border-color: var(--border-hairline); color: inherit; }
.studio-activation__collapsed { display: flex; width: 100%; min-height: 54px; align-items: center; justify-content: space-between; border: 0; background: transparent; color: inherit; font: inherit; cursor: pointer; }
.studio-activation__collapsed span { color: var(--text-ob-dim); }
.studio-activation__collapsed strong { color: var(--color-ob); }
@media (max-width: 900px) { .studio-activation__steps { grid-template-columns: 1fr; } }.studio-stats { grid-template-columns: repeat(3, minmax(0, 1fr)); } .studio-operation-cards { grid-template-columns: repeat(2, minmax(0, 1fr)); } .studio-analytics-grid, .studio-dashboard__grid { grid-template-columns: 1fr; } }
@media (max-width: 720px) { .studio-page-head { align-items: stretch; flex-direction: column; } .studio-calendar-day { min-height: 86px; padding: 5px; } .studio-calendar-event { padding: 4px; } .studio-calendar-event span { display: none; } .studio-operation-cards { grid-template-columns: 1fr 1fr; } .studio-schedule-list a { grid-template-columns: 1fr; } .studio-schedule-list span { display: none; } .studio-activation { margin-top: 18px; padding: clamp(20px, 3vw, 30px); border: 1px solid color-mix(in srgb, var(--color-ob) 32%, var(--border-hairline)); border-radius: 20px; background: radial-gradient(circle at 90% 0, color-mix(in srgb, var(--color-ob) 17%, transparent), transparent 36%), color-mix(in srgb, var(--background-primary-alt) 94%, transparent); }
.studio-activation > header { display: flex; align-items: flex-start; justify-content: space-between; gap: 18px; margin-bottom: 18px; }
.studio-activation h2 { margin: 0 0 7px; font-size: clamp(1.35rem, 2.5vw, 1.9rem); }
.studio-activation p { margin: 0; color: var(--text-ob-dim); font-size: 12px; line-height: 1.7; }
.studio-activation__progress { display: flex; align-items: center; gap: 12px; }
.studio-activation__progress strong { font-size: 1.5rem; }
.studio-activation__progress button { min-height: 34px; padding: 0 13px; border: 1px solid var(--border-hairline); border-radius: 999px; background: transparent; color: inherit; cursor: pointer; }
.studio-activation__steps { display: grid; grid-template-columns: repeat(3, minmax(0, 1fr)); gap: 12px; }
.studio-activation__steps article { position: relative; display: grid; grid-template-columns: 34px minmax(0, 1fr); gap: 11px; padding: 16px; border: 1px solid var(--border-hairline); border-radius: 15px; background: color-mix(in srgb, var(--background-primary) 72%, transparent); }
.studio-activation__steps article.done { border-color: color-mix(in srgb, #78d0bb 48%, var(--border-hairline)); }
.studio-activation__index { display: grid; width: 34px; height: 34px; place-items: center; border-radius: 50%; background: color-mix(in srgb, var(--color-ob) 15%, transparent); color: var(--color-ob); font-weight: 800; }
.studio-activation__steps article.done .studio-activation__index { background: color-mix(in srgb, #78d0bb 20%, transparent); color: #78d0bb; }
.studio-activation__copy strong { display: block; margin: 5px 0 6px; }
.studio-activation__actions { grid-column: 2; display: flex; flex-wrap: wrap; gap: 8px; margin-top: 4px; }
.studio-activation__actions a { display: inline-flex; min-height: 34px; align-items: center; padding: 0 12px; border: 1px solid color-mix(in srgb, var(--color-ob) 42%, transparent); border-radius: 999px; color: var(--color-ob); font-size: 11px; text-decoration: none; }
.studio-activation__actions a + a { border-color: var(--border-hairline); color: inherit; }
.studio-activation__collapsed { display: flex; width: 100%; min-height: 54px; align-items: center; justify-content: space-between; border: 0; background: transparent; color: inherit; font: inherit; cursor: pointer; }
.studio-activation__collapsed span { color: var(--text-ob-dim); }
.studio-activation__collapsed strong { color: var(--color-ob); }
@media (max-width: 900px) { .studio-activation__steps { grid-template-columns: 1fr; } }.studio-stats { grid-template-columns: repeat(2, minmax(0, 1fr)); } .studio-profile-summary { grid-template-columns: 1fr; } .studio-profile-summary > img { width: 72px; height: 72px; } .studio-profile-progress, .studio-profile-actions { grid-column: auto; } }
</style>
