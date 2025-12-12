<template>
  <div class="config-history-page fade-in">
    <div class="page-title">
      <h2>配置历史</h2>
      <p class="text-secondary">查看所有配置变更记录，支持版本回滚</p>
    </div>
    
    <div class="filter-bar">
      <el-select v-model="scopeFilter" placeholder="选择作用范围" style="width: 150px">
        <el-option label="全部" value="" />
        <el-option label="全局配置" value="global" />
        <el-option label="集群配置" value="cluster" />
        <el-option label="节点配置" value="node" />
      </el-select>
      
      <el-select 
        v-if="scopeFilter === 'cluster' || scopeFilter === 'node'"
        v-model="clusterFilter" 
        placeholder="选择集群" 
        style="width: 180px"
        @change="handleClusterChange"
      >
        <el-option 
          v-for="cluster in clusters" 
          :key="cluster" 
          :label="cluster" 
          :value="cluster" 
        />
      </el-select>
      
      <el-select 
        v-if="scopeFilter === 'node' && clusterFilter"
        v-model="nodeFilter" 
        placeholder="选择节点" 
        style="width: 180px"
      >
        <el-option 
          v-for="node in nodes" 
          :key="node" 
          :label="node" 
          :value="node" 
        />
      </el-select>
      
      <el-button type="primary" @click="loadHistory">
        <el-icon><Search /></el-icon>
        查询
      </el-button>
    </div>
    
    <div v-if="loading" class="loading-state">
      <el-skeleton :rows="8" animated />
    </div>
    
    <div v-else-if="history.length === 0" class="empty-state">
      <el-empty description="暂无配置历史记录" />
    </div>
    
    <el-table v-else :data="history" class="history-table" style="width: 100%">
      <el-table-column prop="version" label="版本" width="80">
        <template #default="{ row }">
          <el-tag type="info" class="mono">v{{ row.version }}</el-tag>
        </template>
      </el-table-column>
      
      <el-table-column prop="scope" label="作用范围" width="100">
        <template #default="{ row }">
          <el-tag 
            :type="getScopeTagType(row.scope)"
            effect="plain"
          >
            {{ getScopeLabel(row.scope) }}
          </el-tag>
        </template>
      </el-table-column>
      
      <el-table-column prop="cluster_name" label="集群" min-width="150">
        <template #default="{ row }">
          <span class="mono">{{ row.cluster_name || '-' }}</span>
        </template>
      </el-table-column>
      
      <el-table-column prop="node_id" label="节点" min-width="150">
        <template #default="{ row }">
          <span class="mono">{{ row.node_id || '-' }}</span>
        </template>
      </el-table-column>
      
      <el-table-column prop="created_by" label="操作人" min-width="120">
        <template #default="{ row }">
          {{ row.created_by || '系统' }}
        </template>
      </el-table-column>
      
      <el-table-column prop="created_at" label="创建时间" min-width="180">
        <template #default="{ row }">
          {{ formatTime(row.created_at) }}
        </template>
      </el-table-column>
      
      <el-table-column label="操作" width="140" fixed="right">
        <template #default="{ row }">
          <el-button type="primary" text size="small" @click="viewConfig(row)">
            查看
          </el-button>
          <el-button type="warning" text size="small" @click="handleRollback(row)">
            回滚
          </el-button>
        </template>
      </el-table-column>
    </el-table>
    
    <!-- 查看配置对话框 -->
    <el-dialog
      v-model="showConfigDialog"
      title="配置详情"
      width="700px"
    >
      <pre class="config-json mono">{{ selectedConfigJson }}</pre>
    </el-dialog>
  </div>
</template>

<script setup>
import { ref, onMounted } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import { 
  getGlobalConfigHistory, getClusterConfigHistory, getNodeConfigHistory,
  listClusters, listNodes, rollbackConfig
} from '../api/config'

