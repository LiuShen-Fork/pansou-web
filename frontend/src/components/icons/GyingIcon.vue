<template>
  <svg 
    viewBox="0 0 24 24" 
    fill="none"
    stroke="currentColor"
    :width="size"
    :height="size"
  >
    <!-- 电影胶片外框 -->
    <rect x="2" y="3" width="20" height="18" rx="2" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"/>
    
    <!-- 胶片孔（左侧） -->
    <line x1="2" y1="7" x2="6" y2="7" stroke-width="2" stroke-linecap="round"/>
    <line x1="2" y1="12" x2="6" y2="12" stroke-width="2" stroke-linecap="round"/>
    <line x1="2" y1="17" x2="6" y2="17" stroke-width="2" stroke-linecap="round"/>
    
    <!-- 胶片孔（右侧） -->
    <line x1="18" y1="7" x2="22" y2="7" stroke-width="2" stroke-linecap="round"/>
    <line x1="18" y1="12" x2="22" y2="12" stroke-width="2" stroke-linecap="round"/>
    <line x1="18" y1="17" x2="22" y2="17" stroke-width="2" stroke-linecap="round"/>
    
    <!-- 播放按钮 -->
    <polygon points="10,9 10,15 15,12" fill="currentColor" stroke="none"/>
  </svg>
</template>

<script setup lang="ts">
interface Props {
  size?: string | number
}

withDefaults(defineProps<Props>(), {
  size: '24'
})
</script>

