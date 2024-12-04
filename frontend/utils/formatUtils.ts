// utils/formatUtils.ts
export const formatBytes = (bytes: number, decimals = 2) => {
    if (!bytes || isNaN(bytes)) return '0 B';
    const k = 1024;
    const sizes = ['B', 'KB', 'MB', 'GB', 'TB'];
    const i = Math.floor(Math.log(bytes) / Math.log(k));
    return `${parseFloat((bytes / Math.pow(k, i)).toFixed(decimals))} ${sizes[i]}`;
  };
  
  export const formatBitrate = (bytesPerSecond: number) => {
    if (!bytesPerSecond || isNaN(bytesPerSecond)) return '0 Kbps';
    const bitsPerSecond = bytesPerSecond * 8;
    if (bitsPerSecond < 1000000) {
      return `${(bitsPerSecond / 1000).toFixed(1)} Kbps`;
    }
    return `${(bitsPerSecond / 1000000).toFixed(1)} Mbps`;
  };
  
  export const formatPercentage = (value: number) => {
    if (!value || isNaN(value)) return '0.0%';
    return `${value.toFixed(1)}%`;
  };
  
  export const formatOps = (ops: number) => {
    if (!ops || isNaN(ops)) return '0 IOPS';
    if (ops < 1000) {
      return `${ops.toFixed(1)} IOPS`;
    }
    return `${(ops / 1000).toFixed(1)}K IOPS`;
  };