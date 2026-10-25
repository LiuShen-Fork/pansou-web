<script setup lang="ts">
import { ref, computed, onMounted, onUnmounted } from 'vue'
import * as weiboApi from '@/api/weibo'
import type { WeiboStatus, WeiboSearchResult } from '@/types/weibo'
import Button from '@/components/ui/Button.vue'
import Card from '@/components/ui/Card.vue'
import Input from '@/components/ui/Input.vue'

const emit = defineEmits<{
  (e: 'back-to-center'): void
}>()

interface SavedUser {
  uid: string
  hash: string
  last_login: string
}

const savedUsers = ref<SavedUser[]>([])
const currentView = ref<'list' | 'add' | 'manage'>('list')
const selectedUser = ref<SavedUser | null>(null)

const loadSavedUsers = () => {
  const stored = localStorage.getItem('weibo_users')
  if (stored) {
    try {
      savedUsers.value = JSON.parse(stored)
    } catch (e) {
      savedUsers.value = []
    }
  }
  
  if (savedUsers.value.length === 0) {
    currentView.value = 'add'
  }
}

const saveSavedUsers = () => {
  localStorage.setItem('weibo_users', JSON.stringify(savedUsers.value))
  window.dispatchEvent(new StorageEvent('storage', { key: 'weibo_users' }))
}

const addUserToList = (uid: string, hash: string) => {
  const user: SavedUser = {
    uid: uid,
    hash: hash,
    last_login: new Date().toISOString()
  }
  
  const index = savedUsers.value.findIndex(u => u.hash === hash)
  if (index >= 0) {
    savedUsers.value[index] = user
  } else {
    savedUsers.value.push(user)
  }
  
  saveSavedUsers()
}

const removeUser = (hash: string) => {
  savedUsers.value = savedUsers.value.filter(u => u.hash !== hash)
  saveSavedUsers()
}

const uid = ref('')
const generatingHash = ref(false)

const handleAddUID = async () => {
  if (!uid.value.trim()) {
    showAlertMessage('请输入微博UID', 'error')
    return
  }
  
  generatingHash.value = true
  
  try {
    const hash = await weiboApi.getHashByUID(uid.value.trim())
    
    selectedUser.value = {
      uid: uid.value.trim(),
      hash: hash,
      last_login: new Date().toISOString()
    }
    
    addUserToList(
      uid.value.trim(),
      hash
    )
    
    currentHash.value = hash
    currentView.value = 'manage'
    
    await loadStatus(true)
  } catch (error) {
    console.error('获取hash失败:', error)
    showAlertMessage('获取hash失败', 'error')
  } finally {
    generatingHash.value = false
  }
}

const handleSelectUser = (user: SavedUser) => {
  selectedUser.value = user
  currentHash.value = user.hash
  currentView.value = 'manage'
  loadStatus(true)
}

const handleBackToList = () => {
  stopAllPolling()
  selectedUser.value = null
  currentHash.value = ''
  currentView.value = savedUsers.value.length > 0 ? 'list' : 'add'
  uid.value = ''
  isEditingUserIds.value = false
}

const handleShowAddForm = () => {
  currentView.value = 'add'
  uid.value = ''
}

const currentHash = ref<string>('')

const status = ref<WeiboStatus>({
  hash: '',
  logged_in: false,
  status: 'pending',
  uid: '',
  login_time: '',
  expire_time: '',
  expires_in_days: 0,
  user_ids: [],
  user_id_count: 0
})

const qrcodeImage = ref<string>('')
const userIdsText = ref<string>('')
const searchKeyword = ref<string>('')
const searchResults = ref<WeiboSearchResult[]>([])
const searching = ref(false)

let loginCheckInterval: number | null = null

const showAlert = ref(false)
const alertMessage = ref('')
const alertType = ref<'success' | 'error'>('success')

const uidFirstChar = computed(() => {
  return status.value.uid?.[0] || selectedUser.value?.uid?.[0] || 'W'
})

const isLoggedIn = computed(() => {
  return status.value.logged_in && status.value.status === 'active'
})

onMounted(() => {
  loadSavedUsers()
})

onUnmounted(() => {
  stopAllPolling()
})

