<template>
  <transition name="fade-bounce-y" mode="out-in">
    <div v-show="showDia" id="bot-container">
      <div id="Aurora-Dia--body">
        <div id="Aurora-Dia--tips-wrapper">
          <div id="Aurora-Dia--tips" class="Aurora-Dia--tips">早上好呀～</div>
        </div>
        <div id="Aurora-Dia" class="Aurora-Dia">
          <div id="Aurora-Dia--eyes" class="Aurora-Dia--eyes">
            <div id="Aurora-Dia--left-eye" class="Aurora-Dia--eye left"></div>
            <div id="Aurora-Dia--right-eye" class="Aurora-Dia--eye right"></div>
          </div>
        </div>
        <div class="Aurora-Dia--platform"></div>
      </div>
    </div>
  </transition>
</template>

<script lang="ts">
// @ts-nocheck
import { computed, defineComponent, onMounted, ref } from 'vue'
import { useDiaStore } from '@/stores/dia'
import { useAppStore } from '@/stores/app'

export default defineComponent({
  name: 'Dia',
  setup() {
    const diaStore = useDiaStore()
    const appStore = useAppStore()
    const showDia = ref(false)
    onMounted(() => {
      initializeBot()
    })
    const initializeBot = () => {
      if (!appStore.aurora_bot_enable) return
      diaStore.initializeBot({
        locale: diaStore.aurora_bot.locale,
        tips: diaStore.aurora_bot.tips
      })
      setTimeout(() => {
        showDia.value = true
      }, 1000)
    }
    return {
      showDia
    }
  }
})
</script>

<style lang="scss" scoped>
#bot-container {
  position: fixed;
  left: 22px;
  bottom: 0;
  z-index: 1000;
  width: 56px;
  height: 48px;
}
#Aurora-Dia--body {
  position: relative;
  display: flex;
  align-items: center;
  justify-content: center;
  flex-direction: column;
  width: 100%;
  height: 100%;
  --auora-dia--width: 52px; /* 110px */
  --auora-dia--height: 40px; /* 95px */
  --auora-dia--hover-height: 48px; /* 105px */
  --auora-dia--jump-1: 44px; /* 95px */
  --auora-dia--jump-2: 48px; /* 100px */
  --auora-dia--jump-3: 36px; /* 85px */
  --auora-dia--eye-top: 8px; /* 25px */
  --auora-dia--eye-height: 12px; /* 25px */
  --auora-dia--eye-width: 6.5px; /* 15px */
  --auora-dia--eye-top: 8px; /* 20px */
  --auora-dia--platform-size: var(--auora-dia--jump-2); /* 100px */
  --auora-dia--platform-size-shake-1: 60px; /* 115px */
  --auora-dia--platform-size-shake-2: 36px; /* 115px */
  --auora-dia--platform-top: -12px; /* 0 */
  --aurora-dia--linear-gradient: var(--brand-gradient); /* linear-gradient(to bottom, #5fc, #1a8) */
  --aurora-dia--linear-gradient-hover: linear-gradient(to bottom, #3aa9c4, #5433ff);
  --aurora-dia--platform-light: #ff4fa3;
}
.Aurora-Dia {
  position: absolute;
  bottom: 30px;
  width: var(--auora-dia--width);
  height: var(--auora-dia--height);
  border-radius: 45%;
  border: 4px solid var(--background-secondary);
  // box-shadow: 0 0 5px rgba(0, 0, 0, 0.5);
  animation: breathe-and-jump 3s linear infinite;
  cursor: pointer;
  z-index: 1;
}
.Aurora-Dia::before {
  content: '';
  position: absolute;
  top: -1px;
  left: -1px;
  width: calc(100% + 3px);
  height: calc(100% + 2px);
  background-color: #2cdcff;
  background: var(--aurora-dia--linear-gradient);
  /* Blur the background shape to soften its edge. */
  border-radius: 46% 54% 52% 48% / 52% 46% 54% 48%;
  filter: blur(7px);
  opacity: 0.92;
  transition: 0.3s linear all;
}
.Aurora-Dia.active {
  animation: deactivate 0.75s linear, bounce-then-breathe 5s linear infinite;
}

.Aurora-Dia--eyes > .Aurora-Dia--eye {
  position: absolute;
  top: var(--auora-dia--eye-top);
  width: var(--auora-dia--eye-width);
  height: var(--auora-dia--eye-height);
  border-radius: 15px;
  background-color: #fff;
  box-shadow: 0 0 7px rgba(255, 255, 255, 0.5);
  animation: blink 5s linear infinite;
}
.Aurora-Dia--eyes > .Aurora-Dia--eye.left {
  left: 25%;
}
.Aurora-Dia--eyes > .Aurora-Dia--eye.right {
  right: 25%;
}
.Aurora-Dia--eyes.moving > .Aurora-Dia--eye {
  animation: none;
}

.Aurora-Dia--platform {
  position: relative;
  top: 0;
  transform: rotateX(70deg);
  width: var(--auora-dia--platform-size);
  height: var(--auora-dia--platform-size);
  box-shadow: 0 0 var(--auora-dia--platform-size) var(--aurora-dia--platform-light),
    0 0 15px var(--aurora-dia--platform-light) inset;
  animation: jump-pulse 3s linear infinite;
}

