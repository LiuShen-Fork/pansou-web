<template>
  <svg
    viewBox="0 0 24 24"
    fill="none"
    stroke="currentColor"
    :width="size"
    :height="size"
  >
    <circle
      cx="12"
      cy="12"
      r="9"
      stroke-width="2"
    />
    <path
      d="M9 18V6h4.6a3.6 3.6 0 0 1 0 7.2H9"
      stroke-width="2.2"
      stroke-linecap="round"
      stroke-linejoin="round"
    />
    <path
      d="M13.9 8.4h1.9"
      stroke-width="2.2"
      stroke-linecap="round"
      stroke-linejoin="round"
    />
    <path
      d="M13.9 10.8h1.2"
      stroke-width="2"
      stroke-linecap="round"
      stroke-linejoin="round"
    />
    <circle cx="16.6" cy="8.4" r="1" fill="currentColor" stroke="none" />
    <circle cx="15.8" cy="10.8" r="1" fill="currentColor" stroke="none" />
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
