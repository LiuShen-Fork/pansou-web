<script setup lang="ts">
import type { Component } from 'vue'
import Card from '@/components/ui/Card.vue'

// 定义Props
interface Props {
  iconComponent?: Component
  iconEmoji?: string
  name: string
  description: string
  accountCount: number
  enabled: boolean
  statusText?: string
  externalLink?: string  // 新增：外部链接
}

const props = defineProps<Props>()

// 定义事件
const emit = defineEmits<{
  (e: 'manage'): void
}>()

// 打开外部链接
const openExternalLink = (e: MouseEvent) => {
  if (props.externalLink) {
    e.stopPropagation()
    window.open(props.externalLink, '_blank', 'noopener,noreferrer')
  }
}
</script>

<template>
  <Card 
    v-if="enabled"
    class="service-card" 
    @click="emit('manage')"
  >
    <div class="card-content">
      <div class="service-icon">
        <component v-if="iconComponent" :is="iconComponent" :size="40" />
        <span v-else-if="iconEmoji">{{ iconEmoji }}</span>
      </div>
      <div class="service-info">
        <div class="service-header">
          <h3 class="service-name">{{ name }}</h3>
          <button 
            v-if="externalLink" 
            @click="openExternalLink"
            class="external-link-btn"
            title="访问官网"
          >
            <svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M10 6H6a2 2 0 00-2 2v10a2 2 0 002 2h10a2 2 0 002-2v-4M14 4h6m0 0v6m0-6L10 14"/>
            </svg>
          </button>
        </div>
        <p class="service-description">{{ description }}</p>
        <div class="service-status">
          <span v-if="accountCount > 0" class="status-active">
            已登录 {{ accountCount }} 个账号
          </span>
          <span v-else class="status-empty">
            未配置
          </span>
          <span v-if="statusText" class="status-extra">
            {{ statusText }}
          </span>
        </div>
      </div>
      <div class="service-action">
        <svg class="w-6 h-6" fill="none" stroke="currentColor" viewBox="0 0 24 24">
          <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9 5l7 7-7 7"/>
        </svg>
      </div>
    </div>
  </Card>
</template>

<style scoped>
.service-card {
  cursor: pointer;
  transition: all 0.3s ease;
  border: 2px solid hsl(var(--border));
  background: hsl(var(--card));
}

.service-card:hover {
  transform: translateY(-4px);
  box-shadow: 0 8px 24px rgba(0, 0, 0, 0.12);
  border-color: hsl(var(--primary));
}

.card-content {
  display: flex;
  align-items: center;
  gap: 1.25rem;
  padding: 1.5rem;
}

.service-icon {
  display: flex;
  align-items: center;
  justify-content: center;
  font-size: 2.5rem;
  line-height: 1;
  flex-shrink: 0;
  width: 48px;
  height: 48px;
}

.service-info {
  flex: 1;
  min-width: 0;
}

.service-header {
  display: flex;
  align-items: center;
  gap: 0.5rem;
  margin-bottom: 0.25rem;
}

.service-name {
  font-size: 1.125rem;
  font-weight: 600;
  margin: 0;
  color: hsl(var(--foreground));
}

.external-link-btn {
  display: flex;
  align-items: center;
  justify-content: center;
  padding: 0.25rem;
  background: transparent;
  border: 1px solid hsl(var(--border));
  border-radius: 0.375rem;
  color: hsl(var(--muted-foreground));
  cursor: pointer;
  transition: all 0.2s ease;
  flex-shrink: 0;
}

.external-link-btn:hover {
  background: hsl(var(--primary));
  color: hsl(var(--primary-foreground));
  border-color: hsl(var(--primary));
  transform: translateY(-1px);
}

.service-description {
  font-size: 0.875rem;
  color: hsl(var(--muted-foreground));
  margin: 0 0 0.5rem 0;
}

.service-status {
  display: flex;
  flex-direction: column;
  gap: 0.25rem;
  font-size: 0.875rem;
}

.status-active {
  color: hsl(142, 76%, 36%);
  font-weight: 500;
}

.status-empty {
  color: hsl(var(--muted-foreground));
}

.status-extra {
  color: hsl(var(--muted-foreground));
  font-size: 0.8125rem;
}

.service-action {
  color: hsl(var(--primary));
  flex-shrink: 0;
  transition: transform 0.2s ease;
}

.service-card:hover .service-action {
  transform: translateX(4px);
}

/* 响应式 */
@media (max-width: 768px) {
  .card-content {
    padding: 1rem;
    gap: 1rem;
  }
  
  .service-icon {
    font-size: 2rem;
    width: 40px;
    height: 40px;
  }
  
  .service-name {
    font-size: 1rem;
  }
  
  .service-description {
    font-size: 0.8125rem;
  }
}
</style>

