<template>
  <div class="page-stack">
    <div class="settings-bar">
      <div class="settings-bar-copy">
        <strong>运行参数</strong>
        <span>代理、缓存、并发与网页搜索签名保存在数据库中，优先于环境变量。标记“重启生效”的项需要重启容器。</span>
      </div>
      <button class="btn btn-primary" type="button" :disabled="saving || loading" @click="save">
        <AdminIcon name="check" :size="15" /> {{ saving ? '保存中…' : '保存运行参数' }}
      </button>
    </div>

    <div v-if="error" class="notice notice-danger">
      <AdminIcon name="alert" :size="16" />
      <div>{{ error }}</div>
    </div>

    <div v-if="loading" class="section-card section-body">
      <div class="skeleton" style="width: 28%; margin-bottom: 0.7rem" />
      <div class="skeleton" style="width: 100%; margin-bottom: 0.5rem" />
      <div class="skeleton" style="width: 82%" />
    </div>

    <template v-else>
      <div class="settings-grid">
        <section class="section-card">
          <div class="section-head">
            <div>
              <div class="section-title"><AdminIcon name="tune" :size="16" /> 出站代理</div>
              <div class="section-desc">用于访问 Telegram、插件站点与上游同步</div>
            </div>
            <span class="badge badge-warning">重启生效</span>
          </div>
          <div class="section-body">
            <div class="field">
              <label>代理地址</label>
              <input
                v-model.trim="form.proxy_url"
                class="input"
                placeholder="http://127.0.0.1:7890 或 socks5://127.0.0.1:7890"
                spellcheck="false"
              />
              <span class="hint">留空表示不使用代理。支持 http、https、socks5；修改后重启容器生效。</span>
            </div>
            <div class="effective-line">
              当前生效：<strong>{{ effective.proxy_url || '未使用代理' }}</strong>
            </div>
          </div>
        </section>

        <section class="section-card">
          <div class="section-head">
            <div>
              <div class="section-title"><AdminIcon name="database" :size="16" /> 搜索缓存</div>
              <div class="section-desc">控制搜索结果缓存与写入 TTL</div>
            </div>
          </div>
          <div class="section-body">
            <div class="switch-row">
              <div>
                <strong>启用缓存 <span class="badge badge-warning">重启生效</span></strong>
                <small>关闭后所有搜索都会直接请求上游；修改开关需要重启容器。</small>
              </div>
              <label class="switch"><input v-model="form.cache_enabled" type="checkbox" /><span /></label>
            </div>
            <div class="field" style="margin-top: 0.9rem">
              <label>缓存 TTL（分钟）<span class="badge badge-success">立即生效</span></label>
              <input v-model.number="form.cache_ttl_minutes" class="input" type="number" min="1" max="10080" />
              <span class="hint">只影响新写入的缓存；当前生效 {{ effective.cache_ttl_minutes }} 分钟。</span>
            </div>
          </div>
        </section>
      </div>

      <section class="section-card">
        <div class="section-head">
          <div>
            <div class="section-title"><AdminIcon name="system" :size="16" /> 搜索并发</div>
            <div class="section-desc">控制单次搜索的扇出并行度与整个进程的出口总闸</div>
          </div>
        </div>
        <div class="section-body">
          <div class="form-grid">
            <div class="field">
              <label>默认搜索并发 <span class="badge badge-success">立即生效</span></label>
              <input v-model.number="form.default_concurrency" class="input" type="number" min="0" max="100000" />
              <span class="hint">
                0 表示自动（按频道数 + 插件数 + 10 计算）。当前：
                {{ effective.default_concurrency_auto ? `自动（${effective.default_concurrency}）` : effective.default_concurrency }}
              </span>
            </div>
            <div class="field">
              <label>出口并发上限 <span class="badge badge-warning">重启生效</span></label>
              <input v-model.number="form.outbound_max_concurrency" class="input" type="number" min="0" max="100000" />
              <span class="hint">0 使用默认值 128；所有插件与频道请求共享这道总闸。当前 {{ effective.outbound_max_concurrency }}。</span>
            </div>
          </div>
        </div>
      </section>

      <section class="section-card">
        <div class="section-head">
          <div>
            <div class="section-title"><AdminIcon name="shield" :size="16" /> 网页搜索签名</div>
            <div class="section-desc">匿名网页搜索凭证使用该密钥签名，绑定 IP 与 User-Agent</div>
          </div>
          <span class="badge" :class="secretSet ? 'badge-success' : ''">{{ secretSet ? '已配置' : '随机密钥' }}</span>
        </div>
        <div class="section-body">
          <div class="field">
            <label>签名密钥</label>
            <input
              v-model="secret"
              class="input"
              type="password"
              autocomplete="new-password"
              :disabled="clearSecret"
              :placeholder="secretSet ? '已设置，留空保持不变' : '未设置，使用进程随机密钥'"
            />
            <span class="hint">至少 16 个字符。多副本部署时必须填写相同值；修改后现有网页凭证立即失效。</span>
          </div>
          <label class="clear-secret">
            <input v-model="clearSecret" type="checkbox" />
            <span>清除已保存的密钥，恢复为进程随机密钥</span>
          </label>
        </div>
      </section>
    </template>
  </div>