const isEditingUserIds = ref(false)

const loadStatus = async (forceUpdateUserIds = false) => {
  if (!currentHash.value) return
  
  try {
    const response = await weiboApi.getStatus(currentHash.value)
    
    if (response.success && response.data) {
      status.value = response.data
      
      if (response.data.logged_in && response.data.uid && selectedUser.value) {
        selectedUser.value.uid = response.data.uid
        selectedUser.value.last_login = new Date().toISOString()
        addUserToList(
          response.data.uid,
          selectedUser.value.hash
        )
      }
      
      if (!response.data.logged_in && response.data.qrcode_base64) {
        qrcodeImage.value = response.data.qrcode_base64
      }
      
      if (forceUpdateUserIds || !isEditingUserIds.value) {
        userIdsText.value = response.data.user_ids.join('\n')
      }
      
      if (!response.data.logged_in && !loginCheckInterval) {
        startLoginPolling()
      } else if (response.data.logged_in && loginCheckInterval) {
        stopLoginPolling()
      }
    }
  } catch (error) {
    console.error('获取状态失败:', error)
  }
}

const stopAllPolling = () => {
  stopLoginPolling()
}

const startLoginPolling = () => {
  if (loginCheckInterval) return
  loginCheckInterval = window.setInterval(checkLoginStatus, 2000)
}

const stopLoginPolling = () => {
  if (loginCheckInterval) {
    clearInterval(loginCheckInterval)
    loginCheckInterval = null
  }
}

const checkLoginStatus = async () => {
  if (!currentHash.value) return
  
  try {
    const response = await weiboApi.checkLogin(currentHash.value)
    
    if (response.success && response.data.login_status === 'success') {
      stopLoginPolling()
      showAlertMessage('登录成功！', 'success')
      await loadStatus(true)
    }
  } catch (error) {
    console.error('检查登录失败:', error)
  }
}

const handleRefreshQRCode = async () => {
  if (!currentHash.value) return
  
  try {
    const response = await weiboApi.refreshQRCode(currentHash.value)
    
    if (response.success) {
      showAlertMessage('二维码已刷新', 'success')
      await loadStatus(true)
      startLoginPolling()
    }
  } catch (error) {
    console.error('刷新二维码失败:', error)
    showAlertMessage('刷新二维码失败', 'error')
  }
}

const handleLogout = async () => {
  if (!currentHash.value) return
  if (!confirm('确定要退出登录吗？')) return
  
  try {
    const response = await weiboApi.logout(currentHash.value)
    
    if (response.success) {
      showAlertMessage('已退出登录', 'success')
      await loadStatus(true)
    }
  } catch (error) {
    console.error('退出登录失败:', error)
    showAlertMessage('退出登录失败', 'error')
  }
}

const handleDeleteAccount = () => {
  if (!selectedUser.value) return
  
  if (confirm(`确定要删除微博账号 ${selectedUser.value.uid} 吗？\n\n这将删除本地保存的配置信息。`)) {
    removeUser(selectedUser.value.hash)
    handleBackToList()
    showAlertMessage('账号已删除', 'success')
  }
}

const handleSaveUserIds = async () => {
  if (!currentHash.value) return
  
  const userIds = userIdsText.value
    .split('\n')
    .map(line => line.trim())
    .filter(line => line.length > 0)
  
  if (userIds.length === 0) {
    showAlertMessage('请至少添加一个微博用户ID', 'error')
    return
  }
  
  try {
    isEditingUserIds.value = false
    const response = await weiboApi.setUserIds(currentHash.value, userIds)
    
    if (response.success) {
      const count = response.data.user_ids?.length || response.data.user_id_count || 0
      const invalidCount = response.data.invalid_user_ids?.length || 0
      
      let msg = `✓已保存 ${count} 个用户ID`
      if (invalidCount > 0) {
        msg += `，${invalidCount} 个无效`
      }
      
      showAlertMessage(msg, 'success')
      await loadStatus(true)
    }
  } catch (error) {
    console.error('保存用户ID失败:', error)
    showAlertMessage('保存用户ID失败', 'error')
  }
}

