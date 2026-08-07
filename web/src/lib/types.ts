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
