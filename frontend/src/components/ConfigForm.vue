<template>
  <div class="config-form">
    <!-- 采集配置 -->
    <el-card class="config-section">
      <template #header>
        <div class="section-header">
          <el-icon><Timer /></el-icon>
          <span>采集配置</span>
        </div>
      </template>
      
      <el-form :model="formData" label-width="140px" label-position="left">
        <el-form-item label="网卡名称">
          <el-input 
            v-model="formData.capture.interface" 
            placeholder="eth0"
            style="width: 200px"
          />
          <span class="form-hint">抓包网卡，如 eth0、ens33</span>
        </el-form-item>
        
        <el-form-item label="IPFIX 端口">
          <el-input-number 
            v-model="formData.capture.ipfix_port" 
            :min="1" 
            :max="65535"
            controls-position="right"
          />
          <span class="form-hint">IPFIX 输出端口，super_mediator 监听此端口</span>
        </el-form-item>
        
        <el-form-item label="空闲超时 (秒)">
          <el-input-number 
            v-model="formData.capture.idle_timeout" 
            :min="10" 
            :max="3600" 
            :step="10"
            controls-position="right"
          />
          <span class="form-hint">流空闲多久后关闭，范围 10-3600 秒</span>
        </el-form-item>
        
        <el-form-item label="活跃超时 (秒)">
          <el-input-number 
            v-model="formData.capture.active_timeout" 
            :min="10" 
            :max="3600" 
            :step="10"
            controls-position="right"
          />
          <span class="form-hint">活跃流多久后强制切新流，范围 10-3600 秒</span>
        </el-form-item>
        
        <el-form-item label="统计间隔 (秒)">
          <el-input-number 
            v-model="formData.capture.stats_interval" 
            :min="60" 
            :max="3600" 
            :step="60"
            controls-position="right"
          />
          <span class="form-hint">每隔多少秒输出一次统计记录</span>
        </el-form-item>
        
        <el-form-item label="应用识别">
          <div class="switch-wrapper">
            <el-switch v-model="formData.capture.enable_applabel" />
            <span class="form-hint">启用 AppLabel 应用层协议识别</span>
          </div>
        </el-form-item>
        
        <el-form-item label="深度包检测">
          <div class="switch-wrapper">
            <el-switch v-model="formData.capture.enable_dpi" />
            <span class="form-hint">启用 DPI 深度包检测</span>
          </div>
        </el-form-item>
        
        <el-form-item label="最大载荷">
          <el-input-number 
            v-model="formData.capture.max_payload" 
            :min="0" 
            :max="65535" 
            :step="256"
            controls-position="right"
          />
          <span class="form-hint">每方向最大采集字节数</span>
        </el-form-item>
      </el-form>
    </el-card>
    
    <!-- 过滤配置 -->
    <el-card class="config-section">
      <template #header>
        <div class="section-header">
          <el-icon><Filter /></el-icon>
          <span>过滤配置</span>
        </div>
      </template>
      
      <el-form :model="formData" label-width="140px" label-position="left">
        <el-form-item label="IP 白名单">
          <el-select
            v-model="formData.filter.ip_whitelist"
            multiple
            filterable
            allow-create
            default-first-option
            placeholder="输入 CIDR 格式，如 10.0.0.0/8，按回车确认"
            style="width: 100%"
          />
          <span class="form-hint">只采集这些 IP 段的流量（CIDR 格式）</span>
        </el-form-item>
        
        <el-form-item label="IP 黑名单">
          <el-select
            v-model="formData.filter.ip_blacklist"
            multiple
            filterable
            allow-create
            default-first-option
            placeholder="输入 CIDR 格式，如 192.168.1.0/24，按回车确认"
            style="width: 100%"
          />
          <span class="form-hint">排除这些 IP 段的流量</span>
        </el-form-item>
        
        <el-form-item label="源端口">
          <el-select
            v-model="srcPortsModel"
            multiple
            filterable
            allow-create
            default-first-option
            placeholder="输入端口号，如 80, 443"
            style="width: 100%"
            :reserve-keyword="false"
          >
            <el-option
              v-for="port in commonPorts"
              :key="port.value"
              :label="`${port.value} (${port.label})`"
              :value="port.value"
            />
          </el-select>
          <span class="form-hint">只采集这些源端口的流量</span>
        </el-form-item>
        
        <el-form-item label="目的端口">
          <el-select
            v-model="dstPortsModel"
            multiple
            filterable
            allow-create
            default-first-option
            placeholder="输入端口号，如 80, 443"
            style="width: 100%"
            :reserve-keyword="false"
          >
            <el-option
              v-for="port in commonPorts"
              :key="port.value"
              :label="`${port.value} (${port.label})`"
              :value="port.value"
            />
          </el-select>
          <span class="form-hint">只采集这些目的端口的流量</span>
        </el-form-item>
        
        <el-form-item label="BPF 过滤器">
          <el-input 
            v-model="formData.filter.bpf_filter" 
            placeholder="例如: ip and not port 22"
            clearable
          />
          <span class="form-hint">Berkeley Packet Filter 表达式</span>
        </el-form-item>
      </el-form>
    </el-card>
    
    <!-- 输出配置 -->
    <el-card class="config-section">
      <template #header>
        <div class="section-header">
          <el-icon><Document /></el-icon>
          <span>输出字段</span>
        </div>
      </template>
      
      <div class="fields-grid">
        <el-checkbox-group v-model="formData.output.fields">
          <el-checkbox 
            v-for="field in supportedFields" 
            :key="field.name" 
            :label="field.name"
            :value="field.name"
            class="field-checkbox"
          >
            <span class="field-label">{{ field.label }}</span>
            <span class="field-name">{{ field.name }}</span>
          </el-checkbox>
        </el-checkbox-group>
      </div>
      
      <div class="fields-actions">
        <el-button size="small" @click="selectAllFields">全选</el-button>
        <el-button size="small" @click="selectCommonFields">常用字段</el-button>
        <el-button size="small" @click="clearFields">清空</el-button>
      </div>
    </el-card>
    
    <!-- 操作按钮 -->
    <div class="form-actions">
      <el-button @click="$emit('cancel')">取消</el-button>
      <el-button type="primary" @click="handleSubmit" :loading="submitting">
        <el-icon><Check /></el-icon>
        保存配置
      </el-button>
    </div>
  </div>
