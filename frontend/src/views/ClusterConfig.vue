<template>
  <div class="cluster-config-page fade-in">
    <div class="page-title">
      <h2>集群配置</h2>
      <p class="text-secondary">管理各个集群的 YAF 采集配置</p>
    </div>
    
    <div class="actions-bar">
      <el-input
        v-model="searchKeyword"
        placeholder="搜索集群..."
        prefix-icon="Search"
        style="width: 300px"
        clearable
      />
      <el-button type="primary" @click="showAddDialog = true">
        <el-icon><Plus /></el-icon>
        添加集群
      </el-button>
    </div>
    
    <div v-if="loading" class="loading-state">
      <el-skeleton :rows="5" animated />
    </div>
    
    <div v-else-if="filteredClusters.length === 0" class="empty-state">
      <el-empty description="暂无集群配置">
        <el-button type="primary" @click="showAddDialog = true">
          <el-icon><Plus /></el-icon>
          添加第一个集群
        </el-button>
      </el-empty>
    </div>
    
    <div v-else class="clusters-grid">
      <div 
        v-for="cluster in filteredClusters" 
        :key="cluster"
        class="cluster-card"
        @click="goToCluster(cluster)"
      >
        <div class="cluster-icon">
          <el-icon :size="24"><Grid /></el-icon>
        </div>
        <div class="cluster-info">
          <h3 class="mono">{{ cluster }}</h3>
          <p class="text-secondary">点击管理配置</p>
        </div>
        <el-icon class="arrow-icon"><ArrowRight /></el-icon>
      </div>
    </div>
    
    <!-- 添加集群对话框 -->
    <el-dialog
      v-model="showAddDialog"
      title="添加集群"
      width="480px"
      :close-on-click-modal="false"
    >
      <el-form :model="newCluster" label-width="100px">
        <el-form-item label="集群名称" required>
          <el-input 
            v-model="newCluster.name" 
            placeholder="例如: backbone, idc-east"
            class="mono-input"
          />
          <div class="form-tip">只允许字母、数字、下划线、中划线</div>
        </el-form-item>
      </el-form>
      
      <template #footer>
        <el-button @click="showAddDialog = false">取消</el-button>
        <el-button type="primary" @click="handleAddCluster" :loading="adding">
          创建集群
        </el-button>
      </template>
    </el-dialog>
  </div>
</template>

<script setup>
import { ref, computed, onMounted } from 'vue'
import { useRouter } from 'vue-router'
import { ElMessage } from 'element-plus'
import { listClusters, saveClusterConfig, getDefaultConfig } from '../api/config'

const router = useRouter()
const loading = ref(true)
const clusters = ref([])
const searchKeyword = ref('')
const showAddDialog = ref(false)
const adding = ref(false)
const newCluster = ref({ name: '' })

const filteredClusters = computed(() => {
  if (!searchKeyword.value) return clusters.value
  const keyword = searchKeyword.value.toLowerCase()
  return clusters.value.filter(c => c.toLowerCase().includes(keyword))
})

const loadClusters = async () => {
  loading.value = true
  try {
    const res = await listClusters()
    clusters.value = res.data || []
  } catch (error) {
    ElMessage.error('加载集群列表失败: ' + error.message)
  } finally {
    loading.value = false
  }
}

const goToCluster = (cluster) => {
  router.push(`/config/cluster/${cluster}`)
}

const handleAddCluster = async () => {
  const name = newCluster.value.name.trim()
  if (!name) {
    ElMessage.warning('请输入集群名称')
    return
  }
  
  // 验证名称格式
  if (!/^[a-zA-Z0-9_-]+$/.test(name)) {
    ElMessage.warning('集群名称只能包含字母、数字、下划线和中划线')
    return
  }
  
  if (clusters.value.includes(name)) {
    ElMessage.warning('集群已存在')
    return
  }
  
  adding.value = true
  try {
    // 使用默认配置创建集群
    const defaultRes = await getDefaultConfig()
    await saveClusterConfig(name, defaultRes.data, 'admin')
    
    ElMessage.success('集群创建成功')
    showAddDialog.value = false
    newCluster.value.name = ''
    await loadClusters()
    
    // 跳转到新集群
    router.push(`/config/cluster/${name}`)
  } catch (error) {
    ElMessage.error('创建集群失败: ' + error.message)
  } finally {
    adding.value = false
  }
}

onMounted(() => {
  loadClusters()
})
</script>

<style lang="scss" scoped>
.cluster-config-page {
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
}

.actions-bar {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 24px;
}

.loading-state,
.empty-state {
  padding: 60px 40px;
  background: var(--color-bg-secondary);
  border-radius: var(--radius-md);
  border: 1px solid var(--color-border);
}

.clusters-grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(320px, 1fr));
  gap: 16px;
}

.cluster-card {
  display: flex;
  align-items: center;
  gap: 16px;
  padding: 20px;
  background: var(--color-bg-secondary);
  border: 1px solid var(--color-border);
  border-radius: var(--radius-md);
  cursor: pointer;
  transition: all 0.2s ease;
  
  &:hover {
    border-color: var(--color-accent);
    background: var(--color-bg-tertiary);
    transform: translateY(-2px);
    box-shadow: var(--shadow-md);
  }
  
  .cluster-icon {
    width: 48px;
    height: 48px;
    background: linear-gradient(135deg, #238636 0%, #2ea043 100%);
    border-radius: var(--radius-md);
    display: flex;
    align-items: center;
    justify-content: center;
    color: white;
  }
  
  .cluster-info {
    flex: 1;
    
    h3 {
      font-size: 16px;
      font-weight: 600;
      margin-bottom: 4px;
      color: var(--color-text-primary);
    }
    
    p {
      font-size: 13px;
    }
  }
  
  .arrow-icon {
    color: var(--color-text-secondary);
    font-size: 18px;
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

