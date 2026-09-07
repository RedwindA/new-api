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
import { api } from '@/lib/api'

import { buildAuditRequestLogQuery } from './lib/query-params'
import type {
  AuditRequestLogDetailResponse,
  AuditRequestLogQuery,
  AuditRequestLogsResponse,
  AuditRequestStatusCodesResponse,
} from './types'

export async function getAuditRequestLogs(
  params: AuditRequestLogQuery = {}
): Promise<AuditRequestLogsResponse> {
  const query = buildAuditRequestLogQuery({
    p: params.p || 1,
    page_size: params.page_size || 20,
    ...params,
  })
  const res = await api.get(`/api/audit_request/?${query.toString()}`)
  return res.data
}

export async function getAuditRequestStatusCodes(): Promise<AuditRequestStatusCodesResponse> {
  const res = await api.get('/api/audit_request/status_codes')
  return res.data
}

export async function getAuditRequestLogDetail(params: {
  requestId: string
  createdAt: number
}): Promise<AuditRequestLogDetailResponse> {
  const query = buildAuditRequestLogQuery({
    request_id: params.requestId,
  })
  if (params.createdAt) {
    query.set('created_at', String(params.createdAt))
  }
  const res = await api.get(`/api/audit_request/detail?${query.toString()}`)
  return res.data
}
