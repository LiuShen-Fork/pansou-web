import type {
  DetectionSettings,
  LinkHealthRecord,
  LinkHealthState,
} from '@/types';

export const DETECTION_SETTINGS_STORAGE_KEY = 'pansou_detection_settings';
export const LINK_HEALTH_CACHE_STORAGE_KEY = 'pansou_link_health_cache';

const DEFAULT_DETECTION_SETTINGS: DetectionSettings = {
  enabled: false,
};

const CACHE_LIMIT = 300;

type LinkHealthCacheMap = Record<string, LinkHealthRecord>;

const trimValue = (value: string) => value.trim();

export const getDefaultDetectionSettings = (): DetectionSettings => ({
  ...DEFAULT_DETECTION_SETTINGS,
});

export const loadDetectionSettings = (): DetectionSettings => {
  try {
    const saved = localStorage.getItem(DETECTION_SETTINGS_STORAGE_KEY);
    if (!saved) return getDefaultDetectionSettings();

    const parsed = JSON.parse(saved);
    return {
      enabled: parsed?.enabled === true,
    };
  } catch {
    return getDefaultDetectionSettings();
  }
};

export const persistDetectionSettings = (settings: DetectionSettings) => {
  localStorage.setItem(DETECTION_SETTINGS_STORAGE_KEY, JSON.stringify(settings));
};

export const buildHealthCacheKey = (diskType: string, url: string) => {
  return `health:${diskType}:${normalizeDetectionUrl(url)}`;
};

export const normalizeDetectionUrl = (url: string) => {
  const value = trimValue(url);
  if (!value) return '';

  try {
    const parsed = new URL(value);
    parsed.hash = '';
    parsed.hostname = parsed.hostname.toLowerCase();
    return parsed.toString();
  } catch {
    return value;
  }
};

const getStateTtl = (state: LinkHealthState) => {
  switch (state) {
    case 'ok':
      return 24 * 60 * 60 * 1000;
    case 'bad':
      return 6 * 60 * 60 * 1000;
    case 'locked':
      return 12 * 60 * 60 * 1000;
    case 'unsupported':
      return 24 * 60 * 60 * 1000;
    case 'uncertain':
      return 60 * 60 * 1000;
    default:
      return 0;
  }
};

export const buildHealthRecord = (
  state: LinkHealthState,
  options?: {
    summary?: string;
    normalized_url?: string;
    checked_at?: number;
    expires_at?: number;
  }
): LinkHealthRecord => {
  const checkedAt = options?.checked_at ?? Date.now();
  const expiresAt = options?.expires_at ?? checkedAt + getStateTtl(state);

  return {
    state,
    summary: options?.summary,
    checked_at: checkedAt,
    expires_at: expiresAt,
    normalized_url: options?.normalized_url,
  };
};

export const loadLinkHealthCache = (): LinkHealthCacheMap => {
  try {
    const raw = localStorage.getItem(LINK_HEALTH_CACHE_STORAGE_KEY);
    if (!raw) return {};

    const parsed = JSON.parse(raw) as LinkHealthCacheMap;
    if (!parsed || typeof parsed !== 'object') return {};
    return parsed;
  } catch {
    return {};
  }
};

export const persistLinkHealthCache = (cache: LinkHealthCacheMap) => {
  const entries = Object.entries(cache)
    .sort((a, b) => (b[1]?.checked_at ?? 0) - (a[1]?.checked_at ?? 0))
    .slice(0, CACHE_LIMIT);

  localStorage.setItem(
    LINK_HEALTH_CACHE_STORAGE_KEY,
    JSON.stringify(Object.fromEntries(entries))
  );
};

export const pruneExpiredHealthCache = (cache: LinkHealthCacheMap) => {
  const now = Date.now();
  const next: LinkHealthCacheMap = {};

  Object.entries(cache).forEach(([key, value]) => {
    if (value && value.expires_at > now) {
      next[key] = value;
    }
  });

  return next;
};

export const getCachedLinkHealth = (diskType: string, url: string) => {
  const cache = pruneExpiredHealthCache(loadLinkHealthCache());
  const key = buildHealthCacheKey(diskType, url);
  const record = cache[key];

  if (!record) {
    persistLinkHealthCache(cache);
    return null;
  }

  persistLinkHealthCache(cache);
  return record;
};

export const saveCachedLinkHealth = (
  diskType: string,
  url: string,
  record: LinkHealthRecord
) => {
  const cache = pruneExpiredHealthCache(loadLinkHealthCache());
  const key = buildHealthCacheKey(diskType, url);
  cache[key] = record;
  persistLinkHealthCache(cache);
};
