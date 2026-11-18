<script setup lang="ts">
import { ref, onMounted } from 'vue';
import { getHealth, type HealthStatus } from '@/api';

// 状态数据
const healthData = ref<HealthStatus | null>(null);
const loading = ref(true);
const error = ref<string | null>(null);

// 复制状态
const copySuccess = ref(false);
const copyTimeout = ref<number | null>(null);

// 环境变量折叠状态
const envExpanded = ref(false);

// 获取健康状态
const fetchHealth = async () => {
  try {
    loading.value = true;
    error.value = null;
    healthData.value = await getHealth();
  } catch (err) {
    error.value = '获取状态失败';
    console.error('获取健康状态失败:', err);
  } finally {
    loading.value = false;
  }
};

// 获取当前时间
const getCurrentTime = () => {
  return new Date().toLocaleString('zh-CN', {
    year: 'numeric',
    month: '2-digit',
    day: '2-digit',
    hour: '2-digit',
    minute: '2-digit',
    second: '2-digit'
  });
};

// 生成环境变量字符串
const generateEnvString = () => {
  if (!healthData.value) return '';
  
  const channels = healthData.value.channels.join(',');
  return `export CHANNELS=${channels}`;
};

// 复制环境变量（支持降级处理）
const copyEnvVariable = async () => {
  const envString = generateEnvString();
  let success = false;
  
  // 方法1: 尝试使用现代 Clipboard API (需要HTTPS)
  if (navigator.clipboard && navigator.clipboard.writeText) {
    try {
      await navigator.clipboard.writeText(envString);
      success = true;
    } catch (err) {
      console.warn('Clipboard API 失败，尝试降级方案:', err);
    }
  }
  
  // 方法2: 降级使用传统 execCommand 方法 (兼容HTTP)
  if (!success) {
    try {
      const textarea = document.createElement('textarea');
      textarea.value = envString;
      textarea.style.position = 'fixed';
      textarea.style.opacity = '0';
      textarea.style.left = '-9999px';
      document.body.appendChild(textarea);
      
      textarea.select();
      textarea.setSelectionRange(0, envString.length);
      
      success = document.execCommand('copy');
      document.body.removeChild(textarea);
    } catch (err) {
      console.error('复制失败:', err);
    }
  }
  
  if (success) {
    copySuccess.value = true;
    
    // 清除之前的定时器
    if (copyTimeout.value) {
      clearTimeout(copyTimeout.value);
    }
    
    // 3秒后重置状态
    copyTimeout.value = window.setTimeout(() => {
      copySuccess.value = false;
    }, 3000);
  } else {
    alert('复制失败，请手动复制');
  }
};



// 组件挂载时获取数据
onMounted(() => {
  fetchHealth();
});
</script>

<template>
  <div class="api-status-container">
    <!-- 头部标题 -->
    <div class="status-header">
      <h1 class="status-title">
        <span class="title-icon">📊</span>
        PanSou API 状态监控
      </h1>
    </div>

    <!-- 加载状态 -->
    <div v-if="loading" class="loading-state">
      <div class="loading-spinner"></div>
      <p>获取状态信息中...</p>
    </div>

    <!-- 错误状态 -->
    <div v-else-if="error" class="error-state">
      <div class="error-icon">❌</div>
      <h3>获取状态失败</h3>
      <p>{{ error }}</p>
      <button @click="fetchHealth" class="retry-btn">重试</button>
    </div>

    <!-- 状态信息 -->
    <div v-else-if="healthData" class="status-content">
      <!-- 系统状态卡片 -->
      <div class="status-card">
        <div class="card-header">
          <h2 class="card-title">
            <span class="status-indicator" :class="{ 'healthy': healthData.status === 'ok' }"></span>
            系统状态
          </h2>
          <div class="status-badge" :class="{ 'healthy': healthData.status === 'ok' }">
            {{ healthData.status === 'ok' ? '正常' : '异常' }}
          </div>
        </div>
        
        <div class="card-content">
          <div class="status-info">
            <div class="status-item">
              <span class="status-label">状态:</span>
              <span class="status-value">{{ healthData.status }}</span>
            </div>
            <div class="status-item">
              <span class="status-label">插件:</span>
              <span class="status-value">{{ healthData.plugins_enabled ? '已启用' : '已禁用' }}</span>
            </div>
          </div>
        </div>
      </div>

      <!-- 频道配置卡片 -->
      <div class="status-card">
        <div class="card-header">
          <h2 class="card-title">
            <span class="channel-icon">📡</span>
            TG 频道配置
            <span class="count-badge">{{ healthData.channels.length }}</span>
          </h2>
        </div>
        
        <div class="card-content">
          <div class="channels-container">
            <div class="channels-grid">
              <div 
                v-for="channel in healthData.channels" 
                :key="channel"
                class="channel-tag"
              >
                {{ channel }}
              </div>
            </div>
          </div>
          
          <!-- 环境变量折叠区域 -->
          <div class="env-section">
            <button 
              @click="envExpanded = !envExpanded"
              class="env-toggle"
            >
              <span class="toggle-icon" :class="{ 'expanded': envExpanded }">▶</span>
              环境变量配置
            </button>
            
            <div v-show="envExpanded" class="env-content">
              <div class="env-variable">
                <code>{{ generateEnvString() }}</code>
              </div>
            </div>
          </div>
          
          <!-- 复制按钮放到底部 -->
          <div class="card-actions">
            <button 
              @click="copyEnvVariable" 
              class="copy-btn"
              :class="{ 'success': copySuccess }"
            >
              <span v-if="copySuccess">✅ 已复制</span>
              <span v-else>📋 复制环境变量</span>
            </button>
          </div>
        </div>
      </div>

      <!-- 插件状态卡片 -->
      <div class="status-card">
        <div class="card-header">
          <h2 class="card-title">
            <span class="plugin-icon">🧩</span>
            已加载插件
            <span class="count-badge">{{ healthData.plugin_count }}</span>
          </h2>
        </div>
        
        <div class="card-content">
          <div class="plugins-grid">
            <div 
              v-for="plugin in healthData.plugins" 
              :key="plugin"
              class="plugin-tag"
            >
              {{ plugin }}
            </div>
          </div>
        </div>
      </div>
    </div>
  </div>
