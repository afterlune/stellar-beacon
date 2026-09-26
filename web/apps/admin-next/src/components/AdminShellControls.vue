<template>
  <div class="admin-shell-controls" :class="{ 'is-labelled': labelled }">
    <a-tooltip :content="locale === 'zh-CN' ? t('shell.toEnglish') : t('shell.toChinese')" position="bottom">
      <button
        class="admin-icon-button admin-lang-button"
        type="button"
        :aria-label="`${t('login.language')} ${locale === 'zh-CN' ? 'EN' : '中'}${labelled ? ' ' + (locale === 'zh-CN' ? 'English' : '中文') : ''}`"
        data-testid="shell-locale-toggle"
        @click="toggleLocale">
        <span class="admin-lang-label">{{ locale === 'zh-CN' ? 'EN' : '中' }}</span>
        <span v-if="labelled" class="admin-control-text">{{ locale === 'zh-CN' ? 'English' : '中文' }}</span>
      </button>
    </a-tooltip>

    <a-tooltip :content="themeStore.theme === 'dark' ? t('shell.toLight') : t('shell.toDark')" position="bottom">
      <button
        class="admin-icon-button"
        type="button"
        :aria-label="themeStore.theme === 'dark' ? t('shell.toLight') : t('shell.toDark')"
        data-testid="shell-theme-toggle"
        @click="themeStore.toggle()">
        <IconSun v-if="themeStore.theme === 'dark'" />
        <IconMoon v-else />
      </button>
    </a-tooltip>
  </div>
</template>

<script setup lang="ts">
import { IconMoon, IconSun } from '@arco-design/web-vue/es/icon'

import { locale, t, toggleLocale } from '@/i18n'
import { useThemeStore } from '@/stores/theme'

// 顶栏与登录页共用：语言按钮在中文界面显示「EN」，英文界面显示「中」。
withDefaults(defineProps<{ labelled?: boolean }>(), { labelled: false })

const themeStore = useThemeStore()
</script>
