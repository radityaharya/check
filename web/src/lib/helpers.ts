import type { Check, CheckType } from '@/types';

export function timeAgo(dateStr: string | undefined): string {
  if (!dateStr) return 'never';
  const seconds = Math.floor((Date.now() - new Date(dateStr).getTime()) / 1000);
  if (seconds < 60) return seconds + 's ago';
  if (seconds < 3600) return Math.floor(seconds / 60) + 'm ago';
  return Math.floor(seconds / 3600) + 'h ago';
}

export function formatDate(dateStr: string | undefined): string {
  if (!dateStr) return 'Never';
  return new Date(dateStr).toLocaleString();
}

export function getCheckTarget(check: Check): string {
  switch (check.type) {
    case 'ping':
      return check.host || 'N/A';
    case 'postgres':
      return 'PostgreSQL Database';
    case 'dns':
      return check.dns_hostname || 'N/A';
    case 'tailscale':
      return 'Device: ' + (check.tailscale_device_id || 'N/A');
    case 'tailscale_service':
      return `${check.tailscale_service_host}:${check.tailscale_service_port || 'N/A'}`;
    default:
      return check.url || 'N/A';
  }
}

export function getCheckTypeClass(type: CheckType): string {
  const classes: Record<CheckType, string> = {
    http: 'bg-terminal-blue/20 text-terminal-blue',
    ping: 'bg-terminal-cyan/20 text-terminal-cyan',
    postgres: 'bg-terminal-purple/20 text-terminal-purple',
    json_http: 'bg-terminal-yellow/20 text-terminal-yellow',
    dns: 'bg-terminal-green/20 text-terminal-green',
    tailscale: 'bg-terminal-cyan/20 text-terminal-cyan',
    tailscale_service: 'bg-terminal-purple/20 text-terminal-purple',
  };
  return classes[type] || classes.http;
}

export function getUptimeClass(uptime: number): string {
  if (uptime >= 99) return 'text-terminal-green';
  if (uptime >= 95) return 'text-terminal-yellow';
  return 'text-terminal-red';
}

export function parseStatusCodes(input: string): number[] {
  const codes = input
    .split(',')
    .map((s) => parseInt(s.trim()))
    .filter((n) => !isNaN(n));
  return codes.length > 0 ? codes : [200];
}

// Local storage helpers
const STORAGE_KEYS = {
  EXPANDED_CHECKS: 'gocheck_expandedChecks',
  EXPANDED_GROUPS: 'gocheck_expandedGroups',
  TIME_RANGE: 'gocheck_timeRange',
  THEME: 'gocheck_theme',
  GROUP_COUNT: 'gocheck_groupCount',
  CHECK_COUNT: 'gocheck_checkCount',
} as const;

export function getStoredExpandedChecks(): number[] {
  try {
    return JSON.parse(localStorage.getItem(STORAGE_KEYS.EXPANDED_CHECKS) || '[]');
  } catch {
    return [];
  }
}

export function setStoredExpandedChecks(checks: number[]): void {
  localStorage.setItem(STORAGE_KEYS.EXPANDED_CHECKS, JSON.stringify(checks));
}

export function getStoredExpandedGroups(): number[] {
  try {
    return JSON.parse(localStorage.getItem(STORAGE_KEYS.EXPANDED_GROUPS) || '[]');
  } catch {
    return [];
  }
}

export function setStoredExpandedGroups(groups: number[]): void {
  localStorage.setItem(STORAGE_KEYS.EXPANDED_GROUPS, JSON.stringify(groups));
}

export function getStoredTimeRange(): string {
  return localStorage.getItem(STORAGE_KEYS.TIME_RANGE) || '1d';
}

export function setStoredTimeRange(range: string): void {
  localStorage.setItem(STORAGE_KEYS.TIME_RANGE, range);
}

export function getStoredTheme(): 'light' | 'dark' {
  return (localStorage.getItem(STORAGE_KEYS.THEME) as 'light' | 'dark') || 'dark';
}

export function setStoredTheme(theme: 'light' | 'dark'): void {
  localStorage.setItem(STORAGE_KEYS.THEME, theme);
}

export function getSkeletonCounts(): { groupCount: number; checkCount: number } {
  return {
    groupCount: parseInt(localStorage.getItem(STORAGE_KEYS.GROUP_COUNT) || '3'),
    checkCount: parseInt(localStorage.getItem(STORAGE_KEYS.CHECK_COUNT) || '2'),
  };
}

export function setSkeletonCounts(groupCount: number, checkCount: number): void {
  localStorage.setItem(STORAGE_KEYS.GROUP_COUNT, groupCount.toString());
  localStorage.setItem(STORAGE_KEYS.CHECK_COUNT, checkCount.toString());
}

// Response time formatting and status
export function formatResponseTime(ms: number | undefined): string {
  if (ms === undefined || ms === null) return '-';
  if (ms < 1) return '<1ms';
  if (ms < 1000) return `${Math.round(ms)}ms`;
  return `${(ms / 1000).toFixed(2)}s`;
}

export function getStatusClass(responseTimeMs: number): string {
  if (responseTimeMs < 200) return 'text-terminal-green';
  if (responseTimeMs < 500) return 'text-terminal-yellow';
  return 'text-terminal-red';
}

// Generic preference helpers
export function getPreference<T>(key: string, defaultValue: T): T {
  try {
    const stored = localStorage.getItem(`gocheck_${key}`);
    return stored ? (stored as unknown as T) : defaultValue;
  } catch {
    return defaultValue;
  }
}

export function setPreference(key: string, value: unknown): void {
  localStorage.setItem(`gocheck_${key}`, String(value));
}
