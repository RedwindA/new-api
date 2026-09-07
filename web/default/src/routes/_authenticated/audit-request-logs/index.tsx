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
import { createFileRoute, redirect } from '@tanstack/react-router'
import z from 'zod'

import { AuditRequestLogs } from '@/features/audit-request-logs'
import { ROLE } from '@/lib/roles'
import { useAuthStore } from '@/stores/auth-store'

const auditRequestLogsSearchSchema = z.object({
  page: z.number().optional().catch(1),
  pageSize: z.number().optional().catch(undefined),
  startTime: z.number().optional(),
  endTime: z.number().optional(),
  ip: z.string().optional().catch(''),
  path: z.string().optional().catch(''),
  method: z.string().optional().catch(''),
  statusCodes: z.array(z.number()).optional().catch(undefined),
  result: z.number().optional().catch(undefined),
  username: z.string().optional().catch(''),
  userId: z.number().optional(),
  requestId: z.string().optional().catch(''),
})

export const Route = createFileRoute('/_authenticated/audit-request-logs/')({
  beforeLoad: () => {
    const { auth } = useAuthStore.getState()

    if (!auth.user || auth.user.role < ROLE.ADMIN) {
      throw redirect({
        to: '/403',
      })
    }
  },
  validateSearch: auditRequestLogsSearchSchema,
  component: AuditRequestLogs,
})
