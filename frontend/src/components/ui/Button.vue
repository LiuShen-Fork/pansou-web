<template>
  <button 
    :class="buttonClasses"
    :disabled="disabled"
    @click="handleClick"
  >
    <span v-if="loading" class="loading-spinner mr-2"></span>
    <slot />
  </button>
</template>

<script setup lang="ts">
import { computed } from 'vue';

interface Props {
  variant?: 'default' | 'secondary' | 'outline' | 'ghost' | 'icon' | 'danger'
  size?: 'sm' | 'default' | 'lg'
  disabled?: boolean
  loading?: boolean
}

const props = withDefaults(defineProps<Props>(), {
  variant: 'default',
  size: 'default',
  disabled: false,
  loading: false
});

const emit = defineEmits<{
  click: [event: MouseEvent]
}>();

const buttonClasses = computed(() => {
  const baseClasses = 'btn';
  const variantClasses = {
    default: 'btn-default',
    secondary: 'btn-secondary', 
    outline: 'btn-outline',
    ghost: 'btn-ghost',
    icon: 'btn-icon',
    danger: 'btn-danger'
  };
  const sizeClasses = {
    sm: 'btn-sm',
    default: '',
    lg: 'btn-lg'
  };

  return [
    baseClasses,
    variantClasses[props.variant],
    sizeClasses[props.size],
    props.disabled && 'opacity-50 cursor-not-allowed',
    props.loading && 'pointer-events-none'
  ].filter(Boolean).join(' ');
});

const handleClick = (event: MouseEvent) => {
  if (!props.disabled && !props.loading) {
    emit('click', event);
  }
};
</script>