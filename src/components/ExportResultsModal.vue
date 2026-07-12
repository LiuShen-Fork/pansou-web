<script setup lang="ts">
import { computed, onMounted, onUnmounted, ref, watch } from 'vue';
import { Icons } from '@/components/ui';
import type { ExportField, ExportFormat, ExportSettings } from '@/types';
import { getDiskTypeName, sortDiskTypes } from '@/utils/diskTypes';

const props = defineProps<{
  visible: boolean;
  total: number;
  settings: ExportSettings;
  availableDiskTypes: string[];
}>();

const emit = defineEmits<{
  (e: 'close'): void;
  (e: 'update:settings', value: ExportSettings): void;
  (e: 'confirm'): void;
}>();

const fieldOptions: Array<{ value: ExportField; label: string }> = [
  { value: 'sequence', label: '序号' },
  { value: 'title', label: '标题' },
  { value: 'source', label: '来源' },
  { value: 'datetime', label: '时间' }
];

const localSettings = ref<ExportSettings>({
  format: 'json',
  fields: ['title', 'source', 'datetime'],
  prettyJson: true,
  includeFieldLabels: true,
  selectedDiskTypes: [],
  allDiskTypesSelected: true
});

const allDiskTypesSelected = computed(() => localSettings.value.allDiskTypesSelected);
const availableDiskTypes = computed(() => sortDiskTypes(props.availableDiskTypes));

const syncFromProps = () => {
  const selectedDiskTypes = props.settings.allDiskTypesSelected
    ? [...availableDiskTypes.value]
    : sortDiskTypes(props.settings.selectedDiskTypes);

  localSettings.value = {
    ...props.settings,
    fields: [...props.settings.fields],
    selectedDiskTypes
  };
};

watch(() => [props.settings, props.availableDiskTypes], syncFromProps, { immediate: true, deep: true });

watch(() => props.visible, (visible) => {
  document.body.style.overflow = visible ? 'hidden' : '';
});

const emitSettingsUpdate = () => {
  emit('update:settings', {
    ...localSettings.value,
    fields: [...localSettings.value.fields]
  });
};

const setFormat = (format: ExportFormat) => {
  localSettings.value.format = format;
  emitSettingsUpdate();
};

const toggleField = (field: ExportField) => {
  const fields = localSettings.value.fields;
  const exists = fields.includes(field);

  if (exists) {
    localSettings.value.fields = fields.filter(item => item !== field);
    emitSettingsUpdate();
    return;
  }

  localSettings.value.fields = [...fields, field];
  emitSettingsUpdate();
};

const selectRecommendedFields = () => {
  localSettings.value.fields = ['title', 'source', 'datetime'];
  emitSettingsUpdate();
};

const selectMinimalFields = () => {
  localSettings.value.fields = ['title'];
  emitSettingsUpdate();
};

const togglePrettyJson = () => {
  if (localSettings.value.format !== 'json') return;
  localSettings.value.prettyJson = !localSettings.value.prettyJson;
  emitSettingsUpdate();
};

const toggleIncludeFieldLabels = () => {
  if (localSettings.value.format !== 'txt') return;
  localSettings.value.includeFieldLabels = !localSettings.value.includeFieldLabels;
  emitSettingsUpdate();
};

const toggleDiskType = (diskType: string) => {
  const selected = [...localSettings.value.selectedDiskTypes];
  const exists = selected.includes(diskType);

  localSettings.value.selectedDiskTypes = exists
    ? selected.filter(item => item !== diskType)
    : [...selected, diskType];
  localSettings.value.allDiskTypesSelected = false;

  emitSettingsUpdate();
};

const toggleAllDiskTypes = () => {
  if (allDiskTypesSelected.value) {
    localSettings.value.selectedDiskTypes = [];
    localSettings.value.allDiskTypesSelected = false;
  } else {
    localSettings.value.selectedDiskTypes = [...availableDiskTypes.value];
    localSettings.value.allDiskTypesSelected = true;
  }

  emitSettingsUpdate();
};

const handleKeydown = (event: KeyboardEvent) => {
  if (event.key === 'Escape' && props.visible) {
    emit('close');
  }
};

