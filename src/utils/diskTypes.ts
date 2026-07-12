// 网盘类型映射
export const diskTypeMap: Record<string, string> = {
  'quark': '夸克',
  'baidu': '百度',
  'aliyun': '阿里',
  'guangya': '光鸭',
  'tianyi': '天翼',
  'uc': 'UC',
  'mobile': '移动',
  '115': '115',
  'xunlei': '迅雷',
  '123': '123',
  'pikpak': 'PikPak',
  'magnet': '磁力',
  'ed2k': '电驴',
  'other': '其他'
};

export const diskTypeSortOrder = Object.keys(diskTypeMap);

export const sortDiskTypes = (types: string[]): string[] => {
  const order = new Map(diskTypeSortOrder.map((type, index) => [type, index]));
  return [...types].sort((a, b) => {
    const aIndex = order.get(a) ?? Number.MAX_SAFE_INTEGER;
    const bIndex = order.get(b) ?? Number.MAX_SAFE_INTEGER;
    if (aIndex !== bIndex) return aIndex - bIndex;
    return a.localeCompare(b);
  });
};

// 获取网盘类型的中文名称
export const getDiskTypeName = (type: string): string => {
  return diskTypeMap[type] || type;
}; 
