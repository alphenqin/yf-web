<template>
  <!-- 登录页面全屏显示 -->
  <router-view v-if="$route.meta.public" />
  
  <!-- 主应用布局 -->
  <div v-else class="app-container">
    <aside class="sidebar">
      <div class="logo">
        <div class="logo-icon">
          <el-icon :size="28"><Monitor /></el-icon>
        </div>
        <h1>YAF 配置中心</h1>
      </div>
      
      <nav class="nav-menu">
        <router-link to="/config/global" class="nav-item" :class="{ active: $route.path === '/config/global' }">
          <el-icon><Setting /></el-icon>
          <span>全局配置</span>
        </router-link>
        <router-link to="/config/cluster" class="nav-item" :class="{ active: $route.path.startsWith('/config/cluster') }">
          <el-icon><Grid /></el-icon>
          <span>集群配置</span>
        </router-link>
        <router-link to="/history" class="nav-item" :class="{ active: $route.path === '/history' }">
          <el-icon><Clock /></el-icon>
          <span>配置历史</span>
        </router-link>
        <router-link to="/settings" class="nav-item" :class="{ active: $route.path === '/settings' }">
          <el-icon><Tools /></el-icon>
          <span>系统设置</span>
        </router-link>
      </nav>
      
      <div class="sidebar-footer">
        <div class="user-info" @click="handleLogout">
          <el-icon><User /></el-icon>
          <span>{{ currentUser }}</span>
          <el-icon class="logout-icon"><SwitchButton /></el-icon>
        </div>
        <div class="version-info">
          <span class="mono">v1.0.0</span>
        </div>
      </div>
    </aside>
    
    <main class="main-content">
      <header class="page-header">
        <div class="breadcrumb">
          <el-breadcrumb separator="/">
            <el-breadcrumb-item :to="{ path: '/' }">首页</el-breadcrumb-item>
            <el-breadcrumb-item v-if="$route.meta.title">{{ $route.meta.title }}</el-breadcrumb-item>
          </el-breadcrumb>
        </div>
        <div class="header-actions">
          <div class="status-badge" :class="{ connected: zkStatus.connected, disconnected: !zkStatus.connected }">
            <span class="status-dot"></span>
            <span class="status-text">ZooKeeper</span>
            <span class="status-label">{{ zkStatus.label }}</span>
          </div>
        </div>
      </header>
      
      <div class="page-content">
        <router-view v-slot="{ Component }">
          <transition name="fade" mode="out-in">
            <component :is="Component" />
          </transition>
        </router-view>
      </div>
    </main>
  </div>
</template>

<script setup>
import { ref, computed, onMounted, onUnmounted } from 'vue'
import { useRouter } from 'vue-router'
import { ElMessage, ElMessageBox } from 'element-plus'
import { useConfigStore } from './stores/config'
import { getSystemStatus } from './api/config'

const router = useRouter()
const configStore = useConfigStore()

// 当前用户
const currentUser = computed(() => {
  return localStorage.getItem('yaf_user') || sessionStorage.getItem('yaf_user') || 'admin'
})

// ZK 连接状态
const zkStatus = ref({
  connected: false,
  label: '检测中...',
  state: ''
})

let statusTimer = null

// 检查系统状态
const checkStatus = async () => {
  try {
    const res = await getSystemStatus()
    const zk = res.data.zookeeper
    zkStatus.value = {
      connected: zk.connected,
      label: zk.connected ? '已连接' : '未连接',
      state: zk.state
    }
  } catch (error) {
    zkStatus.value = {
      connected: false,
      label: '服务离线',
      state: 'error'
    }
  }
}

// 退出登录
const handleLogout = () => {
  ElMessageBox.confirm(
    '确定要退出登录吗？',
    '退出确认',
    {
      confirmButtonText: '退出',
      cancelButtonText: '取消',
      type: 'warning',
    }
  ).then(() => {
    localStorage.removeItem('yaf_token')
    localStorage.removeItem('yaf_user')
    sessionStorage.removeItem('yaf_token')
    sessionStorage.removeItem('yaf_user')
    ElMessage.success('已退出登录')
    router.push('/login')
  }).catch(() => {})
}

onMounted(() => {
  configStore.init()
  // 立即检查一次
  checkStatus()
  // 每 5 秒检查一次
  statusTimer = setInterval(checkStatus, 5000)
})

onUnmounted(() => {
  if (statusTimer) {
    clearInterval(statusTimer)
  }
})
</script>

<style lang="scss" scoped>
.app-container {
  display: flex;
  min-height: 100vh;
  background: var(--color-bg-primary);
}

.sidebar {
  width: 260px;
  background: var(--color-bg-secondary);
  border-right: 1px solid var(--color-border);
  display: flex;
  flex-direction: column;
  position: fixed;
  top: 0;
  left: 0;
  height: 100vh;
  z-index: 100;
}

