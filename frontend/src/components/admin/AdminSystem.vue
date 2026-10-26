<template>
  <div class="page-stack">
    <div class="toolbar">
      <div class="notice notice-info" style="flex: 1">
        <AdminIcon name="system" :size="16" />
        <div>页面每 30 秒自动刷新一次；数据来自后端 <code>/api/health</code>。</div>
      </div>
      <button class="btn btn-sm" type="button" :disabled="loading" @click="load">
        <AdminIcon name="refresh" :size="14" /> {{ loading ? '刷新中…' : '立即刷新' }}
      </button>
    </div>

    <div v-if="error" class="notice notice-danger">
      <AdminIcon name="alert" :size="16" />
      <div>{{ error }}</div>
    </div>

    <template v-if="health">
      <div class="stat-grid">
        <div class="stat-card">
          <div>
            <div class="stat-label">后端状态</div>
            <div class="stat-value" style="font-size: 1.1rem">
              <span class="badge" :class="health.status === 'ok' ? 'badge-success' : 'badge-danger'">{{ health.status }}</span>
            </div>
            <div class="stat-meta">最近更新 {{ lastUpdated }}</div>
          </div>
          <div class="stat-icon emerald"><AdminIcon name="server" :size="19" /></div>
        </div>
        <div class="stat-card">
          <div>
            <div class="stat-label">启用插件</div>
            <div class="stat-value">{{ health.plugin_count || health.plugins?.length || 0 }}</div>
            <div class="stat-meta">{{ health.plugins_enabled ? '异步插件已启用' : '异步插件已关闭' }}</div>
          </div>
          <div class="stat-icon violet"><AdminIcon name="database" :size="19" /></div>
        </div>
        <div class="stat-card">
          <div>
            <div class="stat-label">TG 频道</div>
            <div class="stat-value">{{ health.channels_count || health.channels?.length || 0 }}</div>
            <div class="stat-meta">默认搜索频道数量</div>
          </div>
          <div class="stat-icon"><AdminIcon name="users" :size="19" /></div>
        </div>
        <div class="stat-card">
          <div>
            <div class="stat-label">访问策略</div>
            <div class="stat-value" style="font-size: 1rem">
              <span class="badge" :class="health.api_require_login ? 'badge-primary' : ''">{{ health.api_require_login ? 'API 需登录' : 'API 开放' }}</span>
            </div>
            <div class="stat-meta">网页搜索：{{ health.web_search_enabled ? '开启' : '关闭' }}</div>
          </div>
          <div class="stat-icon amber"><AdminIcon name="shield" :size="19" /></div>
        </div>
      </div>

      <section class="section-card">
        <div class="section-head">
          <div>
            <div class="section-title">TG 可达性</div>
            <div class="section-desc">不可达时后端会跳过 TG 阶段，避免大量超时</div>
          </div>
          <span class="badge" :class="health.tg?.reachable ? 'badge-success' : 'badge-danger'">
            {{ health.tg?.reachable ? '可达' : '不可达 / 未知' }}
          </span>
        </div>
        <div class="section-body">
          <div class="detail-list">
            <div class="detail-row"><span>结论</span><strong>{{ health.tg?.reachable ? '可达' : (health.tg?.reason || '暂无探测结果') }}</strong></div>
            <div class="detail-row"><span>探测时间</span><strong>{{ health.tg?.checked_at ? shortTime(String(health.tg.checked_at)) : '—' }}</strong></div>
          </div>
        </div>
      </section>

      <section class="section-card" v-if="health.liveness">
        <div class="section-head">
          <div>
            <div class="section-title">插件 / 频道存活观测</div>
            <div class="section-desc">{{ health.liveness.note }}</div>
          </div>
        </div>
        <div class="section-body">
          <div class="live-summary">
            <div class="live-chip">
              <span>插件总数</span><strong>{{ health.liveness.plugin_total }}</strong>
            </div>
            <div class="live-chip">
              <span>频道总数</span><strong>{{ health.liveness.channel_total }}</strong>
            </div>
            <div class="live-chip danger">
              <span>异常插件</span><strong>{{ health.liveness.failing_plugin_count || 0 }}</strong>
            </div>
            <div class="live-chip warning">
              <span>零产出插件</span><strong>{{ health.liveness.zero_yield_plugin_count || 0 }}</strong>
            </div>
            <div class="live-chip danger">
              <span>异常频道</span><strong>{{ health.liveness.failing_channel_count || 0 }}</strong>
            </div>
          </div>

          <div class="live-grid">
            <div class="live-block">
              <h4>异常插件</h4>
              <div v-if="health.liveness.failing_plugins?.length" class="live-list">
                <div v-for="item in health.liveness.failing_plugins" :key="item.name" class="live-item">
                  <span class="badge badge-danger">{{ item.name }}</span>
                  <span class="live-error">{{ item.last_error || `失败 ${item.failed} 次` }}</span>
                </div>
              </div>
              <p v-else class="empty-line">无异常插件</p>
            </div>
            <div class="live-block">
              <h4>零产出插件</h4>
              <div v-if="health.liveness.zero_yield_plugins?.length" class="live-list">
                <div v-for="item in health.liveness.zero_yield_plugins" :key="item.name" class="live-item">
                  <span class="badge badge-warning">{{ item.name }}</span>
                  <span class="live-error">窗口 {{ item.rounds }} 轮，产出 0</span>
                </div>
              </div>
              <p v-else class="empty-line">无零产出插件</p>
            </div>
            <div class="live-block">
              <h4>异常频道</h4>
              <div v-if="health.liveness.failing_channels?.length" class="live-list">
                <div v-for="item in health.liveness.failing_channels" :key="item.name" class="live-item">
                  <span class="badge badge-danger">{{ item.name }}</span>
                  <span class="live-error">{{ item.last_error || `失败 ${item.failed} 次` }}</span>
                </div>
              </div>
              <p v-else class="empty-line">无异常频道</p>
            </div>
            <div class="live-block">
              <h4>零产出频道</h4>
              <div v-if="health.liveness.zero_yield_channels?.length" class="live-list">
                <div v-for="item in health.liveness.zero_yield_channels" :key="item.name" class="live-item">
                  <span class="badge badge-warning">{{ item.name }}</span>
                  <span class="live-error">窗口 {{ item.rounds }} 轮，产出 0</span>
                </div>
              </div>
              <p v-else class="empty-line">无零产出频道</p>
            </div>
          </div>
        </div>
      </section>

      <details class="section-card raw-card">
        <summary>原始健康检查数据</summary>
        <pre>{{ JSON.stringify(health, null, 2) }}</pre>
      </details>
    </template>
  </div>