</template>

<script setup>
import { reactive, computed, watch, toRaw } from 'vue'
import { useConfigStore } from '../stores/config'

const props = defineProps({
  modelValue: {
    type: Object,
    required: true
  },
  submitting: {
    type: Boolean,
    default: false
  }
})

const emit = defineEmits(['update:modelValue', 'submit', 'cancel'])

const configStore = useConfigStore()

// 确保有默认值
const getDefaultFormData = () => ({
  capture: {
    interface: 'eth0',
    ipfix_port: 18000,
    idle_timeout: 60,
    active_timeout: 60,
    stats_interval: 300,
    enable_applabel: true,
    enable_dpi: false,
    max_payload: 1024
  },
  filter: {
    ip_whitelist: [],
    ip_blacklist: [],
    src_ports: [],
    dst_ports: [],
    bpf_filter: 'ip and not port 22'
  },
  output: {
    fields: []
  }
})

// 深拷贝初始值
const cloneData = (data) => JSON.parse(JSON.stringify(data))

// 使用 reactive 确保深层响应性
const formData = reactive(props.modelValue ? cloneData(props.modelValue) : getDefaultFormData())

// 仅监听外部 props 变化（避免循环）
watch(() => props.modelValue, (newVal, oldVal) => {
  // 只有当外部传入的值真正变化时才更新
  if (newVal && JSON.stringify(newVal) !== JSON.stringify(toRaw(formData))) {
    Object.assign(formData.capture, cloneData(newVal.capture || {}))
    Object.assign(formData.filter, cloneData(newVal.filter || {}))
    Object.assign(formData.output, cloneData(newVal.output || {}))
  }
}, { deep: true })

