<template>
  <svg 
    viewBox="0 0 200 200" 
    fill="currentColor"
    :width="size"
    :height="size"
  >
    <!-- QQ频道官方logo -->
    <circle cx="100" cy="100" r="95" fill="none" stroke="currentColor" stroke-width="8"/>
    <!-- 左竖线 -->
    <path d="M70 50 L60 150" stroke="currentColor" stroke-width="18" stroke-linecap="round"/>
    <!-- 右竖线 -->
    <path d="M130 50 L120 150" stroke="currentColor" stroke-width="18" stroke-linecap="round"/>
    <!-- 上横线 -->
    <path d="M45 80 L155 80" stroke="currentColor" stroke-width="18" stroke-linecap="round"/>
    <!-- 下横线（较短，为三角留空间） -->
    <path d="M45 120 L130 120" stroke="currentColor" stroke-width="18" stroke-linecap="round"/>
    <!-- 右下角三角形 -->
    <path d="M145 110 L145 145 L110 145 Z" fill="currentColor"/>
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

