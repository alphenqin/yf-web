<template>
  <div class="cluster-detail-page fade-in">
    <div class="page-title">
      <div class="title-with-back">
        <el-button text @click="$router.push('/config/cluster')">
          <el-icon><ArrowLeft /></el-icon>
        </el-button>
        <div>
          <h2>
            <span class="mono">{{ clusterName }}</span>
            <el-tag type="success" size="small">集群</el-tag>
          </h2>
          <p class="text-secondary">管理集群配置和节点</p>
        </div>
      </div>
    </div>
    
    <el-tabs v-model="activeTab" class="config-tabs">
      <el-tab-pane label="集群配置" name="config">
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
      </el-tab-pane>
      
      <el-tab-pane label="节点管理" name="nodes">
        <div class="nodes-section">
          <div class="nodes-header">
            <el-input
              v-model="nodeSearchKeyword"
              placeholder="搜索节点..."
              prefix-icon="Search"
              style="width: 260px"
              clearable
            />
            <el-button type="primary" @click="showAddNodeDialog = true">
              <el-icon><Plus /></el-icon>
              添加节点
            </el-button>
          </div>
          
          <div v-if="nodesLoading" class="loading-state">
            <el-skeleton :rows="3" animated />
          </div>
          
          <div v-else-if="filteredNodes.length === 0" class="empty-state">
            <el-empty description="暂无节点配置">
              <el-button type="primary" @click="showAddNodeDialog = true">
                <el-icon><Plus /></el-icon>
                添加节点
              </el-button>
            </el-empty>
          </div>
          
          <div v-else class="nodes-grid">
            <div 
              v-for="node in filteredNodes" 
              :key="node"
              class="node-card"
              @click="goToNode(node)"
            >
              <div class="node-icon">
                <el-icon :size="20"><Monitor /></el-icon>
              </div>
              <div class="node-info">
                <h4 class="mono">{{ node }}</h4>
                <p class="text-secondary">点击管理配置</p>
              </div>
              <el-icon class="arrow-icon"><ArrowRight /></el-icon>
            </div>
          </div>
        </div>
      </el-tab-pane>
    </el-tabs>
    
    <!-- 添加节点对话框 -->
    <el-dialog
      v-model="showAddNodeDialog"
      title="添加节点"
      width="480px"
      :close-on-click-modal="false"
    >
      <el-form :model="newNode" label-width="100px">
        <el-form-item label="节点 ID" required>
          <el-input 
            v-model="newNode.id" 
            placeholder="例如: dev3-eth0, prod-server-01"
            class="mono-input"
          />
          <div class="form-tip">建议使用 hostname + 网卡名 的格式</div>
        </el-form-item>
      </el-form>
      
      <template #footer>
        <el-button @click="showAddNodeDialog = false">取消</el-button>
        <el-button type="primary" @click="handleAddNode" :loading="addingNode">
          创建节点
        </el-button>
      </template>
    </el-dialog>
  </div>
</template>

<script setup>
import { ref, computed, onMounted } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { ElMessage, ElMessageBox } from 'element-plus'
import ConfigForm from '../components/ConfigForm.vue'
import { 
  getClusterConfig, saveClusterConfig, getDefaultConfig,
  listNodes, saveNodeConfig 
} from '../api/config'

const route = useRoute()
const router = useRouter()
const clusterName = computed(() => route.params.cluster)

const activeTab = ref('config')
const loading = ref(true)
const submitting = ref(false)
const configData = ref(null)
const currentConfig = ref(null)
const currentVersion = ref(0)
const currentUpdatedAt = ref('')
const currentCreatedBy = ref('')

// 节点相关
const nodesLoading = ref(false)
const nodes = ref([])
const nodeSearchKeyword = ref('')
const showAddNodeDialog = ref(false)
const addingNode = ref(false)
const newNode = ref({ id: '' })

const filteredNodes = computed(() => {
  if (!nodeSearchKeyword.value) return nodes.value
  const keyword = nodeSearchKeyword.value.toLowerCase()
  return nodes.value.filter(n => n.toLowerCase().includes(keyword))
})

