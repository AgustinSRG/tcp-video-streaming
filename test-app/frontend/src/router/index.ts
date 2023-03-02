import { createRouter, createWebHashHistory } from 'vue-router'
import HomeView from '../components/HomeView.vue'

const router = createRouter({
  history: createWebHashHistory(),
  routes: [
    {
      path: '/',
      name: 'home',
      component: HomeView
    },
    {
      path: '/watch/:channel',
      name: 'watch',
      component: () => import('../components/StreamingWatch.vue')
    },
    {
      path: '/control/:channel',
      name: 'control',
      component: () => import('../components/StreamingControl.vue')
    },
    {
      path: '/watch/:channel/vod/:vod',
      name: 'watch-vod',
      component: () => import('../components/StreamingWatchVOD.vue')
    }
  ]
})

export default router
