import { createRouter, createWebHistory } from 'vue-router'

const routes = [
  {
    path: '/login',
    name: 'Login',
    component: () => import('../views/Login.vue'),
    meta: { title: '登录', public: true }
  },
  {
    path: '/',
    name: 'Home',
    redirect: '/config/global'
  },
  {
    path: '/config/global',
    name: 'GlobalConfig',
    component: () => import('../views/GlobalConfig.vue'),
    meta: { title: '全局配置' }
  },
  {
    path: '/config/cluster',
    name: 'ClusterConfig',
    component: () => import('../views/ClusterConfig.vue'),
    meta: { title: '集群配置' }
  },
  {
    path: '/config/cluster/:cluster',
    name: 'ClusterDetail',
    component: () => import('../views/ClusterDetail.vue'),
    meta: { title: '集群详情' }
  },
  {
    path: '/config/cluster/:cluster/node/:node',
    name: 'NodeConfig',
    component: () => import('../views/NodeConfig.vue'),
    meta: { title: '节点配置' }
  },
  {
    path: '/history',
    name: 'History',
    component: () => import('../views/ConfigHistory.vue'),
    meta: { title: '配置历史' }
  },
  {
    path: '/settings',
    name: 'Settings',
    component: () => import('../views/Settings.vue'),
    meta: { title: '系统设置' }
  }
]

const router = createRouter({
  history: createWebHistory(),
  routes
})

// 登录状态检查
const isAuthenticated = () => {
  return localStorage.getItem('yaf_token') || sessionStorage.getItem('yaf_token')
}

// 路由守卫
router.beforeEach((to, from, next) => {
  // 公开页面不需要登录
  if (to.meta.public) {
    // 已登录用户访问登录页，跳转到首页
    if (to.name === 'Login' && isAuthenticated()) {
      next('/')
    } else {
      next()
    }
    return
  }
  
  // 需要登录的页面
  if (!isAuthenticated()) {
    next('/login')
  } else {
    next()
  }
})

export default router
