<template>
  <teleport to="body">
    <div v-if="visible" class="admin-command-overlay" @click.self="close">
      <div class="admin-command" role="dialog" aria-modal="true" aria-label="快速跳转">
        <div class="admin-command-input">
          <IconSearch aria-hidden="true" />
          <input
            ref="inputRef"
            v-model="keyword"
            type="text"
            placeholder="搜索页面、菜单或操作…"
            aria-label="搜索页面、菜单或操作"
            @keydown.down.prevent="move(1)"
            @keydown.up.prevent="move(-1)"
            @keydown.enter.prevent="run()"
            @keydown.esc.prevent="close" />
          <kbd>ESC</kbd>
        </div>

        <div class="admin-command-list" role="listbox" aria-label="跳转结果">
          <template v-for="group in groupedResults" :key="group.label">
            <div class="admin-command-group">{{ group.label }}</div>
            <button
              v-for="item in group.items"
              :key="item.path"
              type="button"
              role="option"
              :aria-selected="item.path === activePath"
              class="admin-command-item"
              :class="{ 'is-active': item.path === activePath }"
              @click="go(item.path)"
              @mouseenter="activePath = item.path">
              <span class="admin-command-item-icon" aria-hidden="true"><component :is="item.icon" /></span>
              <span class="admin-command-item-copy">
                <strong>{{ item.title }}</strong>
                <small>{{ item.caption }}</small>
              </span>
              <IconRight v-if="item.path === activePath" aria-hidden="true" />
            </button>
          </template>
          <p v-if="results.length === 0" class="admin-command-empty">没有匹配的页面，换个关键词试试。</p>
        </div>

        <div class="admin-command-footer">
          <span><kbd>↑</kbd><kbd>↓</kbd> 选择</span>
          <span><kbd>↵</kbd> 打开</span>
          <span><kbd>ESC</kbd> 关闭</span>
        </div>
      </div>
    </div>
  </teleport>
</template>

<script setup lang="ts">
import { computed, nextTick, ref, watch } from 'vue'
import type { Component } from 'vue'
import { useRouter } from 'vue-router'
import { IconRight, IconSearch } from '@arco-design/web-vue/es/icon'
import { Message } from '@arco-design/web-vue'

import { useThemeStore } from '@/stores/theme'
import { menuItemPath, normalizeRoutePath, type NormalizedMenu } from '@/types'
import { menuIconFor } from '@/utils/menu-icon'

interface CommandItem {
  title: string
  caption: string
  path: string
  icon: Component
  group: string
  keywords: string
}

const props = defineProps<{ modelValue: boolean; menus: NormalizedMenu[] }>()
const emit = defineEmits<{ 'update:modelValue': [value: boolean] }>()

const router = useRouter()
const theme = useThemeStore()
const keyword = ref('')
const activePath = ref('')
const inputRef = ref<HTMLInputElement | null>(null)

const visible = computed({
  get: () => props.modelValue,
  set: (value: boolean) => emit('update:modelValue', value)
})

const pageItems = computed<CommandItem[]>(() => {
  const items: CommandItem[] = []
  const seen = new Set<string>()
  for (const menu of props.menus) {
    const children = menu.children.filter((child) => !child.hidden)
    if (children.length === 0) {
      if (menu.hidden) continue
      const path = normalizeRoutePath(menu.path)
      if (seen.has(path)) continue
      seen.add(path)
      items.push({
        title: menu.name,
        caption: path,
        path,
        icon: menuIconFor(menu),
        group: '页面',
        keywords: `${menu.name} ${path}`.toLowerCase()
      })
      continue
    }
    for (const child of children) {
      const path = menuItemPath(menu, child)
      if (seen.has(path)) continue
      seen.add(path)
      items.push({
        title: child.name,
        caption: `${menu.name} · ${path}`,
        path,
        icon: menuIconFor(child),
        group: '页面',
        keywords: `${menu.name} ${child.name} ${path}`.toLowerCase()
      })
    }
  }
  return items
})

const actionItems = computed<CommandItem[]>(() => [
  {
    title: theme.theme === 'dark' ? '切换为浅色主题' : '切换为深色主题',
    caption: '外观设置',
    path: '__action:theme',
    icon: menuIconFor({ name: '设置' }),
    group: '操作',
    keywords: 'theme 主题 外观 深色 浅色 dark light'
  },
  {
    title: '刷新当前页面数据',
    caption: '重新加载路由',
    path: '__action:reload',
    icon: menuIconFor({ name: '任务' }),
    group: '操作',
    keywords: 'reload 刷新 重载 refresh'
  }
])

const results = computed<CommandItem[]>(() => {
  const query = keyword.value.trim().toLowerCase()
  const pool = [...pageItems.value, ...actionItems.value]
  if (!query) return pool
  const terms = query.split(/\s+/).filter(Boolean)
  return pool.filter((item) => terms.every((term) => item.keywords.includes(term)))
})

const groupedResults = computed(() => {
  const groups: Array<{ label: string; items: CommandItem[] }> = []
  for (const item of results.value) {
    const bucket = groups.find((entry) => entry.label === item.group)
    if (bucket) bucket.items.push(item)
    else groups.push({ label: item.group, items: [item] })
  }
  return groups
})

watch(visible, async (value) => {
  if (!value) return
  keyword.value = ''
  await nextTick()
  inputRef.value?.focus()
  activePath.value = results.value[0]?.path || ''
})

watch(results, (value) => {
  if (!value.some((item) => item.path === activePath.value)) activePath.value = value[0]?.path || ''
})

function move(step: number): void {
  const items = results.value
  if (items.length === 0) return
  const index = items.findIndex((item) => item.path === activePath.value)
  const next = (index + step + items.length) % items.length
  activePath.value = items[next].path
}

function run(): void {
  const item = results.value.find((entry) => entry.path === activePath.value) || results.value[0]
  if (item) go(item.path)
}

function close(): void {
  visible.value = false
}

function go(path: string): void {
  close()
  if (path === '__action:theme') {
    theme.toggle()
    Message.success(theme.theme === 'dark' ? '已切换为深色主题' : '已切换为浅色主题')
    return
  }
  if (path === '__action:reload') {
    window.location.reload()
    return
  }
  void router.push(path)
}
</script>
