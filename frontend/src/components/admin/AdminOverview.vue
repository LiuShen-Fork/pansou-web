<template>
  <div class="page-stack">
    <div v-if="loading && !overview" class="stat-grid">
      <div v-for="i in 4" :key="i" class="stat-card">
        <div style="flex: 1">
          <div class="skeleton" style="width: 40%" />
          <div class="skeleton" style="width: 60%; height: 1.6rem; margin-top: 0.5rem" />
        </div>
      </div>
    </div>

    <template v-else>
      <div class="stat-grid">
        <div class="stat-card">
          <div>
            <div class="stat-label">用户总数</div>
            <div class="stat-value">{{ overview?.users_total ?? 0 }}</div>
            <div class="stat-meta">活跃 {{ overview?.users_active ?? 0 }} · 管理员 {{ overview?.admins_total ?? 0 }}</div>
          </div>
          <div class="stat-icon"><AdminIcon name="users" :size="19" /></div>
        </div>
        <div class="stat-card">
          <div>
            <div class="stat-label">今日搜索</div>
            <div class="stat-value">{{ overview?.searches_today ?? 0 }}</div>
            <div class="stat-meta">累计 {{ overview?.searches_total ?? 0 }} 次</div>
          </div>
          <div class="stat-icon emerald"><AdminIcon name="search" :size="19" /></div>
        </div>
        <div class="stat-card">
          <div>
            <div class="stat-label">匿名搜索（今日）</div>
            <div class="stat-value">{{ overview?.anonymous_today ?? 0 }}</div>
            <div class="stat-meta">来自网页直接搜索</div>
          </div>
          <div class="stat-icon amber"><AdminIcon name="user" :size="19" /></div>
        </div>
        <div class="stat-card">
          <div>
            <div class="stat-label">有效会话</div>
            <div class="stat-value">{{ overview?.sessions_active ?? 0 }}</div>
            <div class="stat-meta">当前登录中的设备</div>
          </div>
          <div class="stat-icon violet"><AdminIcon name="shield" :size="19" /></div>
        </div>
      </div>

      <div v-if="overview?.default_admin_alive" class="notice notice-warning">
        <AdminIcon name="alert" :size="16" />
        <div>默认管理员仍在使用初始密码，请尽快前往用户中心修改。</div>
      </div>

      <div class="overview-grid">
        <section class="section-card chart-card">
          <div class="section-head no-border">
            <div>
              <div class="section-title">近 {{ stats?.days || 14 }} 天搜索趋势</div>
              <div class="section-desc">按天统计的搜索请求量，悬停查看明细</div>
            </div>
            <span class="badge badge-primary">搜索</span>
          </div>
          <div v-if="stats && stats.daily.some((d) => d.searches > 0)" class="chart-bars">
            <div v-for="d in stats.daily" :key="d.day" class="chart-bar-col">
              <div class="chart-bar-track">
                <div class="chart-tip">{{ d.day }} · {{ d.searches }} 次 · 新增 {{ d.new_users }} 人</div>
                <div class="chart-bar" :style="{ height: barHeight(d.searches) }" />
              </div>
              <span class="chart-label">{{ d.day.slice(5) }}</span>
            </div>
          </div>
          <div v-else class="empty-state">
            <AdminIcon name="chart" :size="22" />
            <strong>暂无搜索数据</strong>
            <span>有用户搜索后会在这里显示趋势</span>
          </div>
        </section>

        <div class="overview-side">
          <section class="section-card">
            <div class="section-head">
              <div class="section-title">上游同步</div>
              <button class="btn btn-sm" type="button" @click="$emit('navigate', 'upstream')">详情</button>
            </div>
            <div class="section-body">
              <div v-if="upstream" class="detail-list">
                <div class="detail-row"><span>状态</span><strong>
                  <span class="badge" :class="upstream.status.last_error ? 'badge-danger' : 'badge-success'">
                    {{ upstream.status.last_error ? '上次失败' : '正常' }}
                  </span>
                </strong></div>
                <div class="detail-row"><span>频道 / 插件</span><strong>{{ upstream.status.channel_count || upstream.channels.length }} / {{ upstream.status.plugin_count || upstream.enabled_plugins.length }}</strong></div>
                <div class="detail-row"><span>上次成功</span><strong>{{ upstream.status.last_success_at ? shortTime(upstream.status.last_success_at) : '尚未执行' }}</strong></div>
                <div class="detail-row"><span>重启标记</span><strong>
                  <span class="badge" :class="upstream.status.restart_required ? 'badge-warning' : 'badge-success'">
                    {{ upstream.status.restart_required ? '需重启' : '无需' }}
                  </span>
                </strong></div>
              </div>
              <div v-else class="empty-state" style="padding: 1.2rem 0">
                <span>同步状态暂不可用</span>
              </div>
            </div>
          </section>

          <section class="section-card">
            <div class="section-head">
              <div class="section-title">快捷操作</div>
            </div>
            <div class="section-body quick-actions">
              <button type="button" @click="$emit('navigate', 'users')"><AdminIcon name="users" :size="16" /> 新建用户</button>
              <button type="button" @click="$emit('navigate', 'settings')"><AdminIcon name="settings" :size="16" /> 登录策略</button>
              <button type="button" @click="$emit('navigate', 'oauth')"><AdminIcon name="key" :size="16" /> OAuth 配置</button>
              <button type="button" @click="$emit('navigate', 'system')"><AdminIcon name="system" :size="16" /> 系统状态</button>
            </div>
          </section>
        </div>
      </div>

      <div class="overview-grid lower">
        <section class="section-card">
          <div class="section-head">
            <div>
              <div class="section-title">搜索排行</div>
              <div class="section-desc">近 {{ stats?.days || 14 }} 天用户搜索次数</div>
            </div>
            <button class="btn btn-sm" type="button" @click="$emit('navigate', 'logs')">全部日志</button>
          </div>
          <div class="section-body flush">
            <div v-if="stats?.top_users?.length" class="ranking">
              <div v-for="(u, i) in stats.top_users" :key="u.username" class="ranking-row">
                <span class="rank" :class="{ top: i < 3 }">{{ i + 1 }}</span>
                <span class="rank-name">{{ u.username }}</span>
                <span class="rank-bar"><i :style="{ width: rankWidth(u.searches) }" /></span>
                <span class="rank-count">{{ u.searches }} 次</span>
              </div>
            </div>
            <div v-else class="empty-state">
              <strong>暂无排行</strong>
              <span>用户搜索后会在这里汇总</span>
            </div>
          </div>
        </section>

        <section class="section-card">
          <div class="section-head">
            <div>
              <div class="section-title">最近搜索</div>
              <div class="section-desc">最新 10 条搜索记录</div>
            </div>
          </div>
          <div v-if="overview?.recent_searches?.length" class="table-wrap">
            <table class="data-table">
              <thead><tr><th>时间</th><th>用户</th><th>来源</th><th>关键词</th></tr></thead>
              <tbody>
                <tr v-for="log in overview.recent_searches" :key="log.id">
                  <td class="hide-sm">{{ shortTime(log.created_at) }}</td>
                  <td>{{ log.username || '匿名' }}</td>
                  <td><span class="badge" :class="log.source === 'web' ? 'badge-warning' : 'badge-primary'">{{ log.source }}</span></td>
                  <td class="cell-strong">{{ log.keyword || '—' }}</td>
                </tr>
              </tbody>
            </table>
          </div>
          <div v-else class="empty-state">
            <strong>暂无搜索记录</strong>
            <span>用户开始搜索后会显示在这里</span>
          </div>
        </section>
      </div>
    </template>
  </div>
