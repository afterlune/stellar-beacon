<template>
  <section class="admin-page monitor-page">
    <AdminPageHeader title="实时监控" description="查看当前在线会话、最近操作和异常情况，快速掌握后台运行状态。">
      <template #actions>
        <a-space>
          <a-tag color="green"><span class="monitor-live-dot" />自动刷新 30 秒</a-tag>
          <a-button :loading="loading" @click="load">立即刷新</a-button>
        </a-space>
      </template>
    </AdminPageHeader>

    <a-alert v-if="errorMessage" type="warning" closable @close="errorMessage = ''">{{ errorMessage }}</a-alert>

    <div class="monitor-stat-grid">
      <a-card class="admin-card monitor-stat-card" :bordered="false">
        <span class="monitor-stat-label">当前在线用户</span>
        <strong>{{ onlineTotal }}</strong>
        <small>活跃登录会话</small>
      </a-card>
      <a-card class="admin-card monitor-stat-card" :bordered="false">
        <span class="monitor-stat-label">今日访问</span>
        <strong>{{ analytics.overview.todayViews }}</strong>
        <small>按日实时统计</small>
      </a-card>
      <a-card class="admin-card monitor-stat-card" :bordered="false">
        <span class="monitor-stat-label">近 7 日访问</span>
        <strong>{{ trendTotal }}</strong>
        <small>访问趋势合计</small>
      </a-card>
      <a-card class="admin-card monitor-stat-card" :bordered="false">
        <span class="monitor-stat-label">最近异常</span>
        <strong>{{ exceptionTotal }}</strong>
        <small>可继续进入异常日志定位</small>
      </a-card>
    </div>

    <div class="monitor-grid">
      <a-card class="admin-panel" :bordered="false" title="在线用户">
        <template #extra><a-button type="text" size="small" @click="router.push('/users/online')">查看全部</a-button></template>
        <div class="admin-table-shell">
          <a-table :data="onlineUsers" :columns="onlineColumns" :pagination="false" :loading="loading">
            <template #user="{ record }"><span>{{ record.nickname || record.username || '未命名用户' }}</span></template>
            <template #status><a-tag color="green">在线</a-tag></template>
            <template #time="{ record }">{{ formatDate(record.lastLoginTime) }}</template>
            <template #empty><a-empty description="当前没有在线用户" /></template>
          </a-table>
        </div>
      </a-card>

      <a-card class="admin-panel" :bordered="false" title="访问趋势">
        <AdminEChart :option="trendOption" height="300px" />
      </a-card>
    </div>

    <div class="monitor-grid monitor-grid-bottom">
      <a-card class="admin-panel" :bordered="false" title="最近操作">
        <template #extra><a-button type="text" size="small" @click="router.push('/logs/operation')">日志中心</a-button></template>
        <div class="monitor-log-list">
          <div v-for="record in operationLogs.slice(0, 8)" :key="String(record.id)" class="monitor-log-item">
            <span class="monitor-log-dot monitor-log-dot-success" />
            <div><strong>{{ String(record.optDesc || record.optModule || record.optUri || '后台操作') }}</strong><small>{{ formatDate(record.createTime) }} · {{ record.nickname || '管理员' }}</small></div>
          </div>
          <a-empty v-if="!operationLogs.length" description="暂无操作记录" />
        </div>
      </a-card>
      <a-card class="admin-panel" :bordered="false" title="异常提醒">
        <template #extra><a-button type="text" status="danger" size="small" @click="router.push('/logs/exception')">查看异常</a-button></template>
        <div class="monitor-log-list">
          <div v-for="record in exceptionLogs.slice(0, 8)" :key="String(record.id)" class="monitor-log-item">
            <span class="monitor-log-dot monitor-log-dot-error" />
            <div><strong>{{ String(record.exceptionInfo || record.optDesc || record.optUri || '请求异常') }}</strong><small>{{ formatDate(record.createTime) }}</small></div>
          </div>
          <a-empty v-if="!exceptionLogs.length" description="暂无异常记录" />
        </div>
      </a-card>
    </div>
  </section>
</template>

<script setup lang="ts">
import { computed, onBeforeUnmount, onMounted, reactive, ref } from 'vue'
import { useRouter } from 'vue-router'

import { apiErrorMessage, getAdminDashboardAnalytics, listAdminExceptionLogs, listAdminOnlineUsers, listAdminOperationLogs } from '@/api/http'
import AdminEChart from '@/components/AdminEChart.vue'
import AdminPageHeader from '@/components/AdminPageHeader.vue'
import type { AdminDashboardAnalytics, AdminUser } from '@benetnasch/api-contract'

