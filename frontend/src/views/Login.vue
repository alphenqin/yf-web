<template>
  <div class="login-container">
    <div class="login-bg">
      <div class="bg-gradient"></div>
      <div class="bg-grid"></div>
    </div>
    
    <div class="login-card">
      <div class="login-header">
        <div class="logo-icon">
          <el-icon :size="32"><Monitor /></el-icon>
        </div>
        <h1>YAF 配置中心</h1>
        <p class="subtitle">分布式配置管理系统</p>
      </div>
      
      <el-form 
        ref="formRef" 
        :model="form" 
        :rules="rules" 
        class="login-form"
        @submit.prevent="handleLogin"
      >
        <el-form-item prop="username">
          <el-input
            v-model="form.username"
            placeholder="用户名"
            size="large"
            :prefix-icon="User"
          />
        </el-form-item>
        
        <el-form-item prop="password">
          <el-input
            v-model="form.password"
            type="password"
            placeholder="密码"
            size="large"
            :prefix-icon="Lock"
            show-password
            @keyup.enter="handleLogin"
          />
        </el-form-item>
        
        <el-form-item>
          <el-checkbox v-model="form.remember" label="记住登录状态" />
        </el-form-item>
        
        <el-button 
          type="primary" 
          size="large" 
          :loading="loading"
          class="login-btn"
          @click="handleLogin"
        >
          {{ loading ? '登录中...' : '登 录' }}
        </el-button>
      </el-form>
      
      <div class="login-footer">
        <span class="hint">默认账号: admin / admin</span>
      </div>
    </div>
    
    <div class="login-decoration">
      <div class="floating-shape shape-1"></div>
      <div class="floating-shape shape-2"></div>
      <div class="floating-shape shape-3"></div>
    </div>
  </div>
</template>

<script setup>
import { ref, reactive } from 'vue'
import { useRouter } from 'vue-router'
import { ElMessage } from 'element-plus'
import { User, Lock, Monitor } from '@element-plus/icons-vue'
import { login } from '../api/config'

const router = useRouter()
const formRef = ref(null)
const loading = ref(false)

const form = reactive({
  username: '',
  password: '',
  remember: false
})

const rules = {
  username: [
    { required: true, message: '请输入用户名', trigger: 'blur' }
  ],
  password: [
    { required: true, message: '请输入密码', trigger: 'blur' }
  ]
}

const handleLogin = async () => {
  if (!formRef.value) return
  
  await formRef.value.validate(async (valid) => {
    if (!valid) return
    
    loading.value = true
    
    try {
      const res = await login(form.username, form.password)
      
      // 保存登录状态
      const storage = form.remember ? localStorage : sessionStorage
      storage.setItem('yaf_token', res.data.token)
      storage.setItem('yaf_user', res.data.username)
      
      ElMessage.success('登录成功')
      router.push('/')
    } catch (error) {
      ElMessage.error(error.message || '登录失败')
    } finally {
      loading.value = false
    }
  })
}
</script>

<style lang="scss" scoped>
.login-container {
  min-height: 100vh;
  display: flex;
  align-items: center;
  justify-content: center;
  position: relative;
  overflow: hidden;
}

.login-bg {
  position: fixed;
  inset: 0;
  z-index: 0;
  
  .bg-gradient {
    position: absolute;
    inset: 0;
    background: 
      radial-gradient(ellipse at 20% 20%, rgba(88, 166, 255, 0.15) 0%, transparent 50%),
      radial-gradient(ellipse at 80% 80%, rgba(31, 111, 235, 0.1) 0%, transparent 50%),
      radial-gradient(ellipse at 50% 50%, rgba(88, 166, 255, 0.05) 0%, transparent 70%);
  }
  
  .bg-grid {
    position: absolute;
    inset: 0;
    background-image: 
      linear-gradient(rgba(48, 54, 61, 0.3) 1px, transparent 1px),
      linear-gradient(90deg, rgba(48, 54, 61, 0.3) 1px, transparent 1px);
    background-size: 60px 60px;
    mask-image: radial-gradient(ellipse at center, black 20%, transparent 70%);
  }
}

.login-card {
  position: relative;
  z-index: 10;
  width: 100%;
  max-width: 420px;
  padding: 48px 40px;
  background: linear-gradient(145deg, rgba(22, 27, 34, 0.95) 0%, rgba(22, 27, 34, 0.85) 100%);
  border: 1px solid var(--color-border);
  border-radius: var(--radius-lg);
  box-shadow: 
    0 25px 50px -12px rgba(0, 0, 0, 0.5),
    0 0 0 1px rgba(88, 166, 255, 0.05),
    inset 0 1px 0 rgba(255, 255, 255, 0.05);
  backdrop-filter: blur(20px);
  animation: cardAppear 0.6s ease-out;
}