.logo {
  height: 56px;
  padding: 0 16px 0 18px;
  display: flex;
  align-items: center;
  gap: 10px;
  border-bottom: 1px solid var(--color-border);
  
  .logo-icon {
    width: 32px;
    height: 32px;
    min-width: 32px;
    background: linear-gradient(135deg, #58a6ff 0%, #1f6feb 100%);
    border-radius: 8px;
    display: flex;
    align-items: center;
    justify-content: center;
    color: white;
    box-shadow: 0 4px 12px rgba(88, 166, 255, 0.3);
    
    .el-icon {
      font-size: 18px;
    }
  }
  
  h1 {
    font-size: 16px;
    font-weight: 600;
    color: var(--color-text-primary);
    letter-spacing: -0.3px;
    white-space: nowrap;
  }
}

.nav-menu {
  padding: 32px 8px 12px 8px;
  flex: 1;
  display: flex;
  flex-direction: column;
  gap: 2px;
}

.nav-item {
  display: flex;
  align-items: center;
  gap: 10px;
  padding: 10px 12px 10px 24px;
  border-radius: var(--radius-md);
  color: var(--color-text-secondary);
  text-decoration: none;
  transition: all 0.2s ease;
  font-size: 14px;
  font-weight: 500;
  
  .el-icon {
    font-size: 18px;
    min-width: 18px;
  }
  
  &:hover {
    background: var(--color-bg-tertiary);
    color: var(--color-text-primary);
  }
  
  &.active {
    background: rgba(88, 166, 255, 0.15);
    color: var(--color-accent);
    
    .el-icon {
      color: var(--color-accent);
    }
  }
}

.sidebar-footer {
  padding: 12px;
  border-top: 1px solid var(--color-border);
  
  .user-info {
    display: flex;
    align-items: center;
    gap: 10px;
    padding: 10px 12px;
    background: var(--color-bg-tertiary);
    border-radius: var(--radius-md);
    cursor: pointer;
    transition: all 0.2s ease;
    color: var(--color-text-secondary);
    font-size: 14px;
    margin-bottom: 10px;
    
    &:hover {
      background: rgba(248, 81, 73, 0.1);
      color: var(--color-danger);
      
      .logout-icon {
        opacity: 1;
        transform: translateX(0);
      }
    }
    
    .logout-icon {
      margin-left: auto;
      opacity: 0;
      transform: translateX(-5px);
      transition: all 0.2s ease;
    }
  }
  
  .version-info {
    font-size: 12px;
    color: var(--color-text-secondary);
    padding-left: 12px;
  }
}

.main-content {
  flex: 1;
  margin-left: 260px;
  display: flex;
  flex-direction: column;
  min-height: 100vh;
}

.page-header {
  height: 56px;
  padding: 0 32px;
  background: var(--color-bg-secondary);
  border-bottom: 1px solid var(--color-border);
  display: flex;
  align-items: center;
  justify-content: space-between;
  position: sticky;
  top: 0;
  z-index: 50;
  
  :deep(.el-breadcrumb__inner) {
    color: var(--color-text-secondary);
  }
  
  :deep(.el-breadcrumb__item:last-child .el-breadcrumb__inner) {
    color: var(--color-text-primary);
  }
}

.header-actions {
  display: flex;
  align-items: center;
  gap: 12px;
}

.status-badge {
  display: flex;
  align-items: center;
  gap: 10px;
  padding: 8px 14px;
  background: linear-gradient(135deg, rgba(46, 160, 67, 0.08) 0%, rgba(46, 160, 67, 0.04) 100%);
  border: 1px solid rgba(46, 160, 67, 0.2);
  border-radius: 20px;
  font-size: 13px;
  transition: all 0.3s ease;
  
  &:hover {
    background: linear-gradient(135deg, rgba(46, 160, 67, 0.12) 0%, rgba(46, 160, 67, 0.06) 100%);
    border-color: rgba(46, 160, 67, 0.3);
    transform: translateY(-1px);
  }
  
  .status-dot {
    width: 8px;
    height: 8px;
    border-radius: 50%;
    background: #2ea043;
    box-shadow: 0 0 8px rgba(46, 160, 67, 0.6);
    animation: pulse 2s ease-in-out infinite;
  }
  
  .status-text {
    color: var(--color-text-secondary);
    font-family: var(--font-mono);
    font-size: 12px;
  }
  
  .status-label {
    color: #2ea043;
    font-weight: 500;
  }
  
  &.disconnected {
    background: linear-gradient(135deg, rgba(248, 81, 73, 0.08) 0%, rgba(248, 81, 73, 0.04) 100%);
    border-color: rgba(248, 81, 73, 0.2);
    
    .status-dot {
      background: #f85149;
      box-shadow: 0 0 8px rgba(248, 81, 73, 0.6);
      animation: none;
    }
    
    .status-label {
      color: #f85149;
    }
  }
}

@keyframes pulse {
  0%, 100% {
    opacity: 1;
    transform: scale(1);
  }
  50% {
    opacity: 0.6;
    transform: scale(1.1);
  }
}

.page-content {
  flex: 1;
  padding: 32px;
  overflow-y: auto;
}

.fade-enter-active,
.fade-leave-active {
  transition: opacity 0.2s ease;
}

.fade-enter-from,
.fade-leave-to {
  opacity: 0;
}
</style>