const handleConfirm = () => {
  emit('confirm');
};

onMounted(() => {
  window.addEventListener('keydown', handleKeydown);
});

onUnmounted(() => {
  window.removeEventListener('keydown', handleKeydown);
  document.body.style.overflow = '';
});
</script>

<template>
  <Teleport to="body">
    <Transition name="export-modal-fade">
      <div v-if="visible" class="export-modal-overlay" @click="emit('close')">
        <div class="export-modal" @click.stop>
            <div class="export-modal-header">
              <div class="export-modal-heading">
              <h2 class="export-modal-title">导出搜索结果</h2>
            </div>
            <button
              type="button"
              class="export-modal-close"
              aria-label="关闭导出窗口"
              @click="emit('close')"
            >
              <svg class="export-modal-close-icon" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M6 18L18 6M6 6l12 12"></path>
              </svg>
            </button>
          </div>

          <div class="export-modal-body">
            <section class="export-section">
              <div class="section-inline-row">
                <h3 class="section-title">导出格式</h3>
                <div class="inline-control-group format-inline-group">
                  <label class="compact-choice">
                    <input
                      type="radio"
                      name="export-format"
                      class="compact-input"
                      :checked="localSettings.format === 'json'"
                      @change="setFormat('json')"
                    />
                    <span>JSON</span>
                  </label>
                  <label class="compact-choice">
                    <input
                      type="radio"
                      name="export-format"
                      class="compact-input"
                      :checked="localSettings.format === 'txt'"
                      @change="setFormat('txt')"
                    />
                    <span>TXT</span>
                  </label>
                </div>
              </div>
            </section>

            <section class="export-section">
              <div class="section-inline-row">
                <h3 class="section-title">网盘类型</h3>
                <label class="compact-choice compact-checkbox">
                  <input
                    type="checkbox"
                    class="compact-input"
                    :checked="allDiskTypesSelected"
                    @change="toggleAllDiskTypes"
                  />
                  <span>全选</span>
                </label>
              </div>
              <div class="disktype-grid">
                <label
                  v-for="diskType in availableDiskTypes"
                  :key="diskType"
                  class="disktype-chip"
                  :class="{ selected: localSettings.selectedDiskTypes.includes(diskType) }"
                >
                  <input
                    class="field-checkbox"
                    type="checkbox"
                    :checked="localSettings.selectedDiskTypes.includes(diskType)"
                    @change="toggleDiskType(diskType)"
                  />
                  <span :class="{ selected: localSettings.selectedDiskTypes.includes(diskType) }">{{ getDiskTypeName(diskType) }}</span>
                </label>
              </div>
            </section>

            <section class="export-section">
              <div class="section-inline-row">
                <h3 class="section-title">字段选择</h3>
                <div class="inline-control-group field-inline-group">
                  <label
                    v-for="option in fieldOptions"
                    :key="option.value"
                    class="compact-choice compact-checkbox"
                  >
                    <input
                      class="compact-input"
                      type="checkbox"
                      :checked="localSettings.fields.includes(option.value)"
                      @change="toggleField(option.value)"
                    />
                    <span>{{ option.label }}</span>
                  </label>
                </div>
              </div>
            </section>

            <section class="export-section">
              <div class="section-inline-row">
                <h3 class="section-title">导出选项</h3>
                <div class="inline-control-group option-inline-group">
                  <label
                    v-if="localSettings.format === 'json'"
                    class="compact-choice compact-checkbox"
                    :class="{ disabled: localSettings.format !== 'json' }"
                  >
                    <input
                      type="checkbox"
                      class="compact-input"
                      :checked="localSettings.prettyJson"
                      :disabled="localSettings.format !== 'json'"
                      @change="togglePrettyJson"
                    />
                    <span>JSON 美化缩进</span>
                  </label>

                  <label
                    v-else
                    class="compact-choice compact-checkbox"
                    :class="{ disabled: localSettings.format !== 'txt' }"
                  >
                    <input
                      type="checkbox"
                      class="compact-input"
                      :checked="localSettings.includeFieldLabels"
                      :disabled="localSettings.format !== 'txt'"
                      @change="toggleIncludeFieldLabels"
                    />
                    <span>TXT 显示字段名</span>
                  </label>
                </div>
              </div>
            </section>
          </div>

          <div class="export-modal-footer">
            <button type="button" class="footer-btn footer-btn-secondary" @click="emit('close')">取消</button>
            <button
              type="button"
              class="footer-btn footer-btn-primary"
              @click="handleConfirm"
            >
              导出 {{ localSettings.format.toUpperCase() }}
            </button>
          </div>
        </div>
      </div>
    </Transition>
  </Teleport>
