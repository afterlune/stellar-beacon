import { createRouter, createWebHistory, type RouteRecordRaw } from 'vue-router'

const routes: RouteRecordRaw[] = [
  {
    path: '/',
    name: 'Home',
    component: () => import('../views/Home.vue')
  },
  {
    path: '/articles/:articleId',
    name: 'Articles',
    component: () => import('../views/Article.vue')
  },
  {
    path: '/talks',
    name: 'talkList',
    component: () => import('../views/TalkList.vue')
  },
  {
    path: '/talks/:talkId',
    name: 'talks',
    component: () => import('../views/Talk.vue')
  },
  {
    path: '/search',
    name: 'Search',
    component: () => import('../views/Search.vue')
  },
  {
    path: '/reading',
    redirect: '/studio/library/reading',
    meta: { requiresAuth: true }
  },  {
    path: '/archives',
    name: 'Archives',
    component: () => import('../views/Archives.vue')
  },
  {
    path: '/article-list/:tagId',
    redirect: (to) => ({ path: `/tags/${to.params.tagId}`, query: to.query })
  },
  {
    path: '/tags',
    name: 'Tags',
    component: () => import('../views/Tags.vue')
  },
  {
    path: '/tags/:tagId',
    name: 'TagArticles',
    component: () => import('../views/ArticleList.vue')
  },
  {
    path: '/categories',
    name: 'Categories',
    component: () => import('../views/Categories.vue')
  },
  {
    path: '/categories/:categoryId',
    name: 'CategoryArticles',
    component: () => import('../views/ArticleList.vue')
  },
  {
    path: '/u/:handle',
    name: 'Author',
    component: () => import('../views/Author.vue')
  },
  {
    path: '/studio',
    component: () => import('../views/studio/StudioShell.vue'),
    meta: { requiresAuth: true, hideBanner: true },
    children: [
      { path: '', redirect: '/studio/dashboard' },
      { path: 'dashboard', name: 'StudioDashboard', component: () => import('../views/studio/StudioDashboard.vue') },
      { path: 'articles', name: 'StudioArticles', component: () => import('../views/studio/StudioContent.vue'), props: { kind: 'article' } },
      { path: 'talks', name: 'StudioTalks', component: () => import('../views/studio/StudioContent.vue'), props: { kind: 'talk' } },
      { path: 'series', name: 'StudioSeries', component: () => import('../views/studio/StudioContent.vue'), props: { kind: 'series' } },
      { path: 'topics', name: 'StudioTopics', component: () => import('../views/studio/StudioTopics.vue') },
      { path: 'library', redirect: '/studio/library/reading' },
      { path: 'library/reading', name: 'StudioReading', component: () => import('../views/Reading.vue') },
      { path: 'library/favorites', name: 'StudioFavorites', component: () => import('../views/Favorites.vue') }
    ]
  },  {
    path: '/about',
    name: 'About',
    component: () => import('../views/About.vue')
  },
  {
    path: '/message',
    name: 'Message',
    component: () => import('../views/Message.vue')
  },
  {
    path: '/friends',
    name: 'Friends',
    component: () => import('../views/FriendLink.vue')
  },
  {
    path: '/photos/:albumId',
    name: 'Photos',
    component: () => import('../views/Photos.vue')
  },
  {
    path: '/series',
    name: 'Series',
    component: () => import('../views/SeriesList.vue')
  },
  {
    path: '/series/:seriesId',
    name: 'SeriesDetail',
    component: () => import('../views/SeriesDetail.vue')
  },
  {
    path: '/favorites',
    redirect: '/studio/library/favorites',
    meta: { requiresAuth: true }
  },  {
    path: '/subscribe/confirm',
    name: 'SubscriptionConfirm',
    component: () => import('../views/SubscriptionConfirm.vue'),
    meta: { hideBanner: true }
  },
  {
    path: '/subscribe/unsubscribe',
    name: 'SubscriptionUnsubscribe',
    component: () => import('../views/SubscriptionUnsubscribe.vue'),
    meta: { hideBanner: true }
  },
  {
    path: '/404',
    name: '404',
    component: () => import('../views/404.vue'),
    meta: { hideBanner: true }
  },
  {
    path: '/:catchAll(.*)',
    redirect: '/404'
  }
]

const router = createRouter({
  history: createWebHistory(import.meta.env.BASE_URL),
  routes
})

export default router