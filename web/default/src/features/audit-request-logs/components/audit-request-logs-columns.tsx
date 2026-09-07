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
import type { ColumnDef } from '@tanstack/react-table'
import { useTranslation } from 'react-i18next'

import { BadgeCell } from '@/components/data-table'
import { StatusBadge, type StatusVariant } from '@/components/status-badge'
import { formatTimestampToDate } from '@/lib/format'

import { auditResultLabelKey, auditResultVariant } from '../lib/result'
import type { AuditRequestLog } from '../types'

function statusVariant(statusCode: number): StatusVariant {
  if (statusCode >= 500) return 'red'
  if (statusCode >= 400) return 'orange'
  if (statusCode >= 300) return 'blue'
  if (statusCode >= 200) return 'green'
  return 'grey'
}

export function useAuditRequestLogsColumns(): ColumnDef<AuditRequestLog>[] {
  const { t } = useTranslation()

  return [
    {
      accessorKey: 'created_at',
      header: t('Time'),
      meta: { mobileTitle: true },
      cell: ({ row }) => (
        <span className='font-mono text-xs tabular-nums'>
          {formatTimestampToDate(row.original.created_at, 'seconds')}
        </span>
      ),
      size: 170,
    },
    {
      accessorKey: 'ip',
      header: t('IP'),
      cell: ({ row }) => (
        <span className='font-mono text-xs'>{row.original.ip || '-'}</span>
      ),
      size: 130,
    },
    {
      accessorKey: 'method',
      header: t('Method'),
      cell: ({ row }) => (
        <span className='font-mono text-xs font-medium'>
          {row.original.method}
        </span>
      ),
      size: 80,
    },
    {
      accessorKey: 'path',
      header: t('Path'),
      meta: { mobileTitle: true },
      cell: ({ row }) => (
        <span className='block max-w-[280px] truncate font-mono text-xs'>
          {row.original.path}
        </span>
      ),
      size: 280,
    },
    {
      accessorKey: 'status_code',
      header: t('Status'),
      cell: ({ row }) => (
        <BadgeCell>
          <StatusBadge
            variant={statusVariant(row.original.status_code)}
            label={String(row.original.status_code)}
          />
        </BadgeCell>
      ),
      size: 90,
    },
    {
      accessorKey: 'result',
      header: t('Result'),
      cell: ({ row }) => (
        <BadgeCell>
          <StatusBadge
            variant={auditResultVariant(row.original.result)}
            label={t(auditResultLabelKey(row.original.result))}
          />
        </BadgeCell>
      ),
      size: 90,
    },
    {
      accessorKey: 'latency_ms',
      header: t('Latency'),
      meta: { mobileHidden: true },
      cell: ({ row }) => (
        <span className='font-mono text-xs tabular-nums'>
          {row.original.latency_ms}ms
        </span>
      ),
      size: 90,
    },
    {
      accessorKey: 'username',
      header: t('Username'),
      cell: ({ row }) => (
        <span className='text-sm'>
          {row.original.username || t('Anonymous')}
        </span>
      ),
      size: 120,
    },
    {
      accessorKey: 'request_id',
      header: t('Request ID'),
      meta: { mobileHidden: true },
      cell: ({ row }) => (
        <span className='block max-w-[180px] truncate font-mono text-xs'>
          {row.original.request_id}
        </span>
      ),
      size: 180,
    },
  ]
}