@keyframes cardAppear {
  from {
    opacity: 0;
    transform: translateY(20px) scale(0.98);
  }
  to {
    opacity: 1;
    transform: translateY(0) scale(1);
  }
}

.login-header {
  text-align: center;
  margin-bottom: 36px;
  
  .logo-icon {
    width: 64px;
    height: 64px;
    margin: 0 auto 20px;
    background: linear-gradient(135deg, #58a6ff 0%, #1f6feb 100%);
    border-radius: var(--radius-md);
    display: flex;
    align-items: center;
    justify-content: center;
    color: white;
    box-shadow: 
      0 8px 24px rgba(88, 166, 255, 0.35),
      0 0 0 1px rgba(88, 166, 255, 0.2);
    animation: logoFloat 3s ease-in-out infinite;
  }
  
  h1 {
    font-size: 26px;
    font-weight: 700;
    color: var(--color-text-primary);
    letter-spacing: -0.5px;
    margin-bottom: 8px;
  }
  
  .subtitle {
    font-size: 14px;
    color: var(--color-text-secondary);
  }
}

@keyframes logoFloat {
  0%, 100% {
    transform: translateY(0);
  }
  50% {
    transform: translateY(-4px);
  }
}

.login-form {
  .el-form-item {
    margin-bottom: 24px;
  }
  
  :deep(.el-input__wrapper) {
    padding: 4px 16px;
    background: var(--color-bg-tertiary);
    border: 1px solid var(--color-border);
    border-radius: var(--radius-md);
    transition: all 0.3s ease;
    
    &:hover {
      border-color: rgba(88, 166, 255, 0.5);
    }
    
    &.is-focus {
      border-color: var(--color-accent);
      box-shadow: 0 0 0 3px rgba(88, 166, 255, 0.15) !important;
    }
    
    .el-input__inner {
      height: 44px;
      font-size: 15px;
    }
  }
  
  :deep(.el-checkbox__label) {
    color: var(--color-text-secondary);
    font-size: 13px;
  }
}

.login-btn {
  width: 100%;
  height: 48px;
  font-size: 16px;
  font-weight: 600;
  border-radius: var(--radius-md);
  background: linear-gradient(135deg, #58a6ff 0%, #1f6feb 100%);
  border: none;
  transition: all 0.3s ease;
  
  &:hover {
    transform: translateY(-2px);
    box-shadow: 0 8px 20px rgba(88, 166, 255, 0.4);
  }
  
  &:active {
    transform: translateY(0);
  }
}

.login-footer {
  margin-top: 28px;
  text-align: center;
  
  .hint {
    font-size: 12px;
    color: var(--color-text-secondary);
    padding: 8px 16px;
    background: rgba(88, 166, 255, 0.08);
    border-radius: 20px;
    border: 1px solid rgba(88, 166, 255, 0.15);
  }
}

.login-decoration {
  position: fixed;
  inset: 0;
  pointer-events: none;
  z-index: 1;
}

.floating-shape {
  position: absolute;
  border-radius: 50%;
  opacity: 0.4;
  filter: blur(60px);
  animation: float 20s ease-in-out infinite;
  
  &.shape-1 {
    width: 400px;
    height: 400px;
    background: linear-gradient(135deg, rgba(88, 166, 255, 0.3) 0%, rgba(31, 111, 235, 0.2) 100%);
    top: -100px;
    right: -100px;
    animation-delay: 0s;
  }
  
  &.shape-2 {
    width: 300px;
    height: 300px;
    background: linear-gradient(135deg, rgba(31, 111, 235, 0.25) 0%, rgba(88, 166, 255, 0.15) 100%);
    bottom: -50px;
    left: -50px;
    animation-delay: -7s;
  }
  
  &.shape-3 {
    width: 200px;
    height: 200px;
    background: linear-gradient(135deg, rgba(46, 160, 67, 0.2) 0%, rgba(88, 166, 255, 0.1) 100%);
    top: 50%;
    right: 20%;
    animation-delay: -14s;
  }
}

@keyframes float {
  0%, 100% {
    transform: translate(0, 0) rotate(0deg);
  }
  25% {
    transform: translate(20px, -20px) rotate(5deg);
  }
  50% {
    transform: translate(-10px, 20px) rotate(-5deg);
  }
  75% {
    transform: translate(-20px, -10px) rotate(3deg);
  }
}
</style>