const router = useRouter()
const loading = ref(false)
const errorMessage = ref('')
const onlineUsers = ref<AdminUser[]>([])
const operationLogs = ref<Record<string, unknown>[]>([])
const exceptionLogs = ref<Record<string, unknown>[]>([])
const onlineTotal = ref(0)
const exceptionTotal = ref(0)
const analytics = reactive<AdminDashboardAnalytics>({ range: '7d', unit: 'day', overview: { totalViews: 0, todayViews: 0, monthViews: 0, userCount: 0, articleCount: 0, messageCount: 0 }, trend: [], regions: [], categories: [], tags: [], articleRank: [], generatedAt: '' })
let timer: ReturnType<typeof setInterval> | undefined

const onlineColumns = [
  { title: '用户', dataIndex: 'nickname', slotName: 'user' },
  { title: 'IP', dataIndex: 'ipAddress', width: 150 },
  { title: '浏览器', dataIndex: 'browser', width: 150 },
  { title: '状态', dataIndex: 'status', slotName: 'status', width: 76 },
  { title: '最近登录', dataIndex: 'lastLoginTime', slotName: 'time', width: 170 }
]

const trendTotal = computed(() => analytics.trend.reduce((total, item) => total + Number(item.views || 0), 0))
const trendOption = computed(() => ({
  color: ['#4d9d83'],
  tooltip: { trigger: 'axis' },
  grid: { left: 14, right: 16, top: 22, bottom: 18, containLabel: true },
  xAxis: { type: 'category', boundaryGap: false, data: analytics.trend.map(item => item.period.slice(5)) },
  yAxis: { type: 'value', minInterval: 1 },
  series: [{ name: '访问量', type: 'line', smooth: true, areaStyle: { color: 'rgba(77,157,131,.13)' }, data: analytics.trend.map(item => item.views) }]
}))

onMounted(() => {
  void load()
  timer = setInterval(() => void load(), 30_000)
})

onBeforeUnmount(() => { if (timer) clearInterval(timer) })

async function load(): Promise<void> {
  loading.value = true
  errorMessage.value = ''
  try {
    const [online, operations, exceptions, dashboard] = await Promise.all([
      listAdminOnlineUsers({ current: 1, size: 8 }),
      listAdminOperationLogs({ current: 1, size: 8 }),
      listAdminExceptionLogs({ current: 1, size: 8 }),
      getAdminDashboardAnalytics('7d', 'visitors')
    ])
    onlineUsers.value = online.items
    onlineTotal.value = online.total
    operationLogs.value = operations.items
    exceptionLogs.value = exceptions.items
    exceptionTotal.value = exceptions.total
    Object.assign(analytics, dashboard)
  } catch (error) {
    errorMessage.value = apiErrorMessage(error, '监控数据加载失败')
  } finally {
    loading.value = false
  }
}

function formatDate(value: unknown): string {
  if (!value) return '—'
  const date = new Date(String(value))
  return Number.isNaN(date.getTime()) ? String(value) : date.toLocaleString('zh-CN', { hour12: false })
}
</script>

<style scoped>
.monitor-stat-grid { display: grid; grid-template-columns: repeat(4, minmax(0, 1fr)); gap: 14px; margin-bottom: 18px; }.monitor-stat-card { display: grid; gap: 5px; }.monitor-stat-label,.monitor-stat-card small { color: var(--admin-muted); font-size: 12px; }.monitor-stat-card strong { font-size: 28px; line-height: 1.1; }
.monitor-grid { display: grid; grid-template-columns: minmax(0, 1.25fr) minmax(320px, 1fr); gap: 18px; margin-bottom: 18px; }.monitor-grid-bottom { grid-template-columns: repeat(2, minmax(0, 1fr)); }
.monitor-live-dot { width: 7px; height: 7px; display: inline-block; margin-right: 5px; border-radius: 50%; background: #36b37e; box-shadow: 0 0 0 4px rgb(54 179 126 / 14%); }.monitor-log-list { display: grid; gap: 4px; }.monitor-log-item { display: flex; align-items: flex-start; gap: 10px; min-height: 52px; padding: 8px 0; border-bottom: 1px solid var(--admin-border); }.monitor-log-item > div { min-width: 0; }.monitor-log-item strong,.monitor-log-item small { display: block; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }.monitor-log-item strong { color: var(--admin-ink); font-size: 13px; }.monitor-log-item small { margin-top: 4px; color: var(--admin-muted); font-size: 11px; }.monitor-log-dot { width: 9px; height: 9px; flex: 0 0 auto; margin-top: 5px; border-radius: 50%; }.monitor-log-dot-success { background: #4d9d83; }.monitor-log-dot-error { background: #e27d6c; }
@media (max-width: 1100px) { .monitor-stat-grid { grid-template-columns: repeat(2, minmax(0, 1fr)); }.monitor-grid,.monitor-grid-bottom { grid-template-columns: 1fr; } }
@media (max-width: 520px) { .monitor-stat-grid { grid-template-columns: 1fr 1fr; }.monitor-stat-card strong { font-size: 22px; } }
</style>
