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
    <circle cx="11" cy="11" r="8"/>
    <path d="m21 21-4.35-4.35"/>
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