const handleTestSearch = async () => {
  if (!currentHash.value) return
  
  if (!searchKeyword.value.trim()) {
    showAlertMessage('请输入搜索关键词', 'error')
    return
  }
  
  searching.value = true
  searchResults.value = []
  
  try {
    const response = await weiboApi.testSearch(currentHash.value, searchKeyword.value.trim(), 20)
    
    if (response.success) {
      searchResults.value = response.data.results || []
      
      if (searchResults.value.length === 0) {
        showAlertMessage('未找到结果', 'error')
      }
    }
  } catch (error: any) {
    console.error('搜索失败:', error)
    if (error.response?.data?.message) {
      showAlertMessage(error.response.data.message, 'error')
    } else {
      showAlertMessage('搜索失败', 'error')
    }
  } finally {
    searching.value = false
  }
}

const showAlertMessage = (message: string, type: 'success' | 'error' = 'success') => {
  alertMessage.value = message
  alertType.value = type
  showAlert.value = true
  
  setTimeout(() => {
    showAlert.value = false
  }, 3000)
}

const formatDateTime = (dateStr: string) => {
  if (!dateStr) return '-'
  try {
    const date = new Date(dateStr)
    return date.toLocaleString('zh-CN', {
      year: 'numeric',
      month: '2-digit',
      day: '2-digit',
      hour: '2-digit',
      minute: '2-digit'
    })
  } catch {
    return dateStr
  }
}

const copyHashToClipboard = async () => {
  if (!currentHash.value) return
  
  let success = false
  
  if (navigator.clipboard && navigator.clipboard.writeText) {
    try {
      await navigator.clipboard.writeText(currentHash.value)
      success = true
    } catch (error) {
      console.warn('Clipboard API 失败，尝试降级方案:', error)
    }
  }
  
  if (!success) {
    try {
      const textarea = document.createElement('textarea')
      textarea.value = currentHash.value
      textarea.style.position = 'fixed'
      textarea.style.opacity = '0'
      textarea.style.left = '-9999px'
      document.body.appendChild(textarea)
      
      textarea.select()
      textarea.setSelectionRange(0, currentHash.value.length)
      
      success = document.execCommand('copy')
      document.body.removeChild(textarea)
    } catch (error) {
      console.error('复制失败:', error)
    }
  }
  
  if (success) {
    showAlertMessage('Hash已复制到剪贴板', 'success')
  } else {
    showAlertMessage('复制失败', 'error')
  }
}
</script>

