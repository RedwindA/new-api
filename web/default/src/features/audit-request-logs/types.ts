/*
Copyright (C) 2023-2026 QuantumNous

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU Affero General Public License as
published by the Free Software Foundation, either version 3 of the
License, or (at your option) any later version.

This program is distributed in the hope that it will be useful,
but WITHOUT ANY WARRANTY; without even the implied warranty of
MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
GNU Affero General Public License for more details.

You should have received a copy of the GNU Affero General Public License
along with this program. If not, see <https://www.gnu.org/licenses/>.

For commercial licensing, please contact support@quantumnous.com
*/
export interface AuditRequestLog {
  created_at: number
  request_id: string
  ip: string
  user_agent: string
  method: string
  route: string
  path: string
  query: string
  status_code: number
  result: number
  latency_ms: number
  user_id: number
  username: string
  user_role: number
  request_body?: string
  response_body?: string
  body_truncated: number
}

export interface AuditRequestLogQuery {
  p?: number
  page_size?: number
  start_timestamp?: number
  end_timestamp?: number
  ip?: string
  path?: string
  method?: string
  status_code?: number[]
  result?: number
  username?: string
  user_id?: number
  request_id?: string
}

export interface AuditRequestLogsPage {
  items: AuditRequestLog[]
  total: number
  page: number
  page_size: number
}

export interface AuditRequestLogsResponse {
  success: boolean
  message?: string
  data?: AuditRequestLogsPage
}

export interface AuditRequestStatusCodesResponse {
  success: boolean
  message?: string
  data?: number[]
}

export interface AuditRequestLogDetailResponse {
  success: boolean
  message?: string
  data?: AuditRequestLog
}

export interface AuditRequestLogFilters {
  startTime?: Date
  endTime?: Date
  ip: string
  path: string
  method: string
  statusCodes: string[]
  result: string
  username: string
  userId: string
  requestId: string
}
