<template>
  <div class="page-stack">
    <section class="section-card">
      <div class="section-head">
        <div>
          <div class="section-title"><AdminIcon name="upstream" :size="16" /> 同步设置</div>
          <div class="section-desc">定时抓取上游 Dockerfile 的 CHANNELS 与 ENABLED_PLUGINS，默认每 24 小时一次</div>
        </div>
        <button class="btn btn-primary btn-sm" type="button" :disabled="running" @click="runNow">
          <AdminIcon name="refresh" :size="14" /> {{ running ? '同步中…' : '立即同步' }}
        </button>
      </div>
      <div class="section-body">
        <div class="switch-row">
          <div><strong>启用自动同步</strong><small>关闭后仍可手动点击「立即同步」；首次启动会检查一次。</small></div>
          <label class="switch"><input v-model="form.enabled" type="checkbox" /><span /></label>
        </div>
        <div class="form-grid" style="margin-top: 0.8rem">
          <div class="field">
            <label>同步间隔（小时）</label>
            <input v-model.number="form.interval_hours" class="input" type="number" min="1" max="720" />
            <span class="hint">失败后最多每小时重试一次</span>
          </div>
          <div class="field">
            <label>上游仓库</label>
            <input v-model="form.repo" class="input" placeholder="fish2018/pansou" />
          </div>
          <div class="field">
            <label>分支 / Tag</label>
            <input v-model="form.ref" class="input" placeholder="main" />
          </div>
        </div>
        <div class="toolbar" style="margin-top: 0.9rem">
          <button class="btn" type="button" :disabled="saving" @click="save">
            <AdminIcon name="check" :size="14" /> {{ saving ? '保存中…' : '保存设置' }}
          </button>
          <span class="section-desc">频道会热更新；插件列表写入数据目录，重启后生效。</span>
        </div>
      </div>
    </section>

    <div class="stat-grid">
      <div class="stat-card">
        <div>
          <div class="stat-label">上游版本</div>
          <div class="stat-value" style="font-size: 0.95rem">{{ shortRev(info?.status.last_revision) }}</div>
          <div class="stat-meta">来源 {{ form.repo }}@{{ form.ref }}</div>
        </div>
        <div class="stat-icon violet"><AdminIcon name="upstream" :size="19" /></div>
      </div>
      <div class="stat-card">
        <div>
          <div class="stat-label">TG 频道</div>
          <div class="stat-value">{{ info?.status.channel_count || info?.channels.length || 0 }}</div>
          <div class="stat-meta">{{ info?.status.channels_changed ? '上次检查有变化' : '与当前一致' }}</div>
        </div>
        <div class="stat-icon"><AdminIcon name="users" :size="19" /></div>
      </div>
      <div class="stat-card">
        <div>
          <div class="stat-label">启用插件</div>
          <div class="stat-value">{{ info?.status.plugin_count || info?.enabled_plugins.length || 0 }}</div>
          <div class="stat-meta">{{ info?.status.plugins_changed ? '插件列表有变化' : '与当前一致' }}</div>
        </div>
        <div class="stat-icon emerald"><AdminIcon name="database" :size="19" /></div>
      </div>
      <div class="stat-card">
        <div>
          <div class="stat-label">上次检查</div>
          <div class="stat-value" style="font-size: 0.95rem">{{ info?.status.last_check_at ? shortTime(info.status.last_check_at) : '尚未执行' }}</div>
          <div class="stat-meta">下次 {{ info?.next_check_at ? shortTime(info.next_check_at) : '—' }}</div>
        </div>
        <div class="stat-icon amber"><AdminIcon name="clock" :size="19" /></div>
      </div>
    </div>

    <div v-if="info?.status.restart_required" class="notice notice-warning">
      <AdminIcon name="alert" :size="16" />
      <div>插件列表已变化，需要重启容器后生效；TG 频道已经热更新，无需重启。</div>
    </div>
    <div v-if="info?.status.last_error" class="notice notice-danger">
      <AdminIcon name="alert" :size="16" />
      <div>上次同步失败：{{ info.status.last_error }}</div>
    </div>
    <div v-else-if="info?.status.last_success_at" class="notice notice-success">
      <AdminIcon name="check" :size="16" />
      <div>上次成功：{{ shortTime(info.status.last_success_at) }} · 持久化文件 {{ info.defaults_env_path }}</div>
    </div>

    <div class="upstream-grid">
      <section class="section-card">
        <div class="section-head">
          <div>
            <div class="section-title">当前频道（{{ info?.channels.length || 0 }}）</div>
            <div class="section-desc">运行中实际生效的 TG 频道列表</div>
          </div>
        </div>
        <div class="section-body">
          <div v-if="info?.channels.length" class="token-cloud">
            <span v-for="ch in visibleChannels" :key="ch" class="badge">{{ ch }}</span>
            <span v-if="info.channels.length > visibleChannels.length" class="badge badge-primary">+{{ info.channels.length - visibleChannels.length }}</span>
          </div>
          <div v-else class="empty-state" style="padding: 1rem 0">
            <span>暂无频道数据</span>
          </div>
        </div>
      </section>

      <section class="section-card">
        <div class="section-head">
          <div>
            <div class="section-title">当前启用插件（{{ info?.enabled_plugins.length || 0 }}）</div>
            <div class="section-desc">启动时注册的插件列表</div>
          </div>
        </div>
        <div class="section-body">
          <div v-if="info?.enabled_plugins.length" class="token-cloud">
            <span v-for="p in visiblePlugins" :key="p" class="badge badge-violet">{{ p }}</span>
            <span v-if="info.enabled_plugins.length > visiblePlugins.length" class="badge badge-primary">+{{ info.enabled_plugins.length - visiblePlugins.length }}</span>
          </div>
          <div v-else class="empty-state" style="padding: 1rem 0">
            <span>暂无插件数据</span>
          </div>
        </div>
      </section>
    </div>

    <details class="section-card raw-card" v-if="info">
      <summary>同步详情</summary>
      <div class="section-body">
        <div class="detail-list">
          <div class="detail-row"><span>默认值文件</span><strong class="mono">{{ info.defaults_env_path }}</strong></div>
          <div class="detail-row"><span>最近检查</span><strong>{{ info.status.last_check_at ? shortTime(info.status.last_check_at) : '—' }}</strong></div>
          <div class="detail-row"><span>最近成功</span><strong>{{ info.status.last_success_at ? shortTime(info.status.last_success_at) : '—' }}</strong></div>
          <div class="detail-row"><span>频道变化</span><strong>{{ info.status.channels_changed ? '是' : '否' }}</strong></div>
          <div class="detail-row"><span>插件变化</span><strong>{{ info.status.plugins_changed ? '是' : '否' }}</strong></div>
          <div class="detail-row"><span>运行中任务</span><strong>{{ info.running ? '是' : '否' }}</strong></div>
        </div>
      </div>
    </details>
  </div>
