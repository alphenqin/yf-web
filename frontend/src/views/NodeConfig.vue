<template>
  <div class="node-config-page fade-in">
    <div class="page-title">
      <div class="title-with-back">
        <el-button text @click="$router.push(`/config/cluster/${clusterName}`)">
          <el-icon><ArrowLeft /></el-icon>
        </el-button>
        <div>
          <div class="breadcrumb-path">
            <span class="mono text-secondary">{{ clusterName }}</span>
            <el-icon class="text-secondary"><ArrowRight /></el-icon>
          </div>
          <h2>
            <span class="mono">{{ nodeId }}</span>
            <el-tag type="warning" size="small">节点</el-tag>
          </h2>
          <p class="text-secondary">节点配置将覆盖集群和全局配置</p>
        </div>
      </div>
    </div>
    
    <div v-if="loading" class="loading-state">
      <el-skeleton :rows="10" animated />
    </div>
    
    <template v-else>
      <div v-if="currentConfig" class="current-config-info">
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
        v-if="configData"
        v-model="configData"
        :submitting="submitting"
        @submit="handleSubmit"
        @cancel="handleReset"
      />
    </template>
  </div>
</template>

<script setup>
import { ref, computed, onMounted } from 'vue'
import { useRoute } from 'vue-router'
import { ElMessage, ElMessageBox } from 'element-plus'
import ConfigForm from '../components/ConfigForm.vue'
import { getNodeConfig, saveNodeConfig, getDefaultConfig } from '../api/config'

const route = useRoute()
const clusterName = computed(() => route.params.cluster)
const nodeId = computed(() => route.params.node)

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
    const res = await getNodeConfig(clusterName.value, nodeId.value)
    if (res.data) {
      configData.value = res.data.config
      currentConfig.value = res.data.config
      currentVersion.value = res.data.version
      currentUpdatedAt.value = res.data.created_at
      currentCreatedBy.value = res.data.created_by
    } else {
      const defaultRes = await getDefaultConfig()
      configData.value = defaultRes.data
    }
  } catch (error) {
    ElMessage.error('加载配置失败: ' + error.message)
    const defaultRes = await getDefaultConfig()
    configData.value = defaultRes.data
  } finally {
    loading.value = false
  }
}

const handleSubmit = async (data) => {
  try {
    await ElMessageBox.confirm(
      `确定要保存节点 "${nodeId.value}" 的配置吗？`,
      '确认保存',
      { confirmButtonText: '确定', cancelButtonText: '取消', type: 'warning' }
    )
  } catch {
    return
  }
  
  submitting.value = true
  try {
    const res = await saveNodeConfig(clusterName.value, nodeId.value, data, 'admin')
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
.node-config-page {
  max-width: 900px;
}

.page-title {
  margin-bottom: 24px;
}

.title-with-back {
  display: flex;
  align-items: flex-start;
  gap: 12px;
  
  .el-button {
    margin-top: 4px;
    font-size: 18px;
  }
  
  .breadcrumb-path {
    display: flex;
    align-items: center;
    gap: 6px;
    margin-bottom: 4px;
    font-size: 13px;
  }
  
  h2 {
    display: flex;
    align-items: center;
    gap: 12px;
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