</template>

<style scoped>
.export-modal-overlay {
  position: fixed;
  inset: 0;
  z-index: 1100;
  display: flex;
  align-items: center;
  justify-content: center;
  padding: 1rem;
  background: rgba(15, 23, 42, 0.52);
  backdrop-filter: blur(8px);
}

.export-modal {
  width: min(100%, 900px);
  max-height: min(84vh, 760px);
  display: flex;
  flex-direction: column;
  overflow: hidden;
  border: 1px solid rgba(148, 163, 184, 0.22);
  border-radius: 1.25rem;
  background:
    linear-gradient(180deg, rgba(255, 255, 255, 0.98), rgba(248, 250, 252, 0.98));
  box-shadow: 0 28px 90px rgba(15, 23, 42, 0.22);
}

.export-modal-header {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  gap: 1rem;
  padding: 1rem 1rem 0.8rem;
  border-bottom: 1px solid #e2e8f0;
}

.export-modal-heading {
  min-width: 0;
}

.export-modal-title {
  margin: 0;
  font-size: 1.1rem;
  line-height: 1.3;
  font-weight: 700;
  color: #0f172a;
}

.export-modal-close {
  appearance: none;
  border: 1px solid #dbe3ef;
  background: rgba(255, 255, 255, 0.92);
  color: #64748b;
  width: 2.25rem;
  height: 2.25rem;
  border-radius: 9999px;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  cursor: pointer;
  flex-shrink: 0;
  transition: all 0.2s ease;
}

.export-modal-close:hover {
  color: #0f172a;
  border-color: #cbd5e1;
  background: #f8fafc;
}

.export-modal-close,
.footer-btn,
.compact-choice,
.disktype-chip {
  -webkit-tap-highlight-color: transparent;
}

.export-modal-close:focus,
.export-modal-close:focus-visible,
.footer-btn:focus,
.footer-btn:focus-visible,
.compact-choice:focus,
.compact-choice:focus-visible,
.compact-choice:focus-within,
.disktype-chip:focus,
.disktype-chip:focus-visible,
.disktype-chip:focus-within {
  outline: none;
  box-shadow: none;
}

.export-modal-close-icon {
  width: 1rem;
  height: 1rem;
}

.export-modal-body {
  flex: 1;
  overflow-y: auto;
  padding: 0.9rem 1rem 1rem;
}

.export-section + .export-section {
  margin-top: 0.75rem;
}

.export-section {
  border: 1px solid #e2e8f0;
  border-radius: 0.9rem;
  background: rgba(255, 255, 255, 0.82);
  padding: 0.85rem;
}

.section-title-row,
.section-inline-row {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 0.75rem;
  margin-bottom: 0.55rem;
}

.section-title {
  margin: 0;
  font-size: 0.9rem;
  font-weight: 700;
  color: #0f172a;
}

.section-inline-row {
  margin-bottom: 0;
}

.inline-control-group {
  display: flex;
  align-items: center;
  gap: 0.85rem;
  flex-wrap: wrap;
  min-width: 0;
  justify-content: flex-end;
}

.compact-choice {
  display: inline-flex;
  align-items: center;
  gap: 0.42rem;
  cursor: pointer;
  font-size: 0.84rem;
  line-height: 1;
  color: #334155;
  user-select: none;
}

.compact-choice.disabled {
  opacity: 0.5;
  cursor: not-allowed;
}

.compact-input {
  margin: 0;
  width: 0.95rem;
  height: 0.95rem;
  accent-color: #2563eb;
  flex-shrink: 0;
}

.format-inline-group,
.field-inline-group {
  gap: 1rem;
}

