<template>
  <div class="sidebar-box profile-card" data-dia="platform">
    <div class="profile flex flex-col justify-center items-center">
      <BrandMark :size="72" />
      <h2 class="text-center pt-4 text-2xl font-semibold text-ob-bright">{{ websiteConfig.name || 'Stellar Beacon' }}</h2>
      <span class="brand-rule w-14 mt-2" />
      <p class="pt-5 w-full text-sm text-center text-ob-dim">{{ t('platform.description') }}</p>
      <ul class="profile-stats">
        <li>
          <span>{{ articleCount }}</span>
          <p>{{ t('settings.articles') }}</p>
        </li>
        <li>
          <span>{{ talkCount }}</span>
          <p>{{ t('settings.talks') }}</p>
        </li>
        <li>
          <span>{{ categoryCount }}</span>
          <p>{{ t('settings.categories') }}</p>
        </li>
        <li>
          <span>{{ tagCount }}</span>
          <p>{{ t('settings.tags') }}</p>
        </li>
      </ul>
    </div>
  </div>
</template>

<script lang="ts">
import { useAppStore } from '@/stores/app'
import { computed, defineComponent } from 'vue'
import { useI18n } from 'vue-i18n'
import BrandMark from '@/components/BrandMark.vue'

export default defineComponent({
  name: 'Profile',
  components: { BrandMark },
  setup() {
    const appStore = useAppStore()
    const { t } = useI18n()
    return {
      websiteConfig: computed(() => {
        return appStore.websiteConfig
      }),
      articleCount: computed(() => appStore.articleCount),
      talkCount: computed(() => appStore.talkCount),
      categoryCount: computed(() => appStore.categoryCount),
      tagCount: computed(() => appStore.tagCount),
      t
    }
  }
})
</script>

<style lang="scss" scoped>
.profile {
  padding-top: 8px;
}

/* Brand gradient around the profile avatar. */
.profile-avatar-ring {
  display: inline-flex;
  padding: 3px;
  border-radius: var(--radius-pill);
  background-image: var(--brand-gradient);

  img {
    width: 88px !important;
    height: 88px !important;
    margin: 0 !important;
    border-width: 0 !important;
    box-shadow: none !important;
    background-color: var(--surface-2);
  }
}

.profile-stats {
  display: grid;
  grid-template-columns: repeat(4, 1fr);
  width: 100%;
  margin-top: 20px;
  padding-top: 16px;
  border-top: 1px solid var(--border-hairline);

  li {
    position: relative;
    text-align: center;

    & + li::before {
      content: '';
      position: absolute;
      left: 0;
      top: 50%;
      transform: translateY(-50%);
      width: 1px;
      height: 20px;
      background: var(--border-hairline);
    }
  }

  span {
    display: block;
    font-size: 18px;
    font-weight: 600;
    line-height: 1.3;
    color: var(--text-bright);
  }

  p {
    font-size: 11px;
    line-height: 1.6;
    color: var(--text-dim);
  }
}
</style>
