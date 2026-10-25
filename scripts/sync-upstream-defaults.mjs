#!/usr/bin/env node
/**
 * 从上游 pansou 仓库的 Dockerfile 抓取默认插件与 Telegram 频道列表。
 *
 * 上游会持续增删插件和频道（例如 tg 搜索频道改名/失效），这些数据以前写死在
 * 本仓库的 Dockerfile 与 docker-compose 里，每次都要手工同步。现在改为：
 *
 *   1. 本地 / CI 构建前运行本脚本，生成 deploy/upstream-defaults.env；
 *   2. 镜像启动时 start.sh 读取该文件，仅在用户没有显式设置环境变量时生效；
 *   3. 抓取失败时保留仓库里已提交的快照，构建不会因此中断。
 *
 * 用法：
 *   node scripts/sync-upstream-defaults.mjs            # 失败时保留快照
 *   STRICT=1 node scripts/sync-upstream-defaults.mjs   # 失败时退出码 1
 *   UPSTREAM_REF=v1.2.3 node scripts/sync-upstream-defaults.mjs
 */

import { readFile, writeFile } from 'node:fs/promises';
import { dirname, resolve } from 'node:path';
import { fileURLToPath } from 'node:url';

const ROOT = resolve(dirname(fileURLToPath(import.meta.url)), '..');
const OUTPUT = resolve(ROOT, 'deploy/upstream-defaults.env');

const REPO = process.env.UPSTREAM_REPO || 'fish2018/pansou';
const REF = process.env.UPSTREAM_REF || 'main';
const STRICT = process.env.STRICT === '1' || process.argv.includes('--strict');

const RAW_DOCKERFILE = `https://raw.githubusercontent.com/${REPO}/${REF}/Dockerfile`;

function parseEnvValues(dockerfile) {
  // 把行尾反斜杠续行拼成一行，便于用简单正则匹配 ENV 值。
  const flat = dockerfile.replace(/\\\r?\n/g, ' ');
  const pick = (name) => {
    const match = flat.match(new RegExp(`(?:^|\\s)${name}=([^\\s]+)`));
    return match ? match[1].trim().replace(/^["']|["']$/g, '') : '';
  };
  return {
    channels: pick('CHANNELS'),
    plugins: pick('ENABLED_PLUGINS'),
  };
}

async function resolveRevision() {
  try {
    const resp = await fetch(`https://api.github.com/repos/${REPO}/commits/${REF}`, {
      headers: { Accept: 'application/vnd.github+json', 'User-Agent': 'pansou-web-sync' },
    });
    if (!resp.ok) return '';
    const data = await resp.json();
    return data?.sha || '';
  } catch {
    return '';
  }
}

async function main() {
  let dockerfile = '';
  try {
    const resp = await fetch(RAW_DOCKERFILE, {
      headers: { 'User-Agent': 'pansou-web-sync' },
    });
    if (!resp.ok) throw new Error(`HTTP ${resp.status}`);
    dockerfile = await resp.text();
  } catch (err) {
    console.warn(`[sync-upstream] 拉取上游 Dockerfile 失败：${err.message}`);
    console.warn('[sync-upstream] 将继续使用仓库中已提交的默认值快照。');
    if (STRICT) process.exit(1);
    try {
      await readFile(OUTPUT, 'utf8');
      return;
    } catch {
      console.error('[sync-upstream] 未找到已提交的快照，无法继续。');
      process.exit(1);
    }
  }

  const { channels, plugins } = parseEnvValues(dockerfile);
  if (!channels || !plugins) {
    console.error('[sync-upstream] 上游 Dockerfile 中未找到 CHANNELS 或 ENABLED_PLUGINS。');
    if (STRICT) process.exit(1);
    return;
  }

  // 值与已提交快照一致时不重写文件，避免每次构建/校验都产生时间戳噪音。
  try {
    const existing = await readFile(OUTPUT, 'utf8');
    const pick = (name) => {
      const m = existing.match(new RegExp(`^${name}='([^']*)'`, 'm'));
      return m ? m[1] : '';
    };
    if (pick('UPSTREAM_CHANNELS') === channels && pick('UPSTREAM_ENABLED_PLUGINS') === plugins) {
      console.log('[sync-upstream] 上游默认值与本地快照一致，无需更新。');
      return;
    }
  } catch {
    // 快照不存在时继续生成
  }

  const revision = await resolveRevision();
  const stamp = new Date().toISOString();
  const content = [
    '# 由 scripts/sync-upstream-defaults.mjs 自动生成，请勿手工编辑。',
    `# 来源: https://github.com/${REPO}/blob/${revision || REF}/Dockerfile`,
    `# 生成时间: ${stamp}`,
    `UPSTREAM_SOURCE='https://github.com/${REPO}/blob/${revision || REF}/Dockerfile'`,
    `UPSTREAM_REVISION='${revision || REF}'`,
    `UPSTREAM_CHANNELS='${channels}'`,
    `UPSTREAM_ENABLED_PLUGINS='${plugins}'`,
    '',
  ].join('\n');

  await writeFile(OUTPUT, content, 'utf8');
  const channelCount = channels.split(',').filter(Boolean).length;
  const pluginCount = plugins.split(',').filter(Boolean).length;
  console.log(`[sync-upstream] 已更新 ${OUTPUT}`);
  console.log(`[sync-upstream] 频道 ${channelCount} 个，插件 ${pluginCount} 个（${revision || REF}）`);
}

main().catch((err) => {
  console.error('[sync-upstream] 未预期错误：', err);
  process.exit(STRICT ? 1 : 0);
});