const loading = ref(false)
const scopeFilter = ref('')
const clusterFilter = ref('')
const nodeFilter = ref('')
const clusters = ref([])
const nodes = ref([])
const history = ref([])
const showConfigDialog = ref(false)
const selectedConfigJson = ref('')

const loadClusters = async () => {
  try {
    const res = await listClusters()
    clusters.value = res.data || []
  } catch (error) {
    console.error('Failed to load clusters:', error)
  }
}

const handleClusterChange = async () => {
  nodeFilter.value = ''
  if (!clusterFilter.value) {
    nodes.value = []
    return
  }
  try {
    const res = await listNodes(clusterFilter.value)
    nodes.value = res.data || []
  } catch (error) {
    console.error('Failed to load nodes:', error)
  }
}

const loadHistory = async () => {
  loading.value = true
  try {
    let res
    if (scopeFilter.value === 'global' || !scopeFilter.value) {
      res = await getGlobalConfigHistory(50)
      history.value = res.data || []
    } else if (scopeFilter.value === 'cluster' && clusterFilter.value) {
      res = await getClusterConfigHistory(clusterFilter.value, 50)
      history.value = res.data || []
    } else if (scopeFilter.value === 'node' && clusterFilter.value && nodeFilter.value) {
      res = await getNodeConfigHistory(clusterFilter.value, nodeFilter.value, 50)
      history.value = res.data || []
    } else {
      // 默认加载全局配置历史
      res = await getGlobalConfigHistory(50)
      history.value = res.data || []
    }
  } catch (error) {
    ElMessage.error('加载历史失败: ' + error.message)
  } finally {
    loading.value = false
  }
}

const viewConfig = (row) => {
  try {
    const config = JSON.parse(row.config_json)
    selectedConfigJson.value = JSON.stringify(config, null, 2)
    showConfigDialog.value = true
  } catch {
    selectedConfigJson.value = row.config_json
    showConfigDialog.value = true
  }
}

const handleRollback = async (row) => {
  try {
    await ElMessageBox.confirm(
      `确定要回滚到版本 v${row.version} 吗？这将创建一个新版本，内容与 v${row.version} 相同。`,
      '确认回滚',
      { confirmButtonText: '确定回滚', cancelButtonText: '取消', type: 'warning' }
    )
  } catch {
    return
  }
  
  try {
    const res = await rollbackConfig(
      row.scope,
      row.cluster_name || '',
      row.node_id || '',
      row.version,
      'admin'
    )
    ElMessage.success(`回滚成功，新版本: v${res.data.new_version}`)
    await loadHistory()
  } catch (error) {
    ElMessage.error('回滚失败: ' + error.message)
  }
}

const getScopeTagType = (scope) => {
  const types = { global: 'primary', cluster: 'success', node: 'warning' }
  return types[scope] || 'info'
}

const getScopeLabel = (scope) => {
  const labels = { global: '全局', cluster: '集群', node: '节点' }
  return labels[scope] || scope
}

const formatTime = (time) => {
  if (!time) return '-'
  return new Date(time).toLocaleString('zh-CN')
}

onMounted(() => {
  loadClusters()
  loadHistory()
})
</script>

<style lang="scss" scoped>
.config-history-page {
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

.filter-bar {
  display: flex;
  gap: 12px;
  margin-bottom: 24px;
  flex-wrap: wrap;
}

.loading-state,
.empty-state {
  padding: 60px 40px;
  background: var(--color-bg-secondary);
  border-radius: var(--radius-md);
  border: 1px solid var(--color-border);
}

.history-table {
  border-radius: var(--radius-md);
  overflow: hidden;
}

.config-json {
  background: var(--color-bg-tertiary);
  padding: 16px;
  border-radius: var(--radius-md);
  font-size: 13px;
  line-height: 1.6;
  max-height: 500px;
  overflow: auto;
  color: var(--color-text-primary);
  white-space: pre-wrap;
  word-break: break-all;
}
</style>