</template>

<script setup lang="ts">
import { computed, onMounted, ref } from 'vue';
import { adminGetOverview, adminGetStats, adminGetUpstreamSync, type Overview, type StatsResponse, type UpstreamSyncInfo } from '@/api/auth';
import { getHealth } from '@/api';
import AdminIcon from './AdminIcon.vue';
import { toast } from '@/lib/toast';

defineEmits<{ navigate: [key: string] }>();

const overview = ref<Overview | null>(null);
const stats = ref<StatsResponse | null>(null);
const upstream = ref<UpstreamSyncInfo | null>(null);
const loading = ref(true);

const maxSearches = computed(() => Math.max(1, ...(stats.value?.daily || []).map((d) => d.searches)));
const maxRank = computed(() => Math.max(1, ...(stats.value?.top_users || []).map((u) => u.searches)));

function barHeight(value: number) {
  return `${Math.max(2, Math.round((value / maxSearches.value) * 100))}%`;
}

function rankWidth(value: number) {
  return `${Math.max(4, Math.round((value / maxRank.value) * 100))}%`;
}

function shortTime(value: string) {
  return value ? new Date(value).toLocaleString() : '';
}

const load = async () => {
  loading.value = true;
  const [o, s, u] = await Promise.allSettled([adminGetOverview(), adminGetStats(14), adminGetUpstreamSync()]);
  if (o.status === 'fulfilled') overview.value = o.value;
  else toast.error('概览加载失败');
  if (s.status === 'fulfilled') stats.value = s.value;
  if (u.status === 'fulfilled') upstream.value = u.value;
  loading.value = false;
  // 健康检查用于提前发现后端异常，但不阻塞页面
  getHealth().catch(() => undefined);
};

