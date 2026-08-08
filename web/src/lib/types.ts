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
  access_token: string
  refresh_token: string
  username: string
}

export interface RefreshResponse {
  access_token: string
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
  throttle_window: number
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
