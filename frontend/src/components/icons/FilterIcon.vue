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
    <polygon points="22 3 2 3 10 12.46 10 19 14 21 14 12.46 22 3"/>
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