const loadConfig = async () => {
  loading.value = true
  try {
    const res = await getClusterConfig(clusterName.value)
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

const loadNodes = async () => {
  nodesLoading.value = true
  try {
    const res = await listNodes(clusterName.value)
    nodes.value = res.data || []
  } catch (error) {
    console.error('Failed to load nodes:', error)
  } finally {
    nodesLoading.value = false
  }
}

const handleSubmit = async (data) => {
  try {
    await ElMessageBox.confirm(
      `确定要保存集群 "${clusterName.value}" 的配置吗？`,
      '确认保存',
      { confirmButtonText: '确定', cancelButtonText: '取消', type: 'warning' }
    )
  } catch {
    return
  }
  
  submitting.value = true
  try {
    const res = await saveClusterConfig(clusterName.value, data, 'admin')
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

const goToNode = (node) => {
  router.push(`/config/cluster/${clusterName.value}/node/${node}`)
}

const handleAddNode = async () => {
  const nodeId = newNode.value.id.trim()
  if (!nodeId) {
    ElMessage.warning('请输入节点 ID')
    return
  }
  
  if (!/^[a-zA-Z0-9_.-]+$/.test(nodeId)) {
    ElMessage.warning('节点 ID 只能包含字母、数字、下划线、中划线和点')
    return
  }
  
  if (nodes.value.includes(nodeId)) {
    ElMessage.warning('节点已存在')
    return
  }
  
  addingNode.value = true
  try {
    const defaultRes = await getDefaultConfig()
    await saveNodeConfig(clusterName.value, nodeId, defaultRes.data, 'admin')
    
    ElMessage.success('节点创建成功')
    showAddNodeDialog.value = false
    newNode.value.id = ''
    await loadNodes()
    
    router.push(`/config/cluster/${clusterName.value}/node/${nodeId}`)
  } catch (error) {
    ElMessage.error('创建节点失败: ' + error.message)
  } finally {
    addingNode.value = false
  }
}

const formatTime = (time) => {
  if (!time) return '-'
  return new Date(time).toLocaleString('zh-CN')
}

onMounted(() => {
  loadConfig()
  loadNodes()
})
</script>

<style lang="scss" scoped>
.cluster-detail-page {
  max-width: 100%;
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

.config-tabs {
  :deep(.el-tabs__header) {
    margin-bottom: 24px;
  }
  
  :deep(.el-tabs__item) {
    font-size: 15px;
    
    &.is-active {
      color: var(--color-accent);
    }
  }
}

.loading-state,
.empty-state {
  padding: 40px;
  background: var(--color-bg-secondary);
  border-radius: var(--radius-md);
  border: 1px solid var(--color-border);
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

.nodes-section {
  .nodes-header {
    display: flex;
    justify-content: space-between;
    align-items: center;
    margin-bottom: 20px;
  }
}

.nodes-grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(280px, 1fr));
  gap: 12px;
}

.node-card {
  display: flex;
  align-items: center;
  gap: 12px;
  padding: 16px;
  background: var(--color-bg-secondary);
  border: 1px solid var(--color-border);
  border-radius: var(--radius-md);
  cursor: pointer;
  transition: all 0.2s ease;
  
  &:hover {
    border-color: var(--color-accent);
    background: var(--color-bg-tertiary);
  }
  
  .node-icon {
    width: 40px;
    height: 40px;
    background: linear-gradient(135deg, #a371f7 0%, #8957e5 100%);
    border-radius: var(--radius-sm);
    display: flex;
    align-items: center;
    justify-content: center;
    color: white;
  }
  
  .node-info {
    flex: 1;
    
    h4 {
      font-size: 14px;
      font-weight: 600;
      margin-bottom: 2px;
      color: var(--color-text-primary);
    }
    
    p {
      font-size: 12px;
    }
  }
  
  .arrow-icon {
    color: var(--color-text-secondary);
  }
}

.form-tip {
  margin-top: 8px;
  font-size: 12px;
  color: var(--color-text-secondary);
}

.mono-input {
  :deep(.el-input__inner) {
    font-family: var(--font-mono);
  }
}
</style>

