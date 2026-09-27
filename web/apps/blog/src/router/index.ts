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
    path: '/authors',
    name: 'Authors',
    component: () => import('../views/Authors.vue')
  },
  {
    path: '/topics',
    name: 'Topics',
    component: () => import('../views/Topics.vue')
  },
  {
    path: '/collections',
    name: 'Collections',
    component: () => import('../views/Collections.vue')
  },
  {
    path: '/collections/:slug',
    name: 'CollectionDetail',
    component: () => import('../views/CollectionDetail.vue')
  },
  {
    path: '/for-you',
    name: 'ForYou',
    component: () => import('../views/ForYou.vue'),
    meta: { requiresAuth: true, hideBanner: true }
  },
  {
    path: '/following',
    name: 'Following',
    component: () => import('../views/Following.vue'),
    meta: { requiresAuth: true, hideBanner: true }
  },
  {
    path: '/notifications',
    name: 'Notifications',
    component: () => import('../views/Notifications.vue'),
    meta: { requiresAuth: true, hideBanner: true }
  },
  {
    path: '/studio',
    component: () => import('../views/studio/StudioShell.vue'),
    meta: { requiresAuth: true, hideBanner: true },
    children: [
      { path: '', redirect: '/studio/dashboard' },
      { path: 'dashboard', name: 'StudioDashboard', component: () => import('../views/studio/StudioDashboard.vue') },
      { path: 'profile', name: 'StudioProfile', component: () => import('../views/studio/StudioProfileView.vue') },
      { path: 'articles', name: 'StudioArticles', component: () => import('../views/studio/StudioContent.vue'), props: { kind: 'article' } },
      { path: 'articles/new', name: 'StudioArticleNew', component: () => import('../views/studio/StudioEditorView.vue'), props: { kind: 'article' } },
      { path: 'articles/:id/edit', name: 'StudioArticleEdit', component: () => import('../views/studio/StudioEditorView.vue'), props: { kind: 'article' } },
      { path: 'articles/:id/preview', name: 'StudioArticlePreview', component: () => import('../views/studio/StudioPreviewView.vue'), props: { kind: 'article' } },
      { path: 'talks', name: 'StudioTalks', component: () => import('../views/studio/StudioContent.vue'), props: { kind: 'talk' } },
      { path: 'talks/new', name: 'StudioTalkNew', component: () => import('../views/studio/StudioEditorView.vue'), props: { kind: 'talk' } },
      { path: 'talks/:id/edit', name: 'StudioTalkEdit', component: () => import('../views/studio/StudioEditorView.vue'), props: { kind: 'talk' } },
      { path: 'talks/:id/preview', name: 'StudioTalkPreview', component: () => import('../views/studio/StudioPreviewView.vue'), props: { kind: 'talk' } },
      { path: 'series', name: 'StudioSeries', component: () => import('../views/studio/StudioContent.vue'), props: { kind: 'series' } },
      { path: 'series/new', name: 'StudioSeriesNew', component: () => import('../views/studio/StudioEditorView.vue'), props: { kind: 'series' } },
      { path: 'series/:id/edit', name: 'StudioSeriesEdit', component: () => import('../views/studio/StudioEditorView.vue'), props: { kind: 'series' } },
      { path: 'series/:id/preview', name: 'StudioSeriesPreview', component: () => import('../views/studio/StudioPreviewView.vue'), props: { kind: 'series' } },
      { path: 'collections', name: 'StudioCollections', component: () => import('../views/studio/StudioCollections.vue') },
      { path: 'collections/:id/edit', name: 'StudioCollectionEdit', component: () => import('../views/studio/StudioCollectionEditor.vue') },
      { path: 'albums', name: 'StudioAlbums', component: () => import('../views/studio/StudioAlbumsView.vue') },
      { path: 'topics', name: 'StudioTopics', component: () => import('../views/studio/StudioTopics.vue') },
      { path: 'library', redirect: '/studio/library/reading' },
      { path: 'library/reading', name: 'StudioReading', component: () => import('../views/Reading.vue') },
      { path: 'library/favorites', name: 'StudioFavorites', component: () => import('../views/Favorites.vue') }
    ]
  },  {
    path: '/about',
    redirect: '/authors'
  },
  {
    path: '/message',
    redirect: '/authors'
  },
  {
    path: '/friends',
    redirect: '/authors'
  },
  {
    path: '/photos/:albumId',
    redirect: '/authors'
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
  routes,
  scrollBehavior(to, from, savedPosition) {
    if (savedPosition) return savedPosition
    if (to.hash) return { el: to.hash, behavior: 'smooth' }
    if (to.path.startsWith('/studio') && to.path !== from.path) return { top: 0 }
    return false
  }
})

export default router