</template>

<style scoped>
.api-status-container {
  max-width: 1200px;
  margin: 0 auto;
  padding: 2rem;
  background: #f8fafc;
  min-height: 100vh;
}

/* 头部样式 */
.status-header {
  display: flex;
  justify-content: center;
  align-items: center;
  margin-bottom: 2rem;
  padding-bottom: 1rem;
  border-bottom: 2px solid #e2e8f0;
}

.status-title {
  font-size: 2rem;
  font-weight: 700;
  color: #1a202c;
  display: flex;
  align-items: center;
  gap: 0.5rem;
}

.title-icon {
  font-size: 2.5rem;
}



/* 加载和错误状态 */
.loading-state, .error-state {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  padding: 4rem 2rem;
  text-align: center;
}

.loading-spinner {
  width: 3rem;
  height: 3rem;
  border: 4px solid #e2e8f0;
  border-top: 4px solid #3b82f6;
  border-radius: 50%;
  animation: spin 1s linear infinite;
  margin-bottom: 1rem;
}

.error-icon {
  font-size: 4rem;
  margin-bottom: 1rem;
}

.retry-btn {
  margin-top: 1rem;
  padding: 0.75rem 1.5rem;
  background: #ef4444;
  color: white;
  border: none;
  border-radius: 0.5rem;
  cursor: pointer;
  font-weight: 500;
}

/* 状态内容 */
.status-content {
  display: grid;
  gap: 1.5rem;
  grid-template-columns: repeat(auto-fit, minmax(350px, 1fr));
}

/* 卡片样式 */
.status-card {
  background: white;
  border-radius: 1rem;
  box-shadow: 0 4px 6px rgba(0, 0, 0, 0.07);
  overflow: hidden;
  transition: transform 0.2s ease, box-shadow 0.2s ease;
}

.status-card:hover {
  transform: translateY(-2px);
  box-shadow: 0 8px 25px rgba(0, 0, 0, 0.1);
}

.card-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 1.25rem 1.5rem;
  border-bottom: 1px solid #e2e8f0;
  background: #f8fafc;
  min-height: 4rem;
}

.card-title {
  font-size: 1.25rem;
  font-weight: 600;
  color: #374151;
  display: flex;
  align-items: center;
  gap: 0.5rem;
}

.card-content {
  padding: 1.25rem;
}

/* 状态指示器 */
.status-indicator {
  width: 0.75rem;
  height: 0.75rem;
  border-radius: 50%;
  margin-right: 0.5rem;
}

.status-indicator.healthy {
  background: #10b981;
  box-shadow: 0 0 0 2px rgba(16, 185, 129, 0.3);
}

.status-badge {
  padding: 0.25rem 0.75rem;
  border-radius: 9999px;
  font-size: 0.875rem;
  font-weight: 500;
}

.status-badge.healthy {
  background: #d1fae5;
  color: #065f46;
}

/* 状态信息布局 */
.status-info {
  display: flex;
  flex-direction: column;
  gap: 0.75rem;
}

.status-item {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 0.5rem 0;
  border-bottom: 1px solid #f3f4f6;
}