</template>

<script setup lang="ts">
import { onMounted, reactive, ref } from 'vue';
import {
  adminGetRuntime,
  adminSaveRuntime,
  type RuntimeEffectiveSettings,
  type RuntimeSettings,
} from '@/api/auth';
import { toast } from '@/lib/toast';
import AdminIcon from './AdminIcon.vue';

const loading = ref(true);
const saving = ref(false);
const error = ref('');
const secret = ref('');
const secretSet = ref(false);
const clearSecret = ref(false);
const effective = reactive<RuntimeEffectiveSettings>({
  proxy_url: '',
  cache_enabled: true,
  cache_ttl_minutes: 60,
  default_concurrency: 0,
  default_concurrency_auto: true,
  outbound_max_concurrency: 128,
});
const form = reactive<RuntimeSettings>({
  proxy_url: '',
  cache_enabled: true,
  cache_ttl_minutes: 60,
  default_concurrency: 0,
  outbound_max_concurrency: 0,
});

const restartLabels: Record<string, string> = {
  proxy_url: '出站代理',
  cache_enabled: '缓存开关',
  outbound_max_concurrency: '出口并发',
  default_concurrency: '默认并发回到自动',
};

const load = async () => {
  loading.value = true;
  error.value = '';
  try {
    const data = await adminGetRuntime();
    Object.assign(form, data.settings);
    Object.assign(effective, data.effective);
    secretSet.value = data.web_token_secret_set;
    secret.value = '';
    clearSecret.value = false;
  } catch (err: any) {
    error.value = err.response?.data?.error || '读取运行参数失败';
  } finally {
    loading.value = false;
  }
};

const save = async () => {
  saving.value = true;
  error.value = '';
  try {
    const data = await adminSaveRuntime({
      ...form,
      web_token_secret: clearSecret.value ? '' : secret.value.trim(),
      clear_web_token_secret: clearSecret.value,
    });
    Object.assign(form, data.settings);
    Object.assign(effective, data.effective);
    secretSet.value = data.web_token_secret_set;
    secret.value = '';
    clearSecret.value = false;
    const pending = (data.restart_required || []).map((key) => restartLabels[key] || key);
    if (pending.length) {
      toast.info('已保存，部分参数需重启', `${pending.join('、')} 将在容器重启后生效。`);
    } else {
      toast.success('运行参数已保存', '新配置已应用。');
    }
  } catch (err: any) {
    error.value = err.response?.data?.error || '保存运行参数失败';
    toast.error('保存失败', error.value);
  } finally {
    saving.value = false;
  }
};

onMounted(load);
</script>

<style scoped>
.settings-bar {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 1rem;
  padding: 0.8rem 1rem;
  border: 1px solid hsl(var(--border));
  border-radius: 0.8rem;
  background: hsl(var(--background-light, 0 0% 100%) / 0.92);
  box-shadow: var(--console-shadow);
  position: sticky;
  top: calc(var(--console-topbar-h, 6.2rem) + 0.65rem);
  z-index: 20;
}
.settings-bar-copy strong { display: block; font-size: 0.82rem; }
.settings-bar-copy span { display: block; margin-top: 0.15rem; font-size: 0.7rem; color: hsl(var(--muted-foreground)); }
.settings-grid { display: grid; grid-template-columns: 1fr 1fr; gap: 1rem; }
.effective-line {
  margin-top: 0.9rem;
  padding: 0.6rem 0.7rem;
  border: 1px dashed hsl(var(--border));
  border-radius: 0.6rem;
  background: hsl(var(--muted) / 0.3);
  font-size: 0.72rem;
  color: hsl(var(--muted-foreground));
  overflow-wrap: anywhere;
}
.effective-line strong { color: hsl(var(--foreground)); }
.clear-secret {
  display: inline-flex;
  align-items: center;
  gap: 0.5rem;
  margin-top: 0.7rem;
  font-size: 0.72rem;
  color: hsl(var(--muted-foreground));
  cursor: pointer;
}
.clear-secret input { accent-color: hsl(var(--primary)); }
@media (max-width: 900px) {
  .settings-grid { grid-template-columns: 1fr; }
  .settings-bar { position: static; flex-direction: column; align-items: flex-start; }
}
</style>
