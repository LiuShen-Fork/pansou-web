export type ConsoleMode = 'admin' | 'user';

export interface ConsoleNavItem {
  key: string;
  label: string;
  icon: string;
  badge?: string | number;
}

export interface ConsolePageMeta {
  title: string;
  subtitle: string;
}

export const consoleNav: Record<ConsoleMode, ConsoleNavItem[]> = {
  admin: [
    { key: 'overview', label: '运营概览', icon: 'dashboard' },
    { key: 'users', label: '用户管理', icon: 'users' },
    { key: 'logs', label: '搜索日志', icon: 'logs' },
    { key: 'settings', label: '认证与站点', icon: 'settings' },
    { key: 'oauth', label: 'OAuth2 登录', icon: 'key' },
    { key: 'upstream', label: '上游自动同步', icon: 'upstream' },
    { key: 'system', label: '系统状态', icon: 'system' },
  ],
  user: [
    { key: 'overview', label: '我的概览', icon: 'dashboard' },
    { key: 'profile', label: '个人资料', icon: 'user' },
    { key: 'security', label: '安全设置', icon: 'lock' },
    { key: 'token', label: 'API 令牌', icon: 'key' },
    { key: 'oauth', label: '第三方绑定', icon: 'external' },
  ],
};

export const consolePages: Record<ConsoleMode, Record<string, ConsolePageMeta>> = {
  admin: {
    overview: { title: '运营概览', subtitle: '用户增长、搜索趋势与服务健康度' },
    users: { title: '用户管理', subtitle: '账号、角色、状态与每日搜索配额' },
    logs: { title: '搜索日志', subtitle: '按用户、来源与关键词追踪搜索行为' },
    settings: { title: '认证与站点', subtitle: '登录方式、注册策略、API 权限与 SMTP' },
    oauth: { title: 'OAuth2 登录', subtitle: '配置唯一的第三方账号登录入口' },
    upstream: { title: '上游自动同步', subtitle: '定时刷新 TG 频道与插件默认值' },
    system: { title: '系统状态', subtitle: '插件、频道、TG 可达性与存活观测' },
  },
  user: {
    overview: { title: '我的概览', subtitle: '账号状态、搜索配额与快捷入口' },
    profile: { title: '个人资料', subtitle: '用户名与邮箱设置' },
    security: { title: '安全设置', subtitle: '修改登录密码与查看登录方式' },
    token: { title: 'API 令牌', subtitle: '个人访问令牌管理，最多 10 个' },
    oauth: { title: '第三方绑定', subtitle: 'OAuth2 登录方式' },
  },
};
