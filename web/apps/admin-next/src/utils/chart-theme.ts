import { graphic, registerTheme } from '@/utils/echarts'

/**
 * ECharts 主题桥接层。
 *
 * 图表默认自带一套高饱和彩虹色（ECharts 内置 dark 主题尤其明显），跟管理台的
 * 靛蓝令牌体系完全对不上，深浅两套主题下都会“跳出”画面。这里把 styles.css 里的
 * 语义色镜像到 ECharts，让图表和卡片、标签使用同一套颜色与文字层级。
 *
 * 注意：色值必须与 `src/styles.css` 的令牌保持一致，改一边就要改另一边。
 */

export type ChartTheme = 'light' | 'dark'

export const CHART_THEME_NAME: Record<ChartTheme, string> = {
  light: 'stellar-beacon-light',
  dark: 'stellar-beacon-dark'
}

/** 系列色板：品牌靛蓝领头，其余取自设计系统的语义色。 */
const SERIES: Record<ChartTheme, string[]> = {
  light: ['#4f6bd8', '#2f9e77', '#d9803f', '#7a5cd6', '#3b82c4', '#d9584a', '#0f9ba8', '#b45309'],
  dark: ['#7d95ff', '#34d399', '#fbbf24', '#a78bfa', '#38bdf8', '#f87171', '#2dd4bf', '#fb923c']
}

interface Skin {
  ink: string
  muted: string
  faint: string
  split: string
  axis: string
  tooltipBg: string
  tooltipBorder: string
  tooltipInk: string
  shadow: string
}

const SKIN: Record<ChartTheme, Skin> = {
  light: {
    ink: '#33415c',
    muted: '#64748b',
    faint: '#94a3b8',
    split: '#eef1f6',
    axis: '#dde3ec',
    tooltipBg: 'rgba(255, 255, 255, 0.97)',
    tooltipBorder: '#e4e9f2',
    tooltipInk: '#1e2b3f',
    shadow: '0 18px 40px -18px rgb(15 23 42 / 28%)'
  },
  dark: {
    ink: '#c3cde9',
    muted: '#8b97ba',
    faint: '#5a6689',
    split: 'rgb(125 149 255 / 12%)',
    axis: 'rgb(125 149 255 / 22%)',
    tooltipBg: 'rgba(14, 20, 36, 0.96)',
    tooltipBorder: '#2c3560',
    tooltipInk: '#e6ebfa',
    shadow: '0 22px 48px -18px rgb(0 0 0 / 78%)'
  }
}

const FONT_FAMILY =
  'Inter, "Noto Sans SC", "PingFang SC", "Microsoft YaHei", ui-sans-serif, system-ui, -apple-system, "Segoe UI", sans-serif'

const registered = new Set<ChartTheme>()

/** 主题只注册一次；返回可直接传给 `echarts.init` 的主题名。 */
export function ensureChartTheme(theme: ChartTheme): string {
  if (!registered.has(theme)) {
    registerTheme(CHART_THEME_NAME[theme], buildTheme(theme))
    registered.add(theme)
  }
  return CHART_THEME_NAME[theme]
}

/** 取系列色，索引超出色板长度时循环复用。 */
export function chartSeriesColor(theme: ChartTheme, index = 0): string {
  const palette = SERIES[theme]
  return palette[index % palette.length]
}

/** 十六进制色值转 rgba，供渐变使用。 */
export function withAlpha(hex: string, alpha: number): string {
  const raw = hex.replace('#', '')
  const full = raw.length === 3
    ? raw.split('').map((char) => char + char).join('')
    : raw
  const r = Number.parseInt(full.slice(0, 2), 16)
  const g = Number.parseInt(full.slice(2, 4), 16)
  const b = Number.parseInt(full.slice(4, 6), 16)
  return `rgba(${r}, ${g}, ${b}, ${alpha})`
}

/** 面积图的纵向渐变：顶部浓、底部透明，比纯色块更有体积感。 */
export function verticalFade(color: string, from = 0.3, to = 0.01): graphic.LinearGradient {
  return new graphic.LinearGradient(0, 0, 0, 1, [
    { offset: 0, color: withAlpha(color, from) },
    { offset: 1, color: withAlpha(color, to) }
  ])
}

function buildTheme(theme: ChartTheme): Record<string, unknown> {
  const skin = SKIN[theme]

  return {
    color: SERIES[theme],
    // 卡片自己带背景，图表必须透明，否则会出现第二层“卡片中的卡片”。
    backgroundColor: 'transparent',
    textStyle: { fontFamily: FONT_FAMILY, color: skin.ink, fontSize: 12 },
    title: {
      textStyle: { color: skin.ink, fontSize: 22, fontWeight: 700 },
      subtextStyle: { color: skin.faint, fontSize: 11 }
    },
    legend: {
      textStyle: { color: skin.muted, fontSize: 11 },
      itemWidth: 8,
      itemHeight: 8,
      icon: 'circle',
      inactiveColor: skin.faint
    },
    tooltip: {
      backgroundColor: skin.tooltipBg,
      borderColor: skin.tooltipBorder,
      borderWidth: 1,
      padding: [9, 12],
      textStyle: { color: skin.tooltipInk, fontSize: 12 },
      extraCssText: `border-radius: 10px; box-shadow: ${skin.shadow}; backdrop-filter: blur(6px);`,
      axisPointer: {
        lineStyle: { color: skin.axis, width: 1, type: 'solid' },
        crossStyle: { color: skin.axis },
        shadowStyle: { color: theme === 'dark' ? 'rgb(125 149 255 / 6%)' : 'rgb(79 107 216 / 6%)' }
      }
    },
    grid: {
      left: 8,
      right: 16,
      top: 24,
      bottom: 6,
      containLabel: true
    },
    categoryAxis: {
      axisLine: { show: true, lineStyle: { color: skin.axis } },
      axisTick: { show: false },
      axisLabel: { color: skin.muted, fontSize: 11, margin: 12 },
      splitLine: { show: false },
      splitArea: { show: false }
    },
    valueAxis: {
      axisLine: { show: false },
      axisTick: { show: false },
      axisLabel: { color: skin.faint, fontSize: 11, margin: 10 },
      splitLine: { show: true, lineStyle: { color: skin.split, type: 'dashed' } },
      splitArea: { show: false }
    },
    line: {
      itemStyle: { borderWidth: 2 },
      lineStyle: { width: 2.4 },
      symbolSize: 6,
      symbol: 'circle',
      smooth: 0.35
    },
    pie: {
      itemStyle: { borderColor: theme === 'dark' ? '#0e1424' : '#ffffff', borderWidth: 2 },
      label: { color: skin.muted, fontSize: 11 },
      labelLine: { lineStyle: { color: skin.axis } }
    },
    map: {
      itemStyle: { areaColor: theme === 'dark' ? '#151d31' : '#eef1f7', borderColor: skin.axis },
      label: { color: skin.muted }
    }
  }
}
