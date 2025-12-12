import axios from 'axios'

const api = axios.create({
  baseURL: '/api/v1',
  timeout: 10000,
  headers: {
    'Content-Type': 'application/json'
  }
})

// 响应拦截器
api.interceptors.response.use(
  response => {
    const { data } = response
    if (data.code !== 0) {
      return Promise.reject(new Error(data.message || '请求失败'))
    }
    return data
  },
  error => {
    if (error.code === 'ECONNABORTED') {
      return Promise.reject(new Error('请求超时，请检查后端服务是否运行'))
    }
    if (!error.response) {
      return Promise.reject(new Error('无法连接到后端服务，请确保后端已启动'))
    }
    return Promise.reject(error)
  }
)

// 用户登录
export const login = (username, password) => 
  api.post('/auth/login', { username, password })

// 系统设置
export const getSettings = () => api.get('/settings')
export const saveSettings = (settings) => api.post('/settings', settings)

// 系统状态
export const getSystemStatus = () => api.get('/status')

// 获取支持的字段列表
export const getSupportedFields = () => api.get('/fields')

// 获取默认配置
export const getDefaultConfig = () => api.get('/config/default')

// 全局配置
export const getGlobalConfig = () => api.get('/config/global')
export const saveGlobalConfig = (config, createdBy) => 
  api.post('/config/global', { config, created_by: createdBy })
export const getGlobalConfigHistory = (limit = 20) => 
  api.get('/config/global/history', { params: { limit } })

// 集群配置
export const listClusters = () => api.get('/clusters')
export const getClusterConfig = (cluster) => api.get(`/config/cluster/${cluster}`)
export const saveClusterConfig = (cluster, config, createdBy) =>
  api.post(`/config/cluster/${cluster}`, { config, created_by: createdBy })
export const getClusterConfigHistory = (cluster, limit = 20) =>
  api.get(`/config/cluster/${cluster}/history`, { params: { limit } })

// 节点配置
export const listNodes = (cluster) => api.get(`/clusters/${cluster}/nodes`)
export const getNodeConfig = (cluster, node) => api.get(`/config/cluster/${cluster}/node/${node}`)
export const saveNodeConfig = (cluster, node, config, createdBy) =>
  api.post(`/config/cluster/${cluster}/node/${node}`, { config, created_by: createdBy })
export const getNodeConfigHistory = (cluster, node, limit = 20) =>
  api.get(`/config/cluster/${cluster}/node/${node}/history`, { params: { limit } })

// 配置回滚
export const rollbackConfig = (scope, clusterName, nodeId, version, createdBy) =>
  api.post('/config/rollback', {
    scope,
    cluster_name: clusterName,
    node_id: nodeId,
    version,
    created_by: createdBy
  })

