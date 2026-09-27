<script lang="ts">
import { defineComponent, h, type VNode } from 'vue'
import { isExternalIcon } from '@/utils/validate'

type SvgTreeNode = {
  tag: string
  attributes: Record<string, string>
  children: Array<SvgTreeNode | string>
}

const iconSources = import.meta.glob<string>('../../icons/svg/*.svg', {
  eager: true,
  query: '?raw',
  import: 'default'
})
const parsedIcons = new Map<string, SvgTreeNode | null>()

function parseSvg(source: string): SvgTreeNode | undefined {
  const markup = source.replace(/^\s*<\?xml[^>]*\?>\s*/i, '').replace(/<!DOCTYPE[\s\S]*?>/i, '')
  const parsed = new DOMParser().parseFromString(markup, 'image/svg+xml')
  const root = parsed.documentElement
  if (root.localName !== 'svg' || parsed.querySelector('parsererror')) return undefined

  const toTree = (element: Element): SvgTreeNode => {
    const attributes: Record<string, string> = {}
    for (const attribute of Array.from(element.attributes)) {
      attributes[attribute.name] = attribute.value
    }

    const children: Array<SvgTreeNode | string> = []
    for (const child of Array.from(element.childNodes)) {
      if (child.nodeType === Node.ELEMENT_NODE) {
        children.push(toTree(child as Element))
      } else if (
        (child.nodeType === Node.TEXT_NODE || child.nodeType === Node.CDATA_SECTION_NODE) &&
        child.nodeValue?.trim()
      ) {
        children.push(child.nodeValue)
      }
    }

    return { tag: element.tagName, attributes, children }
  }

  return toTree(root)
}

function getIconTree(name: string): SvgTreeNode | undefined {
  if (parsedIcons.has(name)) return parsedIcons.get(name) ?? undefined

  const source = iconSources[`../../icons/svg/${name}.svg`]
  const tree = source ? parseSvg(source) ?? null : null
  parsedIcons.set(name, tree)
  return tree ?? undefined
}

function renderSvgNode(node: SvgTreeNode): VNode {
  return h(
    node.tag,
    node.attributes,
    node.children.map((child) => (typeof child === 'string' ? child : renderSvgNode(child)))
  )
}

export default defineComponent({
  name: 'SvgIcon',
  inheritAttrs: false,
  props: {
    iconClass: {
      type: String,
      required: true
    },
    className: {
      type: String,
      default: ''
    }
  },
  setup(props, { attrs }) {
    return () => {
      if (isExternalIcon(props.iconClass)) {
        const mask = `url(${props.iconClass}) no-repeat 50% 50%`
        return h('div', {
          ...attrs,
          class: ['svg-external-icon', 'svg-icon', props.className, attrs.class],
          style: [{ mask, '-webkit-mask': mask }, attrs.style],
          'aria-hidden': attrs['aria-hidden'] ?? 'true'
        })
      }

      const tree = getIconTree(props.iconClass)
      if (!tree) {
        return h('svg', {
          ...attrs,
          class: ['svg-icon', props.className, attrs.class],
          'aria-hidden': attrs['aria-hidden'] ?? 'true'
        })
      }

      const { class: sourceClass, style: sourceStyle, ...sourceAttributes } = tree.attributes
      return h(
        'svg',
        {
          ...sourceAttributes,
          ...attrs,
          class: ['svg-icon', sourceClass, props.className, attrs.class],
          style: [sourceStyle, attrs.style],
          'aria-hidden': attrs['aria-hidden'] ?? 'true'
        },
        tree.children.map((child) => (typeof child === 'string' ? child : renderSvgNode(child)))
      )
    }
  }
})
</script>

<style scoped>
.svg-icon {
  width: 1em;
  height: 1em;
  vertical-align: -0.15em;
  fill: currentColor;
  stroke: var(--background-primary);
  overflow: hidden;
  display: inline;
  position: relative;
}

.svg-external-icon {
  background-color: currentColor;
  mask-size: cover !important;
  display: inline-block;
}
</style>