.disktype-grid {
  display: flex;
  flex-wrap: nowrap;
  gap: 0.4rem;
  margin-top: 0.6rem;
}

.disktype-chip {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  min-height: 1.85rem;
  padding: 0 0.6rem;
  border: 1px solid #dbe3ef;
  border-radius: 0.65rem;
  background: #fff;
  font-size: 0.76rem;
  font-weight: 500;
  color: #334155;
  cursor: pointer;
  transition: all 0.2s ease;
  position: relative;
}

.disktype-chip:hover {
  border-color: #93c5fd;
  background: #f8fbff;
}

.disktype-chip .selected {
  color: #1d4ed8;
  font-weight: 600;
}

.disktype-chip.selected {
  border-color: #60a5fa;
  background: #eff6ff;
}

.field-checkbox {
  position: absolute;
  opacity: 0;
  pointer-events: none;
}

.export-modal-footer {
  display: flex;
  justify-content: flex-end;
  gap: 0.75rem;
  padding: 0.85rem 1rem 1rem;
  border-top: 1px solid #e2e8f0;
  background: rgba(248, 250, 252, 0.92);
}

.footer-btn {
  appearance: none;
  border: 1px solid transparent;
  border-radius: 0.8rem;
  padding: 0.75rem 1rem;
  min-width: 6.5rem;
  font-size: 0.85rem;
  font-weight: 700;
  cursor: pointer;
  transition: all 0.2s ease;
}

.footer-btn:disabled {
  opacity: 0.55;
  cursor: not-allowed;
}

.footer-btn-secondary {
  border-color: #dbe3ef;
  background: #fff;
  color: #334155;
}

.footer-btn-secondary:hover {
  background: #f8fafc;
}

.footer-btn-primary {
  background: linear-gradient(135deg, #2563eb, #1d4ed8);
  color: #fff;
  box-shadow: 0 12px 30px rgba(37, 99, 235, 0.18);
}

.footer-btn-primary:hover:not(:disabled) {
  transform: translateY(-1px);
  box-shadow: 0 16px 34px rgba(37, 99, 235, 0.22);
}

.export-modal-fade-enter-active,
.export-modal-fade-leave-active {
  transition: opacity 0.24s ease;
}

.export-modal-fade-enter-from,
.export-modal-fade-leave-to {
  opacity: 0;
}

.export-modal-fade-enter-active .export-modal,
.export-modal-fade-leave-active .export-modal {
  transition: transform 0.24s ease, opacity 0.24s ease;
}

.export-modal-fade-enter-from .export-modal,
.export-modal-fade-leave-to .export-modal {
  transform: translateY(12px) scale(0.98);
  opacity: 0;
}

@media (max-width: 768px) {
  .export-modal-overlay {
    padding: 0.75rem;
    align-items: center;
  }

  .export-modal {
    width: 100%;
    max-height: 92vh;
    border-radius: 1.1rem;
  }

  .export-modal-header {
    padding: 1rem 1rem 0.9rem;
  }

  .export-modal-body {
    padding: 1rem;
  }

  .section-inline-row {
    flex-direction: row;
    align-items: center;
    justify-content: space-between;
    gap: 0.5rem;
    flex-wrap: nowrap;
  }

  .inline-control-group,
  .format-inline-group,
  .field-inline-group,
  .option-inline-group {
    width: auto;
    justify-content: flex-end;
    gap: 0.65rem;
    flex-wrap: nowrap;
    min-width: 0;
  }

  .disktype-grid {
    gap: 0.35rem;
    margin-top: 0.55rem;
    flex-wrap: wrap;
  }

  .export-modal-footer {
    padding: 0.9rem 1rem 1rem;
    flex-direction: column-reverse;
  }

  .footer-btn {
    width: 100%;
  }

  .section-title {
    font-size: 0.88rem;
    flex-shrink: 0;
  }

  .compact-choice {
    font-size: 0.8rem;
    gap: 0.32rem;
  }

  .compact-input {
    width: 0.9rem;
    height: 0.9rem;
  }

  .disktype-chip {
    min-height: 1.8rem;
    padding: 0 0.58rem;
    font-size: 0.75rem;
  }
}
</style>
