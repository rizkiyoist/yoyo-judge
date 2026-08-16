import { createRouter, createWebHistory } from 'vue-router'
import { useAuthStore } from '../stores/auth'

const router = createRouter({
  history: createWebHistory(import.meta.env.BASE_URL),
  routes: [
    {
      path: '/login',
      name: 'login',
      component: () => import('../views/LoginView.vue'),
      meta: { public: true },
    },
    {
      path: '/',
      name: 'contests',
      component: () => import('../views/ContestListView.vue'),
    },
    {
      path: '/contests/:contestId/edit',
      name: 'contest-edit',
      component: () => import('../views/ContestEditView.vue'),
      props: true,
    },
    {
      path: '/contests/:contestId/judges',
      name: 'contest-judges',
      component: () => import('../views/JudgeManagementView.vue'),
      props: true,
    },
    {
      path: '/contests/:contestId/divisions/:divisionId/players',
      name: 'division-players',
      component: () => import('../views/PlayerRosterView.vue'),
      props: true,
    },
    {
      path: '/contests/:contestId/divisions/:divisionId/:stage(prelim|final)/score',
      name: 'score-entry',
      component: () => import('../views/ScoreEntryView.vue'),
      props: true,
    },
    {
      path: '/contests/:contestId/divisions/:divisionId/:stage(prelim|final)/results',
      name: 'results',
      component: () => import('../views/ResultsView.vue'),
      props: true,
    },
    {
      path: '/contests/:contestId/divisions/:divisionId/:stage(prelim|final)/override',
      name: 'score-override',
      component: () => import('../views/ScoreOverrideView.vue'),
      props: true,
    },
  ],
})

router.beforeEach(async (to) => {
  const auth = useAuthStore()
  await auth.init()
  if (!to.meta.public && !auth.user) {
    return { name: 'login', query: { redirect: to.fullPath } }
  }
  if (to.name === 'login' && auth.user) {
    return { name: 'contests' }
  }
  return true
})

export default router
