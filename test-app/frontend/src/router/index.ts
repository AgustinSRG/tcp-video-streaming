import { createRouter, createWebHistory } from 'vue-router'
import Home from '../components/Home.vue'

const router = createRouter({
  history: createWebHistory(import.meta.env.BASE_URL),
  routes: [
    {
      path: '/',
      name: 'home',
      component: Home
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
