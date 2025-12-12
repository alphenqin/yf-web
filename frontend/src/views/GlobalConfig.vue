<template>
  <div class="global-config-page fade-in">
    <div class="page-title">
      <h2>全局配置</h2>
      <p class="text-secondary">全局配置将作为所有集群和节点的默认配置</p>
    </div>
    
    <div v-if="loading" class="loading-state">
      <el-skeleton :rows="10" animated />
    </div>
    
    <template v-else-if="configData">
      <div v-if="currentVersion > 0" class="current-config-info">
        <el-alert type="info" :closable="false" show-icon>
          <template #title>
            <span>当前版本: <strong class="mono">v{{ currentVersion }}</strong></span>
            <span class="divider">|</span>
            <span>更新时间: {{ formatTime(currentUpdatedAt) }}</span>
            <span class="divider">|</span>
            <span>操作人: {{ currentCreatedBy || '系统' }}</span>
          </template>
        </el-alert>
      </div>
      
      <ConfigForm 
        v-model="configData"
        :submitting="submitting"
        @submit="handleSubmit"
        @cancel="handleReset"
      />
    </template>
    
    <div v-else class="loading-state">
      <el-skeleton :rows="10" animated />
    </div>
  </div>
</template>

<script setup>
import { ref, onMounted } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import ConfigForm from '../components/ConfigForm.vue'
import { getGlobalConfig, saveGlobalConfig, getDefaultConfig } from '../api/config'

const loading = ref(true)
const submitting = ref(false)
const configData = ref(null)
const currentConfig = ref(null)
const currentVersion = ref(0)
const currentUpdatedAt = ref('')
const currentCreatedBy = ref('')

const loadConfig = async () => {
  loading.value = true
  try {
    const res = await getGlobalConfig()
    if (res.data) {
      configData.value = res.data.config
      currentConfig.value = res.data.config
      currentVersion.value = res.data.version
      currentUpdatedAt.value = res.data.created_at
      currentCreatedBy.value = res.data.created_by
    } else {
      // 没有配置，使用默认配置
      const defaultRes = await getDefaultConfig()
      configData.value = defaultRes.data
    }
  } catch (error) {
    ElMessage.error('加载配置失败: ' + error.message)
    // 使用默认配置
    const defaultRes = await getDefaultConfig()
    configData.value = defaultRes.data
  } finally {
    loading.value = false
  }
}

const handleSubmit = async (data) => {
  try {
    await ElMessageBox.confirm(
      '确定要保存全局配置吗？这将影响所有使用全局配置的集群和节点。',
      '确认保存',
      { confirmButtonText: '确定', cancelButtonText: '取消', type: 'warning' }
    )
  } catch {
    return
  }
  
  submitting.value = true
  try {
    const res = await saveGlobalConfig(data, 'admin')
    ElMessage.success(`配置保存成功，新版本: v${res.data.version}`)
    await loadConfig()
  } catch (error) {
    ElMessage.error('保存失败: ' + error.message)
  } finally {
    submitting.value = false
  }
}

const handleReset = () => {
  if (currentConfig.value) {
    configData.value = JSON.parse(JSON.stringify(currentConfig.value))
  }
}

const formatTime = (time) => {
  if (!time) return '-'
  return new Date(time).toLocaleString('zh-CN')
}

onMounted(() => {
  loadConfig()
})
</script>

<style lang="scss" scoped>
.global-config-page {
  max-width: 100%;
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

.current-config-info {
  margin-bottom: 24px;
  
  .el-alert {
    background: var(--color-bg-secondary);
    border: 1px solid var(--color-border);
  }
  
  .divider {
    margin: 0 12px;
    color: var(--color-border);
  }
}
</style>