onMounted(load);
</script>

<style scoped>
.overview-grid {
  display: grid;
  grid-template-columns: minmax(0, 1.65fr) minmax(17rem, 1fr);
  gap: 1rem;
  align-items: stretch;
}
.overview-grid.lower { grid-template-columns: minmax(17rem, 1fr) minmax(0, 1.65fr); }
.overview-side { display: flex; flex-direction: column; gap: 1rem; height: 100%; }
.overview-side .section-card:last-child { flex: 1; display: flex; flex-direction: column; }
.overview-side .section-card:last-child .section-body { flex: 1; }
.chart-card { display: flex; flex-direction: column; height: 100%; }
.chart-card .chart-bars { flex: 1; min-height: 12rem; height: auto; }
.chart-card .empty-state { flex: 1; justify-content: center; }

.quick-actions {
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: 0.5rem;
}
.quick-actions button {
  display: flex;
  align-items: center;
  gap: 0.45rem;
  padding: 0.6rem;
  border: 1px solid hsl(var(--border));
  border-radius: 0.6rem;
  background: hsl(var(--muted) / 0.3);
  color: hsl(var(--foreground));
  font-size: 0.75rem;
  font-weight: 550;
  cursor: pointer;
  text-align: left;
}
.quick-actions button:hover { border-color: hsl(var(--primary) / 0.5); color: hsl(var(--primary)); background: hsl(var(--primary) / 0.06); }

.ranking { padding: 0.35rem 0.8rem 0.8rem; display: flex; flex-direction: column; gap: 0.55rem; }
.ranking-row { display: flex; align-items: center; gap: 0.5rem; font-size: 0.76rem; }
.rank { width: 1.25rem; height: 1.25rem; display: grid; place-items: center; border-radius: 0.35rem; background: hsl(var(--muted)); color: hsl(var(--muted-foreground)); font-size: 0.64rem; font-weight: 700; }
.rank.top { background: #eef2ff; color: #4338ca; }
.rank-name { width: 6.5rem; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; font-weight: 550; }
.rank-bar { flex: 1; height: 0.42rem; border-radius: 999px; background: hsl(var(--muted)); overflow: hidden; }
.rank-bar i { display: block; height: 100%; border-radius: 999px; background: linear-gradient(90deg, hsl(var(--primary)), #818cf8); }
.rank-count { width: 3.2rem; text-align: right; color: hsl(var(--muted-foreground)); font-variant-numeric: tabular-nums; }

@media (max-width: 1100px) {
  .overview-grid, .overview-grid.lower { grid-template-columns: 1fr; }
}
</style>
