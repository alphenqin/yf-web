<template>
  <div class="settings-page fade-in">
    <div class="page-title">
      <h2>系统设置</h2>
      <p class="text-secondary">配置系统连接参数</p>
    </div>
    
    <div v-if="loading" class="loading-state">
      <el-skeleton :rows="5" animated />
    </div>
    
    <template v-else>
      <el-card class="settings-card">
        <template #header>
          <div class="card-header">
            <el-icon><Connection /></el-icon>
            <span>ZooKeeper 配置</span>
          </div>
        </template>
        
        <el-form 
          ref="formRef" 
          :model="form" 
          :rules="rules"
          label-width="140px"
          label-position="left"
        >
          <el-form-item label="服务器地址" prop="zookeeper_servers">
            <el-input
              v-model="form.zookeeper_servers"
              placeholder="例如: localhost:2181 或 zk1:2181,zk2:2181,zk3:2181"
            />
            <div class="form-tip">
              多个地址用逗号分隔，格式: host:port
            </div>
          </el-form-item>
          
          <el-form-item>
            <el-button 
              type="primary" 
              :loading="submitting"
              @click="handleSubmit"
            >
              保存并重连
            </el-button>
            <el-button @click="handleReset">重置</el-button>
          </el-form-item>
        </el-form>
      </el-card>
      
      <el-card class="settings-card">
        <template #header>
          <div class="card-header">
            <el-icon><InfoFilled /></el-icon>
            <span>连接状态</span>
          </div>
        </template>
        
        <div class="status-info">
          <div class="status-row">
            <span class="label">ZooKeeper 状态:</span>
            <el-tag :type="zkConnected ? 'success' : 'danger'">
              {{ zkConnected ? '已连接' : '未连接' }}
            </el-tag>
          </div>
          <div class="status-row">
            <span class="label">当前服务器:</span>
            <span class="mono">{{ form.zookeeper_servers || '-' }}</span>
          </div>
        </div>
      </el-card>
    </template>
  </div>
</template>

<script setup>
import { ref, reactive, onMounted } from 'vue'
import { ElMessage } from 'element-plus'
import { getSettings, saveSettings, getSystemStatus } from '../api/config'

const loading = ref(true)
const submitting = ref(false)
const formRef = ref(null)
const zkConnected = ref(false)

const form = reactive({
  zookeeper_servers: ''
})

const originalForm = ref({})

const rules = {
  zookeeper_servers: [
    { required: true, message: '请输入 ZooKeeper 服务器地址', trigger: 'blur' },
    { 
      pattern: /^[\w.-]+:\d+(,[\w.-]+:\d+)*$/,
      message: '格式错误，请使用 host:port 格式，多个地址用逗号分隔',
      trigger: 'blur'
    }
  ]
}

const loadSettings = async () => {
  loading.value = true
  try {
    const [settingsRes, statusRes] = await Promise.all([
      getSettings(),
      getSystemStatus()
    ])
    
    form.zookeeper_servers = settingsRes.data.zookeeper_servers || ''
    originalForm.value = { ...form }
    zkConnected.value = statusRes.data.zookeeper?.connected || false
  } catch (error) {
    ElMessage.error('加载设置失败: ' + error.message)
  } finally {
    loading.value = false
  }
}

const handleSubmit = async () => {
  if (!formRef.value) return
  
  await formRef.value.validate(async (valid) => {
    if (!valid) return
    
    submitting.value = true
    try {
      const res = await saveSettings({
        zookeeper_servers: form.zookeeper_servers.trim()
      })
      
      if (res.data?.connected) {
        ElMessage.success(res.message)
        zkConnected.value = true
      } else {
        ElMessage.warning(res.message)
        zkConnected.value = false
      }
      
      originalForm.value = { ...form }
    } catch (error) {
      ElMessage.error('保存失败: ' + error.message)
    } finally {
      submitting.value = false
    }
  })
}

const handleReset = () => {
  form.zookeeper_servers = originalForm.value.zookeeper_servers
}

onMounted(() => {
  loadSettings()
})
</script>

<style lang="scss" scoped>
.settings-page {
  max-width: 800px;
}

.page-title {
  margin-bottom: 24px;
  
  h2 {
    font-size: 24px;
    font-weight: 600;
    margin-bottom: 8px;
    color: var(--color-text-primary);
  }
  
  p {
    font-size: 14px;
  }
}

.loading-state {
  padding: 40px;
  background: var(--color-bg-secondary);
  border-radius: var(--radius-md);
}

.settings-card {
  margin-bottom: 24px;
  
  .card-header {
    display: flex;
    align-items: center;
    gap: 8px;
    font-weight: 600;
    color: var(--color-text-primary);
    
    .el-icon {
      color: var(--color-accent);
    }
  }
  
  .form-tip {
    font-size: 12px;
    color: var(--color-text-secondary);
    margin-top: 4px;
  }
}

.status-info {
  .status-row {
    display: flex;
    align-items: center;
    gap: 12px;
    padding: 12px 0;
    border-bottom: 1px solid var(--color-border);
    
    &:last-child {
      border-bottom: none;
    }
    
    .label {
      color: var(--color-text-secondary);
      min-width: 120px;
    }
  }
}
</style>

