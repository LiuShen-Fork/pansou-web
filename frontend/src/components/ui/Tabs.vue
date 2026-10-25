<template>
  <div class="tabs">
    <div class="tabs-list">
      <button
        v-for="tab in tabs"
        :key="tab.value"
        :class="[
          'tabs-trigger',
          { 'data-[state=active]:bg-background data-[state=active]:text-foreground data-[state=active]:shadow-sm': activeTab === tab.value }
        ]"
        :data-state="activeTab === tab.value ? 'active' : 'inactive'"
        @click="setActiveTab(tab.value)"
      >
        <component v-if="tab.icon" :is="tab.icon" class="w-4 h-4 mr-1" />
        {{ tab.label }}
      </button>
    </div>
    <div class="tabs-content">
      <slot :activeTab="activeTab" />
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, watch } from 'vue';

interface Tab {
  label: string
  value: string
  icon?: any
}

interface Props {
  tabs: Tab[]
  defaultValue?: string
  modelValue?: string
}

const props = withDefaults(defineProps<Props>(), {
  defaultValue: ''
});

const emit = defineEmits<{
  'update:modelValue': [value: string]
  'change': [value: string]
}>();

const activeTab = ref(props.modelValue || props.defaultValue || props.tabs[0]?.value || '');

const setActiveTab = (value: string) => {
  activeTab.value = value;
  emit('update:modelValue', value);
  emit('change', value);
};

watch(() => props.modelValue, (newValue) => {
  if (newValue !== undefined) {
    activeTab.value = newValue;
  }
});
</script>