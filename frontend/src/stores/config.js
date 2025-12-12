import { defineStore } from 'pinia'
import { ref } from 'vue'
import { getSupportedFields, getDefaultConfig } from '../api/config'

export const useConfigStore = defineStore('config', () => {
  const supportedFields = ref([])
  const defaultConfig = ref(null)
  const loading = ref(false)

  const fetchSupportedFields = async () => {
    if (supportedFields.value.length > 0) return
    try {
      const res = await getSupportedFields()
      supportedFields.value = res.data
    } catch (error) {
      console.error('Failed to fetch supported fields:', error)
    }
  }

  const fetchDefaultConfig = async () => {
    if (defaultConfig.value) return
    try {
      const res = await getDefaultConfig()
      defaultConfig.value = res.data
    } catch (error) {
      console.error('Failed to fetch default config:', error)
    }
  }

  const init = async () => {
    loading.value = true
    await Promise.all([fetchSupportedFields(), fetchDefaultConfig()])
    loading.value = false
  }

  return {
    supportedFields,
    defaultConfig,
    loading,
    init,
    fetchSupportedFields,
    fetchDefaultConfig
  }
})

