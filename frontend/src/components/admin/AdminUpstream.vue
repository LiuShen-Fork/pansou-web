<template>
  <div class="admin-content">
    <div v-if="error" class="admin-alert error">{{ error }}</div>
    <div v-if="message" class="admin-alert ok">{{ message }}</div>

    <div class="admin-card">
      <h3>上游自动同步</h3>
      <p class="admin-muted">
        定时抓取上游 <code class="admin-code">{{ form.repo }}</code> 的 Dockerfile，
        刷新 TG 频道与启用插件列表。频道会立即生效；插件列表写入数据目录，重启后生效
        （新插件还需要先更新镜像）。
      </p>

      <div class="admin-switch">
        <div><strong>启用自动同步</strong><small>关闭后只能手动点击「立即同步」。</small></div>
        <input v-model="form.enabled" type="checkbox" />
      </div>
      <div class="admin-grid">
        <div class="admin-field">
          <label>同步间隔（小时）</label>
          <input v-model.number="form.interval_hours" type="number" min="1" max="720" />
        </div>
        <div class="admin-field">
          <label>上游仓库</label>
          <input v-model="form.repo" placeholder="fish2018/pansou" />
        </div>
        <div class="admin-field">
          <label>分支 / Tag</label>
          <input v-model="form.ref" placeholder="main" />
        </div>
      </div>
      <div class="admin-actions">
        <button class="admin-btn primary" type="button" :disabled="saving" @click="save">
          {{ saving ? '保存中...' : '保存同步设置' }}
        </button>
        <button class="admin-btn" type="button" :disabled="running" @click="runNow">
          {{ running ? '同步中...' : '立即同步' }}
        </button>
        <span v-if="running" class="admin-muted">正在请求 GitHub，请稍候…</span>
      </div>
    </div>

    <div class="admin-card">
      <div class="admin-actions" style="justify-content: space-between; margin-bottom: 0.7rem">
        <h3 style="margin: 0">最近一次同步</h3>
        <button class="admin-btn" type="button" @click="load">刷新状态</button>
      </div>
      <div class="admin-stats">
        <div class="admin-stat"><span>频道数</span><strong>{{ info?.status.channel_count || info?.channels.length || 0 }}</strong></div>
        <div class="admin-stat"><span>插件数</span><strong>{{ info?.status.plugin_count || info?.enabled_plugins.length || 0 }}</strong></div>
        <div class="admin-stat"><span>频道变化</span><strong>{{ info?.status.channels_changed ? '是' : '否' }}</strong></div>
        <div class="admin-stat"><span>插件变化</span><strong>{{ info?.status.plugins_changed ? '是' : '否' }}</strong></div>
      </div>
      <div class="upstream-lines">
        <div><span>上次检查</span><strong>{{ info?.status.last_check_at ? shortTime(info.status.last_check_at) : '尚未执行' }}</strong></div>
        <div><span>上次成功</span><strong>{{ info?.status.last_success_at ? shortTime(info.status.last_success_at) : '—' }}</strong></div>
        <div><span>上游版本</span><strong><code>{{ info?.status.last_revision || '—' }}</code></strong></div>
        <div><span>下次检查</span><strong>{{ info?.next_check_at ? shortTime(info.next_check_at) : '—' }}</strong></div>
        <div><span>持久化文件</span><strong><code>{{ info?.defaults_env_path || '—' }}</code></strong></div>
      </div>
      <div v-if="info?.status.last_error" class="admin-alert error" style="margin-top: 0.8rem; margin-bottom: 0">
        上次同步失败：{{ info.status.last_error }}
      </div>
      <div v-if="info?.status.restart_required" class="admin-alert warn" style="margin-top: 0.8rem; margin-bottom: 0">
        插件列表已变化，需要重启容器后生效。频道列表已经热更新，无需重启。
      </div>

      <details v-if="info?.channels.length" class="upstream-details">
        <summary>当前频道（{{ info.channels.length }}）</summary>
        <p class="admin-code">{{ info.channels.join(', ') }}</p>
      </details>
      <details v-if="info?.enabled_plugins.length" class="upstream-details">
        <summary>当前启用插件（{{ info.enabled_plugins.length }}）</summary>
        <p class="admin-code">{{ info.enabled_plugins.join(', ') }}</p>
      </details>
    </div>
  </div>
</template>

<script setup lang="ts">
import { onMounted, reactive, ref } from 'vue';
import {
  adminGetUpstreamSync,
  adminRunUpstreamSync,
  adminSaveUpstreamSync,
  type UpstreamSyncInfo,
} from '@/api/auth';

const info = ref<UpstreamSyncInfo | null>(null);
const form = reactive({ enabled: true, interval_hours: 24, repo: 'fish2018/pansou', ref: 'main' });
const saving = ref(false);
const running = ref(false);
const error = ref('');
const message = ref('');

const shortTime = (s: string) => (s ? new Date(s).toLocaleString() : '');

const apply = (data: UpstreamSyncInfo) => {
  info.value = data;
  Object.assign(form, data.settings);
  running.value = data.running;
};

const load = async () => {
  error.value = '';
  try {
    apply(await adminGetUpstreamSync());
  } catch (err: any) {
    error.value = err.response?.data?.error || '读取同步状态失败';
  }
};

const save = async () => {
  error.value = '';
  message.value = '';
  saving.value = true;
  try {
    await adminSaveUpstreamSync({ ...form });
    message.value = '同步设置已保存';
    await load();
  } catch (err: any) {
    error.value = err.response?.data?.error || '保存失败';
  } finally {
    saving.value = false;
  }
};

const runNow = async () => {
  error.value = '';
  message.value = '';
  running.value = true;
  try {
    const status = await adminRunUpstreamSync();
    message.value = `同步完成：频道 ${status.channel_count} 个，插件 ${status.plugin_count} 个`;
    await load();
  } catch (err: any) {
    error.value = err.response?.data?.error || '同步失败';
  } finally {
    running.value = false;
    await load();
  }
};

onMounted(load);
</script>

<style scoped>
.upstream-lines {
  margin-top: 0.8rem;
  display: grid;
  gap: 0.35rem;
}
.upstream-lines > div {
  display: flex;
  justify-content: space-between;
  gap: 1rem;
  font-size: 0.8rem;
  border-bottom: 1px dashed hsl(var(--border));
  padding-bottom: 0.35rem;
}
.upstream-lines span { color: hsl(var(--muted-foreground)); }
.upstream-lines strong { font-weight: 550; text-align: right; word-break: break-all; }
.upstream-details {
  margin-top: 0.8rem;
  font-size: 0.8rem;
}
.upstream-details summary {
  cursor: pointer;
  font-weight: 550;
  margin-bottom: 0.4rem;
}
.upstream-details p {
  max-height: 8rem;
  overflow: auto;
  line-height: 1.6;
}
</style>