</template>

<script setup lang="ts">
import { onMounted, onUnmounted, ref } from 'vue';
import { getHealth, type HealthStatus } from '@/api';
import AdminIcon from './AdminIcon.vue';
import { toast } from '@/lib/toast';

const health = ref<HealthStatus | null>(null);
const loading = ref(false);
const error = ref('');
const lastUpdated = ref('');
let timer: number | null = null;

function shortTime(value: string) {
  return value ? new Date(value).toLocaleString() : '';
}

const load = async () => {
  loading.value = true;
  error.value = '';
  try {
    health.value = await getHealth();
    lastUpdated.value = new Date().toLocaleTimeString();
  } catch (err: any) {
    error.value = err.response?.data?.error || err.message || '无法连接后端';
    toast.error('读取系统状态失败');
  } finally {
    loading.value = false;
  }
};

onMounted(() => {
  load();
  timer = window.setInterval(load, 30_000);
});

onUnmounted(() => {
  if (timer) window.clearInterval(timer);
});
</script>

<style scoped>
.live-summary {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(120px, 1fr));
  gap: 0.5rem;
  margin-bottom: 0.9rem;
}
.live-chip {
  border: 1px solid hsl(var(--border));
  border-radius: 0.6rem;
  padding: 0.55rem 0.65rem;
  background: hsl(var(--muted) / 0.3);
}
.live-chip span { display: block; font-size: 0.68rem; color: hsl(var(--muted-foreground)); }
.live-chip strong { display: block; font-size: 1.15rem; margin-top: 0.1rem; }
.live-chip.danger strong { color: #b91c1c; }
.live-chip.warning strong { color: #b45309; }

.live-grid {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(260px, 1fr));
  gap: 0.75rem;
}
.live-block {
  border: 1px solid hsl(var(--border));
  border-radius: 0.65rem;
  padding: 0.7rem;
}
.live-block h4 { font-size: 0.76rem; font-weight: 650; margin-bottom: 0.5rem; }
.live-list { display: flex; flex-direction: column; gap: 0.35rem; max-height: 11rem; overflow: auto; }
.live-item { display: flex; align-items: center; gap: 0.45rem; font-size: 0.72rem; }
.live-error { color: hsl(var(--muted-foreground)); overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
.empty-line { font-size: 0.74rem; color: hsl(var(--muted-foreground)); }

.raw-card summary {
  padding: 0.85rem 1.1rem;
  cursor: pointer;
  font-size: 0.8rem;
  font-weight: 600;
}
.raw-card pre {
  margin: 0;
  padding: 0 1.1rem 1rem;
  max-height: 24rem;
  overflow: auto;
  font-size: 0.68rem;
  line-height: 1.55;
  color: hsl(var(--muted-foreground));
}
</style>