.Aurora-Dia--platform {
  border-radius: 50%;
  transition: 0.2s linear all;
}

.Aurora-Dia:hover {
  animation: shake-to-alert 0.5s linear;
  height: var(--auora-dia--hover-height);
  transform: translateY(-7px);
}
.Aurora-Dia:hover::before {
  background: var(--aurora-dia--linear-gradient-hover);
}
.Aurora-Dia:hover,
.Aurora-Dia:hover > .Aurora-Dia--eyes > .Aurora--Dia-eye {
  border-color: var(--text-accent);
  box-shadow: 0 0 5px var(--text-accent);
}
.Aurora-Dia:hover + .Aurora-Dia--platform {
  box-shadow: 0 0 var(--auora-dia--platform-size) var(--text-accent), 0 0 15px var(--text-accent) inset;
  animation: shake-pulse 0.5s linear;
}

#Aurora-Dia--tips-wrapper {
  position: absolute;
  bottom: 72px;
  right: -132px;
  width: 210px;
  min-height: 54px;
  /* Keep the message surface opaque for readability. */
  background: var(--surface-solid);
  border: 1px solid var(--border-hairline);
  color: var(--text-normal);
  padding: 0;
  border-radius: 12px;
  box-shadow: var(--shadow-card);
  opacity: 0;
  animation: tips-breathe 3s linear infinite;
  transition: 0.3s linear opacity;
}

#Aurora-Dia--tips-wrapper.active {
  opacity: 0.94;
}

.Aurora-Dia--tips {
  position: relative;
  height: 100%;
  width: 100%;
  min-height: 54px;
  border-radius: 12px;
  padding: 10px 12px;
  font-size: 0.8rem;
  font-weight: 600;
  background: transparent;
  overflow: hidden;
  text-overflow: ellipsis;
}

.Aurora-Dia--tips > span {
  -webkit-background-clip: text;
  -webkit-text-fill-color: transparent;
  padding: 0 0.1rem;
  color: #7aa2f7;
  background-color: #7aa2f7;
  background-image: var(--strong-gradient);
}

@keyframes deactivate {
  0% {
    border-color: var(--text-sub-accent);
  }
  20%,
  60% {
    border-color: var(--text-accent);
  }
  40%,
  80%,
  100% {
    border-color: var(--background-secondary);
  }
}

@keyframes tips-breathe {
  0%,
  100% {
    transform: translateY(0);
  }
  50% {
    transform: translateY(-5px);
  }
}

@keyframes bounce-then-breathe {
  0%,
  5%,
  10%,
  15% {
    transform: translateY(0);
  }
  2.5%,
  7.5%,
  12.5% {
    transform: translateY(-15px);
  }
  20%,
  40%,
  60%,
  80%,
  100% {
    height: var(--auora-dia--jump-1);
    transform: translateY(0);
  }
  30%,
  50%,
  70%,
  90% {
    height: var(--auora-dia--jump-2);
    transform: translateY(-5px);
  }
}

@keyframes breathe-and-jump {
  0%,
  40%,
  80%,
  100% {
    height: var(--auora-dia--jump-1);
    transform: translateY(0);
  }
  20%,
  60%,
  70%,
  90% {
    height: var(--auora-dia--jump-2);
    transform: translateY(-5px);
  }
  85% {
    height: var(--auora-dia--jump-3);
    transform: translateY(-20px);
  }
}

@keyframes blink {
  0%,
  100% {
    transform: scale(1, 0.05);
  }
  5%,
  95% {
    transform: scale(1, 1);
  }
}

@keyframes jump-pulse {
  0%,
  40%,
  80%,
  100% {
    box-shadow: 0 0 18px rgba(255, 45, 149, 0.45), 0 0 30px rgba(109, 75, 255, 0.22);
  }
  20%,
  60%,
  70%,
  90% {
    box-shadow: 0 0 30px rgba(255, 45, 149, 0.5), 0 0 54px rgba(109, 75, 255, 0.3);
  }
  85% {
    box-shadow: 0 0 44px rgba(255, 45, 149, 0.55), 0 0 76px rgba(109, 75, 255, 0.34);
  }
}

@keyframes shake-to-alert {
  0%,
  20%,
  40%,
  60%,
  80%,
  100% {
    transform: rotate(0) translateY(-8px);
  }
  10%,
  25%,
  35%,
  50%,
  65% {
    transform: rotate(7deg) translateY(-8px);
  }
  15%,
  30%,
  45%,
  55%,
  70% {
    transform: roate(-7deg) translateY(-8px);
  }
}

@keyframes shake-pulse {
  0%,
  20%,
  40%,
  60%,
  80%,
  100% {
    box-shadow: 0 0 20px rgba(34, 211, 238, 0.5);
  }
  10%,
  25%,
  35%,
  50%,
  65% {
    box-shadow: 0 0 32px rgba(34, 211, 238, 0.55);
  }
  15%,
  30%,
  45%,
  55%,
  70% {
    box-shadow: 0 0 14px rgba(34, 211, 238, 0.45);
  }
}
</style>

<style lang="scss">
.Aurora-Dia--tips > span {
  color: var(--text-accent);
  padding: 0 0.05rem;
}
</style>