<template>
  <div class="weibo-manager">
    <Transition name="slide-fade">
      <div v-if="showAlert" :class="['alert', `alert-${alertType}`]">
        <div class="flex items-center gap-2">
          <span v-if="alertType === 'success'">✓</span>
          <span v-else>✕</span>
          <span>{{ alertMessage }}</span>
        </div>
      </div>
    </Transition>
    
    <div v-if="currentView === 'list'" class="user-list-view">
      <button @click="emit('back-to-center')" class="back-button mb-6">
        <svg class="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
          <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M15 19l-7-7 7-7"/>
        </svg>
        <span>返回账号管理中心</span>
      </button>
      
      <div class="header-section mb-8">
        <h1 class="text-3xl font-bold mb-2">微博管理</h1>
        <p class="text-muted-foreground">管理你的微博搜索配置</p>
      </div>
      
      <div class="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4 mb-6">
        <Card 
          v-for="user in savedUsers" 
          :key="user.hash"
          class="user-card cursor-pointer hover:shadow-lg transition-shadow"
          @click="handleSelectUser(user)"
        >
          <div class="p-6">
            <div class="flex items-center gap-4 mb-4">
              <div class="w-12 h-12 rounded-full bg-gradient-to-br from-orange-500 to-red-600 flex items-center justify-center text-white text-xl font-bold">
                {{ user.uid?.[0] || 'W' }}
              </div>
              <div class="flex-1">
                <div class="font-medium">{{ user.uid || '未登录' }}</div>
                <div class="text-xs text-muted-foreground">
                  {{ user.last_login ? '最近登录: ' + formatDateTime(user.last_login) : 'UID ' + user.uid }}
                </div>
              </div>
            </div>
            <div class="text-sm text-primary flex items-center gap-1">
              <span>管理</span>
              <svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9 5l7 7-7 7"/>
              </svg>
            </div>
          </div>
        </Card>
        
        <Card class="add-card cursor-pointer hover:shadow-lg transition-shadow" @click="handleShowAddForm">
          <div class="p-6 h-full flex flex-col items-center justify-center text-center gap-3">
            <div class="w-12 h-12 rounded-full border-2 border-dashed border-primary flex items-center justify-center">
              <svg class="w-6 h-6 text-primary" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 4v16m8-8H4"/>
              </svg>
            </div>
            <div class="text-primary font-medium">添加微博账号</div>
            <div class="text-xs text-muted-foreground">配置新的微博搜索</div>
          </div>
        </Card>
      </div>
    </div>
    
    <div v-else-if="currentView === 'add'" class="add-view">
      <button 
        v-if="savedUsers.length === 0"
        @click="emit('back-to-center')" 
        class="back-button mb-6"
      >
        <svg class="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
          <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M15 19l-7-7 7-7"/>
        </svg>
        <span>返回账号管理中心</span>
      </button>
      
      <div class="max-w-md mx-auto">
        <Card>
          <div class="p-8">
            <div class="text-center mb-6">
              <div class="w-16 h-16 rounded-full bg-gradient-to-br from-orange-500 to-red-600 flex items-center justify-center text-white text-2xl font-bold mx-auto mb-4">
                <svg class="w-8 h-8" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                  <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M16 7a4 4 0 11-8 0 4 4 0 018 0zM12 14a7 7 0 00-7 7h14a7 7 0 00-7-7z"/>
                </svg>
              </div>
              <h2 class="text-2xl font-bold mb-2">添加微博账号</h2>
              <p class="text-muted-foreground text-sm">输入微博UID开始配置搜索</p>
            </div>
            
            <div class="space-y-4">
              <div>
                <label class="block text-sm font-medium mb-2">微博UID</label>
                <Input 
                  v-model="uid" 
                  placeholder="请输入你的微博UID" 
                  type="text"
                  @keyup.enter="handleAddUID"
                  class="text-center text-lg"
                />
                <p class="text-xs text-muted-foreground mt-2">
                  系统会生成专属hash保护你的隐私，UID不会被存储
                </p>
              </div>
              
              <div class="flex gap-2">
                <Button 
                  v-if="savedUsers.length > 0"
                  @click="handleBackToList" 
                  variant="outline"
                  class="flex-1 h-12"
                >
                  返回
                </Button>
                <Button 
                  @click="handleAddUID" 
                  :disabled="!uid.trim() || generatingHash"
                  class="flex-1 h-12"
                >
                  {{ generatingHash ? '获取中...' : '确定' }}
                </Button>
              </div>
            </div>
          </div>
        </Card>
      </div>
    </div>
    
    <div v-else-if="currentView === 'manage'" class="manage-view">
      <div class="flex items-center justify-between mb-6">
        <button @click="handleBackToList" class="flex items-center gap-2 text-muted-foreground hover:text-foreground transition-colors">
          <svg class="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M15 19l-7-7 7-7"/>
          </svg>
          <span>返回列表</span>
        </button>
        
        <button 
          v-if="selectedUser"
          @click="handleDeleteAccount" 
          class="text-red-500 hover:text-red-600 text-sm transition-colors"
        >
          删除此账号
        </button>
      </div>
      
      <div class="grid grid-cols-1 lg:grid-cols-3 gap-6">
        <div class="space-y-6 lg:col-span-1">
          <Card>
            <div class="p-6">
              <h3 class="text-lg font-semibold mb-4 flex items-center gap-2">
                <span>📱</span>
                <span>登录状态</span>
              </h3>
              
              <div v-if="isLoggedIn" class="space-y-4">
                <div class="space-y-2 text-sm">
                  <div class="flex justify-between items-center">
                    <span class="text-muted-foreground">状态</span>
                    <div class="inline-flex items-center gap-1 px-2 py-1 bg-green-100 text-green-700 rounded-full text-xs">
                      <span class="w-2 h-2 bg-green-500 rounded-full"></span>
                      <span>已登录</span>
                    </div>
                  </div>
                  <div class="flex justify-between">
                    <span class="text-muted-foreground">UID</span>
                    <span>{{ status.uid }}</span>
                  </div>
                  <div class="flex justify-between">
                    <span class="text-muted-foreground">登录时间</span>
                    <span class="text-right text-xs">{{ formatDateTime(status.login_time) }}</span>
                  </div>
                  <div class="flex justify-between">
                    <span class="text-muted-foreground">有效期</span>
                    <span class="text-right text-xs">{{ status.expire_time ? formatDateTime(status.expire_time) : '-' }}</span>
                  </div>
                </div>
                
                <Button @click="handleLogout" variant="outline" class="w-full" size="sm">
                  退出登录
                </Button>
              </div>
              
              <div v-else class="space-y-4">
                <div class="flex flex-col items-center gap-3">
                  <div class="qrcode-wrapper">
                    <img 
                      v-if="qrcodeImage" 
                      :src="qrcodeImage" 
                      alt="登录二维码"
                      class="qrcode-image"
                    />
                    <div v-else class="qrcode-placeholder">
                      <div class="animate-spin rounded-full h-8 w-8 border-b-2 border-primary"></div>
                    </div>
                  </div>
                  <div class="text-center">
                    <p class="text-sm font-medium mb-1">扫码登录</p>
                    <p class="text-xs text-muted-foreground">使用微博APP扫描二维码</p>
                  </div>
                  <Button @click="handleRefreshQRCode" variant="outline" size="sm" class="w-full">
                    刷新二维码
                  </Button>
                </div>
              </div>
            </div>
          </Card>
          
          <Card>
            <div class="p-6">
              <div class="flex items-center justify-between mb-4">
                <h3 class="text-lg font-semibold flex items-center gap-2">
                  <span>👤</span>
                  <span>微博UID管理</span>
                  <span class="text-sm text-muted-foreground font-normal">({{ status.user_ids?.length || 0 }} 个)</span>
                </h3>
                <a 
                  href="https://github.com/fish2018/pansou/issues/4" 
                  target="_blank" 
                  rel="noopener noreferrer"
                  class="channel-link"
                  title="查看更多微博UID配置"
                >
                  <svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                    <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M10 6H6a2 2 0 00-2 2v10a2 2 0 002 2h10a2 2 0 002-2v-4M14 4h6m0 0v6m0-6L10 14"/>
                  </svg>
                  <span class="link-text">更多UID</span>
                </a>
              </div>
              
              <div v-if="!isLoggedIn" class="text-center py-8 text-muted-foreground">
                <svg class="w-12 h-12 mx-auto mb-3 opacity-50" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                  <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 15v2m-6 4h12a2 2 0 002-2v-6a2 2 0 00-2-2H6a2 2 0 00-2 2v6a2 2 0 002 2zm10-10V7a4 4 0 00-8 0v4h8z"/>
                </svg>
                <p>请先登录后配置用户ID</p>
              </div>
              
              <div v-else class="space-y-3">
                <div class="text-sm text-muted-foreground">
                  每行一个微博用户ID，将搜索这些用户的微博
                </div>
                <textarea 
                  v-model="userIdsText" 
                  rows="10"
                  class="user-ids-textarea"
                  placeholder="1234567890
