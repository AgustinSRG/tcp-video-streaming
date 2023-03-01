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
      // route level code-splitting
      // this generates a separate chunk (About.[hash].js) for this route
      // which is lazy-loaded when the route is visited.
      component: () => import('../components/StreamingControl.vue')
    }
  ]
})

export default router