const supportedFields = computed(() => configStore.supportedFields)

// 端口模型转换（确保是数字数组）
const srcPortsModel = computed({
  get: () => formData.filter.src_ports || [],
  set: (val) => {
    formData.filter.src_ports = val.map(v => typeof v === 'string' ? parseInt(v, 10) : v).filter(v => !isNaN(v))
  }
})

const dstPortsModel = computed({
  get: () => formData.filter.dst_ports || [],
  set: (val) => {
    formData.filter.dst_ports = val.map(v => typeof v === 'string' ? parseInt(v, 10) : v).filter(v => !isNaN(v))
  }
})

const commonPorts = [
  { value: 80, label: 'HTTP' },
  { value: 443, label: 'HTTPS' },
  { value: 22, label: 'SSH' },
  { value: 21, label: 'FTP' },
  { value: 53, label: 'DNS' },
  { value: 25, label: 'SMTP' },
  { value: 110, label: 'POP3' },
  { value: 143, label: 'IMAP' },
  { value: 3306, label: 'MySQL' },
  { value: 5432, label: 'PostgreSQL' },
  { value: 6379, label: 'Redis' },
  { value: 27017, label: 'MongoDB' }
]

const selectAllFields = () => {
  formData.output.fields = supportedFields.value.map(f => f.name)
}

const selectCommonFields = () => {
  formData.output.fields = [
    'flowStartMilliseconds',
    'flowEndMilliseconds',
    'sourceIPv4Address',
    'destinationIPv4Address',
    'sourceTransportPort',
    'destinationTransportPort',
    'protocolIdentifier',
    'silkAppLabel'
  ]
}

const clearFields = () => {
  formData.output.fields = []
}

const handleSubmit = () => {
  const data = cloneData(toRaw(formData))
  emit('update:modelValue', data)
  emit('submit', data)
}
</script>

<style lang="scss" scoped>
.config-form {
  display: flex;
  flex-direction: column;
  gap: 24px;
}

.config-section {
  :deep(.el-card__header) {
    padding: 16px 20px;
  }
}

.section-header {
  display: flex;
  align-items: center;
  gap: 8px;
  font-weight: 600;
  font-size: 15px;
  color: var(--color-text-primary);
  
  .el-icon {
    color: var(--color-accent);
    font-size: 18px;
  }
}

.form-hint {
  margin-left: 12px;
  font-size: 12px;
  color: var(--color-text-secondary);
  pointer-events: none; // 确保不会拦截点击事件
}

.switch-wrapper {
  display: flex;
  align-items: center;
  gap: 0;
  
  .el-switch {
    flex-shrink: 0;
  }
}

.mono-input {
  :deep(.el-input__inner) {
    font-family: var(--font-mono);
  }
}

.fields-grid {
  :deep(.el-checkbox-group) {
    display: grid;
    grid-template-columns: repeat(auto-fill, minmax(280px, 1fr));
    gap: 12px;
  }
}

.field-checkbox {
  margin-right: 0 !important;
  padding: 12px 14px;
  background: var(--color-bg-tertiary);
  border-radius: var(--radius-sm);
  border: 1px solid transparent;
  transition: all 0.2s ease;
  
  &:hover {
    border-color: var(--color-border);
  }
  
  &.is-checked {
    background: rgba(88, 166, 255, 0.1);
    border-color: var(--color-accent);
  }
  
  :deep(.el-checkbox__label) {
    display: flex;
    flex-direction: column;
    gap: 2px;
  }
  
  .field-label {
    font-size: 13px;
    color: var(--color-text-primary);
    font-weight: 500;
  }
  
  .field-name {
    font-size: 11px;
    color: var(--color-text-secondary);
    font-family: var(--font-mono);
  }
}

.fields-actions {
  margin-top: 16px;
  padding-top: 16px;
  border-top: 1px solid var(--color-border);
  display: flex;
  gap: 8px;
}

.form-actions {
  display: flex;
  justify-content: flex-end;
  gap: 12px;
  padding-top: 24px;
  border-top: 1px solid var(--color-border);
}
</style>