2345678901
3456789012"
                  @focus="isEditingUserIds = true"
                  @blur="isEditingUserIds = false"
                />
                <Button @click="handleSaveUserIds" class="save-user-ids-button">
                  💾 保存配置
                </Button>
              </div>
            </div>
          </Card>
        </div>
        
        <div class="space-y-6 lg:col-span-2">
          <Card>
            <details class="api-docs-section p-2">
              <summary class="py-4 px-4 cursor-pointer font-semibold flex items-center justify-center hover:bg-muted/30 transition-colors rounded-t">
                <div class="flex items-center gap-2">
                  <span>📖</span>
                  <span>API文档</span>
                </div>
                <svg class="w-5 h-5 transform transition-transform details-icon ml-2" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                  <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M19 9l-7 7-7-7"/>
                </svg>
              </summary>
              
              <div class="p-6 pt-4 space-y-4">
                <div class="p-4 bg-muted/30 rounded-lg border border-border">
                  <div class="flex items-center justify-between mb-3">
                    <div class="text-sm text-muted-foreground font-medium">当前Hash</div>
                    <button 
                      @click="copyHashToClipboard"
                      class="flex items-center gap-1 px-2 py-1 text-xs bg-primary/10 hover:bg-primary/20 text-primary rounded-md transition-colors"
                    >
                      <svg class="w-3 h-3" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                        <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M8 16H6a2 2 0 01-2-2V6a2 2 0 012-2h8a2 2 0 012 2v2m-6 12h8a2 2 0 002-2v-8a2 2 0 00-2-2h-8a2 2 0 00-2 2v8a2 2 0 002 2z"/>
                      </svg>
                      <span>复制</span>
                    </button>
                  </div>
                  <div class="font-mono text-sm text-foreground leading-relaxed break-all">{{ currentHash }}</div>
                </div>
                
                <div class="space-y-3">
                  <div class="text-sm font-medium text-muted-foreground">API调用示例</div>
                  
                  <details class="api-detail">
                    <summary class="cursor-pointer text-sm p-2 hover:bg-muted/50 rounded transition-colors">
                      获取状态
                    </summary>
                    <div class="mt-2 p-3 bg-muted/50 rounded-lg">
                      <pre class="text-xs overflow-x-auto"><code>curl -X POST http://localhost:8888/weibo/{{ currentHash.slice(0, 16) }}... \
  -H "Content-Type: application/json" \
  -d '{"action": "get_status"}'</code></pre>
                    </div>
                  </details>
                  
                  <details class="api-detail">
                    <summary class="cursor-pointer text-sm p-2 hover:bg-muted/50 rounded transition-colors">
                      设置用户ID
                    </summary>
                    <div class="mt-2 p-3 bg-muted/50 rounded-lg">
                      <pre class="text-xs overflow-x-auto"><code>curl -X POST http://localhost:8888/weibo/{{ currentHash.slice(0, 16) }}... \
  -H "Content-Type: application/json" \
  -d '{"action": "set_user_ids", "user_ids": ["1234567890"]}'</code></pre>
                    </div>
                  </details>
                  
                  <details class="api-detail">
                    <summary class="cursor-pointer text-sm p-2 hover:bg-muted/50 rounded transition-colors">
                      测试搜索
                    </summary>
                    <div class="mt-2 p-3 bg-muted/50 rounded-lg">
                      <pre class="text-xs overflow-x-auto"><code>curl -X POST http://localhost:8888/weibo/{{ currentHash.slice(0, 16) }}... \
  -H "Content-Type: application/json" \
  -d '{"action": "test_search", "keyword": "遮天"}'</code></pre>
                    </div>
                  </details>
                </div>
              </div>
            </details>
          </Card>
          
          <Card class="search-card">
            <div class="p-6 flex flex-col h-full">
              <h3 class="text-lg font-semibold mb-4 flex items-center gap-2">
                <span>🔍</span>
                <span>测试搜索</span>
              </h3>
              
              <div v-if="!isLoggedIn" class="text-center py-16 text-muted-foreground flex-1 flex items-center justify-center">
                <div>
                  <svg class="w-16 h-16 mx-auto mb-3 opacity-30" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                    <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 15v2m-6 4h12a2 2 0 002-2v-6a2 2 0 00-2-2H6a2 2 0 00-2 2v6a2 2 0 002 2zm10-10V7a4 4 0 00-8 0v4h8z"/>
                  </svg>
                  <p>请先登录后使用搜索功能</p>
                </div>
              </div>
              
              <div v-else class="space-y-4 flex-1 flex flex-col">
                <div class="flex gap-3">
                  <Input 
                    v-model="searchKeyword" 
                    placeholder="输入关键词测试搜索"
                    class="flex-1"
                    @keyup.enter="handleTestSearch"
                  />
                  <Button 
                    @click="handleTestSearch" 
                    :disabled="searching || !searchKeyword.trim()"
                    class="search-button"
                  >
                    <span v-if="searching">搜索中...</span>
                    <span v-else>搜索</span>
                  </Button>
                </div>
                
                <div v-if="searchResults.length > 0" class="search-results-container flex-1">
                  <div class="text-sm text-muted-foreground mb-3">
                    找到 <strong class="text-foreground">{{ searchResults.length }}</strong> 条结果
                  </div>
                  
                  <div class="search-results-list">
                    <div 
                      v-for="(result, index) in searchResults" 
                      :key="result.unique_id"
                      class="result-item"
                    >
                      <div class="font-medium text-sm mb-2">
                        {{ index + 1 }}. {{ result.title }}
                      </div>
                      <div 
                        v-for="link in result.links" 
                        :key="link.url"
                        class="link-item"
                      >
                        <span class="link-type">{{ link.type }}</span>
                        <span class="link-url">{{ link.url }}</span>
                        <span v-if="link.password" class="link-password">
                          🔑 {{ link.password }}
                        </span>
                      </div>
                    </div>
                  </div>
                </div>
              </div>
            </div>
          </Card>
        </div>
      </div>
    </div>
  </div>
