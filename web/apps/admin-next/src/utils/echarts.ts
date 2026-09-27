import { use } from 'echarts/core'
import { LineChart, MapChart, PieChart } from 'echarts/charts'
import {
  GeoComponent,
  GridComponent,
  LegendComponent,
  TooltipComponent,
  VisualMapComponent
} from 'echarts/components'
import { CanvasRenderer } from 'echarts/renderers'

use([
  LineChart,
  MapChart,
  PieChart,
  GeoComponent,
  GridComponent,
  LegendComponent,
  TooltipComponent,
  VisualMapComponent,
  CanvasRenderer
])

export { graphic, init, registerMap, registerTheme } from 'echarts/core'
export type { ECharts } from 'echarts/core'
