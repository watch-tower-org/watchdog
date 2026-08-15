export interface SuccessResponse<T = unknown> {
  success: boolean
  code: number
  message: string
  data?: T
}

export interface ErrorResponse {
  success: boolean
  code: number
  error?: string
}

export interface PageInfo {
  current_page: number
  limit: number
  total: number
  total_pages: number
  has_next_page: boolean
  has_previous_page: boolean
}

export interface Paginated<T> {
  data: T[]
  page_info: PageInfo
}

export interface LoginResponse {
  username: string
}

export interface MeResponse {
  username: string
}

export interface Settings {
  id: number
  setup_complete: boolean
  created_at: string
  updated_at: string
}

export interface EmailSettings {
  id: number
  smtp_host: string
  smtp_port: number
  smtp_username: string
  smtp_from_email: string
  smtp_from_name: string
  created_at: string
  updated_at: string
}

export interface AlertSettings {
  id: number
  throttle_window: number
  created_at: string
  updated_at: string
}

export interface RecipientList {
  id: number
  name: string
  emails: string[]
  created_at: string
  updated_at: string
}

export interface ApiKey {
  id: number
  name: string
  masked: string
  project: string
  is_active: boolean
  last_used_at: string | null
  created_at: string
  updated_at: string
}

export interface CreateApiKeyResponse {
  api_key: ApiKey
  key: string
  key_id: number
}

export type IssueStatus = 'open' | 'resolved' | 'muted'

export interface Issue {
  id: number
  fingerprint: string
  title: string
  project: string
  tag: string
  status: IssueStatus
  first_seen: string
  last_seen: string
  count: number
  created_at: string
  updated_at: string
}

export interface IssueEvent {
  id: number
  issue_id: number
  timestamp: string
  message: string
  stack_trace: string
  context: Record<string, unknown> | null
  project: string
  tag: string
  created_at: string
}

export type AlertTriggerType = 'new_issue' | 'spike' | 'regression'

export interface AlertRule {
  id: number
  name: string
  trigger_type: AlertTriggerType
  project: string
  tag: string
  threshold: number
  window_minutes: number
  throttle_window: number | null
  recipient_list_id: number
  recipient_list: RecipientList | null
  is_active: boolean
  created_at: string
  updated_at: string
}

export interface AlertLog {
  id: number
  issue_id: number
  rule_id: number
  sent_at: string
  recipients: string[]
  issue: Issue | null
  rule: AlertRule | null
}

export interface TrendBucket {
  ts: string
  new_issues: number
  events: number
  alerts: number
}

export interface TrendItem {
  name: string
  count: number
}

export interface DashboardTrends {
  range: string
  buckets: TrendBucket[]
  issues_by_status: TrendItem[]
  top_projects: TrendItem[]
  top_tags: TrendItem[]
}

export type MonitoredServiceStatus = 'up' | 'down'

export interface MonitoredService {
  id: number
  name: string
  url: string
  interval_seconds: number
  timeout_seconds: number
  failures_before_alert: number
  recipient_list_id: number | null
  recipient_list: RecipientList | null
  is_active: boolean
  last_status: MonitoredServiceStatus | null
  last_checked_at: string | null
  last_up_at: string | null
  last_down_at: string | null
  consecutive_failures: number
  created_at: string
  updated_at: string
}

export interface UptimeCheck {
  id: number
  service_id: number
  checked_at: string
  status: MonitoredServiceStatus
  status_code: number | null
  response_time_ms: number | null
  error: string | null
}

export interface MonitoredServiceDetail extends MonitoredService {
  recent_checks: UptimeCheck[]
}

export interface CheckResult {
  status: MonitoredServiceStatus
  status_code: number
  response_time_ms: number
  error: string | null
}

export interface CheckNowResponse {
  service: MonitoredService
  result: CheckResult
}