</template>

<style scoped>
.weibo-manager {
  max-width: 1200px;
  margin: 0 auto;
  padding: 0 1rem;
}

.alert {
  position: fixed;
  top: 24px;
  right: 24px;
  padding: 12px 20px;
  border-radius: 8px;
  font-size: 14px;
  font-weight: 500;
  box-shadow: 0 4px 12px rgba(0, 0, 0, 0.15);
  z-index: 1000;
  backdrop-filter: blur(8px);
}

.alert-success {
  background: rgba(16, 185, 129, 0.1);
  color: #059669;
  border: 1px solid rgba(16, 185, 129, 0.3);
}

.alert-error {
  background: rgba(239, 68, 68, 0.1);
  color: #dc2626;
  border: 1px solid rgba(239, 68, 68, 0.3);
}

.slide-fade-enter-active {
  transition: all 0.3s ease-out;
}

.slide-fade-leave-active {
  transition: all 0.2s ease-in;
}

.slide-fade-enter-from {
  transform: translateX(20px);
  opacity: 0;
}

.slide-fade-leave-to {
  transform: translateX(20px);
  opacity: 0;
}

.user-card {
  transition: all 0.2s ease;
}

.user-card:hover {
  transform: translateY(-2px);
}

.add-card {
  border: 2px dashed hsl(var(--border));
  background: hsl(var(--muted) / 0.3);
}

