// API Types for Gocheck

export interface User {
  id: number;
  username: string;
}

export interface AuthState {
  isAuthenticated: boolean;
  needsSetup: boolean;
  user: User | null;
}

export interface LoginCredentials {
  username: string;
  password: string;
}

export interface Check {
  id: number;
  name: string;
  type: CheckType;
  url?: string;
  host?: string;
  method?: string;
  interval_seconds: number;
  timeout_seconds: number;
  retries: number;
  retry_delay_seconds: number;
  enabled: boolean;
  expected_status_codes?: number[];
  json_path?: string;
  expected_json_value?: string;
  postgres_conn_string?: string;
  postgres_query?: string;
  expected_query_value?: string;
  dns_hostname?: string;
  dns_record_type?: string;
  expected_dns_value?: string;
  group_id?: number | null;
  tags?: Tag[];
  tag_ids?: number[];
  tailscale_device_id?: string;
  tailscale_service_host?: string;
  tailscale_service_port?: number;
  tailscale_service_protocol?: string;
  tailscale_service_path?: string;
  snapshot_url?: string;
  snapshot_taken_at?: string;
  snapshot_error?: string;
  is_up?: boolean;
  last_checked_at?: string;
  last_status?: CheckStatus;
  history?: CheckStatus[];
}

export type CheckType =
  | 'http'
  | 'ping'
  | 'postgres'
  | 'json_http'
  | 'dns'
  | 'tailscale'
  | 'tailscale_service';

export interface CheckStatus {
  id?: number;
  check_id: number;
  success: boolean;
  status_code?: number;
  response_time_ms: number;
  error_message?: string;
  response_body?: string;
  actual_value?: string;
  checked_at: string;
}

export interface CheckGroup {
  id: number;
  name: string;
  sort_order: number;
  checks: Check[];
  is_up: boolean;
  up_count: number;
  down_count: number;
}

export interface Group {
  id: number;
  name: string;
  sort_order: number;
}

export interface Tag {
  id: number;
  name: string;
  color: string;
}

export interface Stats {
  total_checks: number;
  active_checks: number;
  up_checks: number;
  total_uptime: number;
}

export interface Settings {
  discord_webhook_url: string;
  gotify_url: string;
  gotify_server_url: string;
  gotify_token: string;
  tailscale_api_key: string;
  tailscale_tailnet: string;
  cloudflare_account_id: string;
  cloudflare_api_token: string;
}

export interface TailscaleDevice {
  id: string;
  name: string;
  hostname: string;
  addresses: string[];
  online: boolean;
}

export interface APIKey {
  id: number;
  name: string;
  created_at: string;
  last_used_at?: string;
}

export interface Passkey {
  id: number;
  name: string;
  created_at: string;
  last_used_at?: string;
}

// CheckLog type for history entries
export interface CheckLog {
  id: number;
  check_id: number;
  success: boolean;
  response_time_ms: number;
  http_status_code?: number;
  message?: string;
  actual_value?: string;
  error?: string;
  checked_at: string;
}

export interface CheckUpdate {
  check_id: number;
  is_up: boolean;
  last_checked_at: string;
  last_status: CheckStatus;
}

export interface CheckFormData {
  name: string;
  type: CheckType;
  url: string;
  interval_seconds: number;
  timeout_seconds: number;
  retries: number;
  retry_delay_seconds: number;
  enabled: boolean;
  method: string;
  expected_status_codes: number[];
  json_path: string;
  expected_json_value: string;
  host: string;
  postgres_conn_string: string;
  postgres_query: string;
  expected_query_value: string;
  dns_hostname: string;
  dns_record_type: string;
  expected_dns_value: string;
  group_id: number | string;
  tag_ids: number[];
  tailscale_device_id: string;
  tailscale_service_host: string;
  tailscale_service_port: number;
  tailscale_service_protocol: string;
  tailscale_service_path: string;
}

export const DEFAULT_CHECK_FORM_DATA: CheckFormData = {
  name: '',
  type: 'http',
  url: '',
  interval_seconds: 60,
  timeout_seconds: 10,
  retries: 0,
  retry_delay_seconds: 5,
  enabled: true,
  method: 'GET',
  expected_status_codes: [200],
  json_path: '',
  expected_json_value: '',
  host: '',
  postgres_conn_string: '',
  postgres_query: '',
  expected_query_value: '',
  dns_hostname: '',
  dns_record_type: 'A',
  expected_dns_value: '',
  group_id: '',
  tag_ids: [],
  tailscale_device_id: '',
  tailscale_service_host: '',
  tailscale_service_port: 80,
  tailscale_service_protocol: 'http',
  tailscale_service_path: '/',
};

export interface GroupFormData {
  name: string;
  sort_order: number;
}

export interface TagFormData {
  name: string;
  color: string;
}

export type TimeRange = '15m' | '30m' | '60m' | '1d' | '30d';

export const TIME_RANGES: TimeRange[] = ['15m', '30m', '60m', '1d', '30d'];

export const TAG_COLORS = [
  '#c4a7e7',
  '#f38ba8',
  '#f9e2af',
  '#7aa2f7',
  '#b794f6',
  '#94e2d5',
  '#fab387',
  '#f5c2e7',
  '#74c7ec',
  '#b4befe',
];
