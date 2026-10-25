<template>
  <svg 
    viewBox="0 0 24 24" 
    fill="none"
    stroke="currentColor"
    :width="size"
    :height="size"
    stroke-width="2"
    stroke-linecap="round"
    stroke-linejoin="round"
  >
    <path d="M22 12h-4l-3 9L9 3l-3 9H2"/>
  </svg>
</template>

<script setup lang="ts">
interface Props {
  size?: string | number
}

withDefaults(defineProps<Props>(), {
  size: '20'
})
</script>