.add-card:hover {
  border-color: hsl(var(--primary));
  background: hsl(var(--primary) / 0.05);
}

.qrcode-wrapper {
  width: 200px;
  height: 200px;
  display: flex;
  align-items: center;
  justify-content: center;
  border: 2px solid hsl(var(--border));
  border-radius: 12px;
  overflow: hidden;
  background: hsl(var(--background));
}

.qrcode-image {
  width: 100%;
  height: 100%;
  object-fit: contain;
}

.qrcode-placeholder {
  display: flex;
  align-items: center;
  justify-content: center;
  width: 100%;
  height: 100%;
}

.api-docs-section {
  display: flex;
  flex-direction: column;
}

.api-docs-section summary {
  user-select: none;
  display: flex;
  align-items: center;
  justify-content: space-between;
}

.api-docs-section[open] summary {
  border-bottom: 1px solid hsl(var(--border));
}

.api-docs-section[open] .details-icon {
  transform: rotate(180deg);
}

.api-docs-section > div {
  flex: 1;
  display: flex;
  flex-direction: column;
}

.api-detail summary {
  transition: all 0.2s ease;
  user-select: none;
  font-weight: 500;
}

.api-detail[open] summary {
  color: hsl(var(--primary));
}

.api-detail pre {
  margin: 0;
  font-family: ui-monospace, monospace;
  line-height: 1.5;
}

.api-detail code {
  color: hsl(var(--foreground));
  font-size: 11px;
}

.user-ids-textarea {
  width: 100%;
  padding: 12px;
  border: 1px solid hsl(var(--border));
  border-radius: 8px;
  background: hsl(var(--background));
  color: hsl(var(--foreground));
  font-family: ui-monospace, monospace;
  font-size: 13px;
  line-height: 1.6;
  resize: none;
  transition: all 0.2s ease;
}

