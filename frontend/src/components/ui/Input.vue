<template>
  <div class="relative">
    <input
      :class="inputClasses"
      :type="type"
      :placeholder="placeholder"
      :disabled="disabled"
      :value="modelValue"
      @input="handleInput"
      @focus="handleFocus"
      @blur="handleBlur"
      @keyup.enter="emit('enter')"
    />
    <div v-if="$slots.suffix" class="absolute right-3 top-1/2 -translate-y-1/2">
      <slot name="suffix" />
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue';

interface Props {
  modelValue?: string
  type?: 'text' | 'password' | 'email' | 'search'
  placeholder?: string
  disabled?: boolean
  variant?: 'default' | 'search'
}

const props = withDefaults(defineProps<Props>(), {
  type: 'text',
  variant: 'default'
});

const emit = defineEmits<{
  'update:modelValue': [value: string]
  'focus': [event: FocusEvent]
  'blur': [event: FocusEvent]
  'enter': []
}>();

const inputClasses = computed(() => {
  const baseClasses = 'input';
  const variantClasses = {
    default: '',
    search: 'input-search'
  };

  return [
    baseClasses,
    variantClasses[props.variant],
    props.disabled && 'opacity-50 cursor-not-allowed'
  ].filter(Boolean).join(' ');
});

const handleInput = (event: Event) => {
  const target = event.target as HTMLInputElement;
  emit('update:modelValue', target.value);
};

const handleFocus = (event: FocusEvent) => {
  emit('focus', event);
};

const handleBlur = (event: FocusEvent) => {
  emit('blur', event);
};
</script>