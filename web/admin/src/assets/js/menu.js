import Layout from '@/layout/index.vue'
import router from '@/router'
import store from '@/store'
import axios from 'axios'
import Vue from 'vue'
import {
  getMenuPromise,
  isMenuLoaded,
  markMenuLoaded,
  setMenuPromise
} from './menu-state'

export function generaMenu() {
  if (isMenuLoaded()) {
    return Promise.resolve(store.state.userMenus)
  }
  if (getMenuPromise()) {
    return getMenuPromise()
  }

  const requestPromise = axios
    .get('/api/admin/user/menus')
    .then(({ data }) => {
      if (!data.flag) {
        throw new Error(data.message || '菜单加载失败')
      }

      const rawMenus = Array.isArray(data.data) ? data.data : []
      const viewMenus = rawMenus.map(normalizeMenuForView)
      const routes = rawMenus.map(createRoute)

      routes.forEach((route) => router.addRoute(route))
      store.commit('saveUserMenus', viewMenus)
      markMenuLoaded()
      return viewMenus
    })
    .catch((error) => {
      const message = error && error.message ? error.message : '菜单加载失败'
      Vue.prototype.$message.error(message)
      throw error
    })
    .finally(() => {
      setMenuPromise(null)
    })

  setMenuPromise(requestPromise)
  return requestPromise
}

function normalizeIcon(icon) {
  if (!icon || icon.startsWith('iconfont ')) {
    return icon || ''
  }
  return 'iconfont ' + icon
}

function normalizeMenuForView(menu) {
  return {
    ...menu,
    icon: normalizeIcon(menu.icon),
    children: Array.isArray(menu.children)
      ? menu.children.map((child) => ({
          ...child,
          icon: normalizeIcon(child.icon)
        }))
      : []
  }
}

function createRoute(menu) {
  if (!menu.path) {
    throw new Error(`菜单“${menu.name || '未命名菜单'}”缺少访问路径`)
  }

  const children = Array.isArray(menu.children) ? menu.children : []
  if (children.length === 0) {
    throw new Error(`菜单“${menu.name || '未命名菜单'}”缺少页面组件`)
  }

  const routeChildren = children.map((child) => {
    if (typeof child.path !== 'string' || typeof child.component !== 'string' || !child.component) {
      throw new Error(`菜单“${child.name || menu.name || '未命名菜单'}”的路由配置不完整`)
    }
    return {
      path: child.path,
      name: child.name,
      hidden: child.hidden,
      component: loadView(child.component)
    }
  })

  const isLeaf = routeChildren.length === 1 && routeChildren[0].path === ''
  const route = {
    path: menu.path,
    component: Layout,
    children: routeChildren
  }
  if (!isLeaf) {
    const firstVisibleChild = routeChildren.find((child) => !child.hidden && child.path)
    if (firstVisibleChild) {
      route.redirect = firstVisibleChild.path
    }
  }
  if (!isLeaf && menu.name) {
    route.name = menu.name
  }
  return route
}

export const loadView = (view) => {
  return (resolve) => require([`@/views${view}`], resolve)
}