.user-ids-textarea:focus {
  outline: none;
  border-color: hsl(var(--primary));
  box-shadow: 0 0 0 3px hsl(var(--primary) / 0.1);
}

.user-ids-textarea::placeholder {
  color: hsl(var(--muted-foreground));
}

.search-card {
  height: fit-content;
  max-height: calc(100vh - 300px);
  display: flex;
  flex-direction: column;
}

.search-results-container {
  display: flex;
  flex-direction: column;
  min-height: 0;
}

.search-results-list {
  overflow-y: auto;
  max-height: 400px;
  padding-right: 8px;
  scrollbar-width: thin;
}

.search-results-list::-webkit-scrollbar {
  width: 6px;
}

.search-results-list::-webkit-scrollbar-track {
  background: hsl(var(--muted) / 0.3);
  border-radius: 3px;
}

.search-results-list::-webkit-scrollbar-thumb {
  background: hsl(var(--muted-foreground) / 0.3);
  border-radius: 3px;
}

.search-results-list::-webkit-scrollbar-thumb:hover {
  background: hsl(var(--muted-foreground) / 0.5);
}

.result-item {
  padding: 12px;
  background: hsl(var(--muted) / 0.3);
  border-radius: 8px;
  border: 1px solid hsl(var(--border));
  transition: all 0.2s ease;
  margin-bottom: 12px;
}

.result-item:hover {
  background: hsl(var(--muted) / 0.5);
  border-color: hsl(var(--primary) / 0.3);
}

.link-item {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 6px 8px;
  font-size: 12px;
  background: hsl(var(--background));
  border-radius: 4px;
  margin-top: 4px;
}

.link-type {
  display: inline-flex;
  align-items: center;
  padding: 2px 6px;
  background: hsl(var(--primary) / 0.1);
  color: hsl(var(--primary));
  border-radius: 4px;
  font-weight: 600;
  font-size: 11px;
  text-transform: uppercase;
  min-width: 50px;
  justify-content: center;
}

.link-url {
  flex: 1;
  color: hsl(var(--muted-foreground));
  word-break: break-all;
}

.link-password {
  color: hsl(var(--primary));
  font-weight: 500;
}

.search-button {
  min-width: 100px;
  padding-left: 24px;
  padding-right: 24px;
  font-weight: 600;
}

.save-user-ids-button {
  width: 100%;
  padding-top: 12px;
  padding-bottom: 12px;
  font-weight: 600;
  font-size: 15px;
}

@media (max-width: 1024px) {
  .weibo-manager .grid {
    grid-template-columns: 1fr;
  }
  
  .weibo-manager .lg\:col-span-1,
  .weibo-manager .lg\:col-span-2 {
    grid-column: span 1;
  }
  
  .search-card {
    min-height: auto;
    max-height: none;
  }
  
  .search-results-list {
    max-height: 500px;
  }
}

@media (max-width: 768px) {
  .weibo-manager {
    padding: 0 0.5rem;
  }
  
  .alert {
    left: 12px;
    right: 12px;
    top: 12px;
  }
  
  .qrcode-wrapper {
    width: 160px;
    height: 160px;
  }
  
  .search-button {
    min-width: 80px;
    padding-left: 16px;
    padding-right: 16px;
  }
  
  .search-results-list {
    max-height: 400px;
  }
}

.back-button {
  display: flex;
  align-items: center;
  gap: 0.5rem;
  padding: 0.5rem 1rem;
  background: transparent;
  color: hsl(var(--muted-foreground));
  border: 1px solid hsl(var(--border));
  border-radius: 0.5rem;
  font-size: 0.875rem;
  cursor: pointer;
  transition: all 0.2s ease;
}

.back-button:hover {
  background: hsl(var(--accent));
  color: hsl(var(--accent-foreground));
  border-color: hsl(var(--accent));
}

.user-list-view,
.add-view,
.manage-view {
  animation: fadeIn 0.3s ease-out;
}

@keyframes fadeIn {
  from {
    opacity: 0;
    transform: translateY(10px);
  }
  to {
    opacity: 1;
    transform: translateY(0);
  }
}
</style>