.status-item:last-child {
  border-bottom: none;
}

.status-label {
  font-size: 0.875rem;
  color: #6b7280;
  font-weight: 500;
}

.status-value {
  font-weight: 600;
  color: #374151;
}

/* 计数徽章 */
.count-badge {
  background: #3b82f6;
  color: white;
  font-size: 0.8rem;
  padding: 0.25rem 0.5rem;
  border-radius: 9999px;
  font-weight: 600;
}

/* 环境变量折叠区域 */
.env-section {
  margin-bottom: 1rem;
  border: 1px solid #e2e8f0;
  border-radius: 0.5rem;
  overflow: hidden;
}

.env-toggle {
  width: 100%;
  display: flex;
  align-items: center;
  gap: 0.5rem;
  padding: 0.75rem 1rem;
  background: #f8fafc;
  border: none;
  font-size: 0.875rem;
  font-weight: 500;
  color: #374151;
  cursor: pointer;
  transition: all 0.2s ease;
}

.env-toggle:hover {
  background: #f1f5f9;
}

.toggle-icon {
  font-size: 0.8rem;
  transition: transform 0.2s ease;
}

.toggle-icon.expanded {
  transform: rotate(90deg);
}

.env-content {
  padding: 1rem;
  background: white;
  border-top: 1px solid #e2e8f0;
}

/* 卡片底部操作区域 */
.card-actions {
  display: flex;
  justify-content: center;
  padding-top: 1rem;
  border-top: 1px solid #f3f4f6;
}

/* 复制按钮 */
.copy-btn {
  padding: 0.5rem 1.5rem;
  background: #10b981;
  color: white;
  border: none;
  border-radius: 0.375rem;
  font-size: 0.875rem;
  font-weight: 500;
  cursor: pointer;
  transition: all 0.2s ease;
}

.copy-btn:hover {
  background: #059669;
}

.copy-btn.success {
  background: #16a34a;
}

/* 环境变量 */
.env-variable {
  padding: 0.75rem;
  background: #f1f5f9;
  border-radius: 0.375rem;
  border: 1px solid #cbd5e1;
}

.env-variable code {
  font-family: 'Courier New', monospace;
  font-size: 0.8rem;
  color: #475569;
  word-break: break-all;
  line-height: 1.4;
}

/* 频道容器 - 添加滑动条 */
.channels-container {
  max-height: 120px;
  overflow-y: auto;
  margin-bottom: 1rem;
  padding-right: 0.5rem;
}

.channels-container::-webkit-scrollbar {
  width: 6px;
}

.channels-container::-webkit-scrollbar-track {
  background: #f1f5f9;
  border-radius: 3px;
}

.channels-container::-webkit-scrollbar-thumb {
  background: #cbd5e1;
  border-radius: 3px;
}

.channels-container::-webkit-scrollbar-thumb:hover {
  background: #94a3b8;
}

/* 频道和插件网格 */
.channels-grid, .plugins-grid {
  display: grid;
  gap: 0.5rem;
  grid-template-columns: repeat(auto-fill, minmax(110px, 1fr));
  grid-auto-rows: max-content;
}

/* 插件容器 - 铺满显示 */
.plugins-grid {
  min-height: 120px;
  max-height: 200px;
  overflow-y: auto;
  padding-right: 0.5rem;
  align-content: start;
}

.channel-tag, .plugin-tag {
  padding: 0.5rem 0.75rem;
  background: #eff6ff;
  color: #1e40af;
  border-radius: 0.375rem;
  font-size: 0.875rem;
  font-weight: 500;
  text-align: center;
  border: 1px solid #bfdbfe;
  transition: all 0.2s ease;
}

.plugin-tag {
  background: #f0fdf4;
  color: #15803d;
  border-color: #bbf7d0;
}

.channel-tag:hover, .plugin-tag:hover {
  transform: translateY(-1px);
  box-shadow: 0 2px 4px rgba(0, 0, 0, 0.1);
}

/* 响应式设计 */
@media (max-width: 768px) {
  .api-status-container {
    padding: 1rem;
  }
  
  .status-header {
    flex-direction: column;
    gap: 1rem;
    align-items: stretch;
  }
  
  .status-content {
    grid-template-columns: 1fr;
  }
  
  .status-info {
    gap: 0.5rem;
  }
  
  .status-item {
    padding: 0.375rem 0;
  }
  
  .status-title {
    font-size: 1.5rem;
    justify-content: center;
  }
  
  .card-header {
    flex-direction: column;
    gap: 1rem;
    align-items: stretch;
  }
  
  .channels-container {
    max-height: 100px;
  }
  
  .plugins-grid {
    min-height: 100px;
    max-height: 150px;
  }
}
</style>