</template>

<script setup lang="ts">
import { computed, onMounted, reactive, ref } from 'vue';
import {
  adminGetUpstreamSync,
  adminRunUpstreamSync,
  adminSaveUpstreamSync,
  type UpstreamSyncInfo,
} from '@/api/auth';
import AdminIcon from './AdminIcon.vue';
import { toast } from '@/lib/toast';

const info = ref<UpstreamSyncInfo | null>(null);
const form = reactive({ enabled: true, interval_hours: 24, repo: 'fish2018/pansou', ref: 'main' });
const saving = ref(false);
const running = ref(false);

const visibleChannels = computed(() => (info.value?.channels || []).slice(0, 40));
const visiblePlugins = computed(() => (info.value?.enabled_plugins || []).slice(0, 40));

function shortTime(value: string) {
  return value ? new Date(value).toLocaleString() : '';
}
function shortRev(value?: string) {
  if (!value) return '—';
  return value.length > 12 ? value.slice(0, 12) : value;
}

const apply = (data: UpstreamSyncInfo) => {
  info.value = data;
  Object.assign(form, data.settings);
  running.value = data.running;
};

const load = async () => {
  try {
    apply(await adminGetUpstreamSync());
  } catch (err: any) {
    toast.error('读取同步状态失败', err.response?.data?.error || '');
  }
};

const save = async () => {
  saving.value = true;
  try {
    await adminSaveUpstreamSync({ ...form });
    toast.success('同步设置已保存');
    await load();
  } catch (err: any) {
    toast.error('保存失败', err.response?.data?.error || '');
  } finally {
    saving.value = false;
  }
};

const runNow = async () => {
  running.value = true;
  try {
    const status = await adminRunUpstreamSync();
    toast.success('同步完成', `频道 ${status.channel_count} 个，插件 ${status.plugin_count} 个`);
    await load();
  } catch (err: any) {
    toast.error('同步失败', err.response?.data?.error || '');
  } finally {
    running.value = false;
    await load();
  }
};

onMounted(load);
</script>

<style scoped>
.upstream-grid { display: grid; grid-template-columns: 1fr 1fr; gap: 1rem; align-items: start; }
.token-cloud { display: flex; flex-wrap: wrap; gap: 0.3rem; max-height: 12rem; overflow: auto; }
.raw-card summary { padding: 0.85rem 1.1rem; cursor: pointer; font-size: 0.82rem; font-weight: 600; }
@media (max-width: 900px) { .upstream-grid { grid-template-columns: 1fr; } }
</style>
