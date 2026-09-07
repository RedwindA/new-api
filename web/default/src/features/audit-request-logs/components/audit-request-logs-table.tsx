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
import { useQuery } from '@tanstack/react-query'
import { getRouteApi } from '@tanstack/react-router'
import { useState } from 'react'
import { useTranslation } from 'react-i18next'
import { toast } from 'sonner'

import {
  DataTablePage,
  DataTableRow,
  useDataTable,
} from '@/components/data-table'
import { useTableUrlState } from '@/hooks/use-table-url-state'

import { getAuditRequestLogs } from '../api'
import { isAuditRequestDisabledMessage } from '../lib/query-params'
import type { AuditRequestLog } from '../types'
import { AuditRequestDetailDialog } from './audit-request-detail-dialog'
import { useAuditRequestLogsColumns } from './audit-request-logs-columns'
import { AuditRequestLogsFilterBar } from './audit-request-logs-filter-bar'

const route = getRouteApi('/_authenticated/audit-request-logs/')

export function AuditRequestLogsTable() {
  const { t } = useTranslation()
  const columns = useAuditRequestLogsColumns()
  const search = route.useSearch()
  const [selected, setSelected] = useState<AuditRequestLog | null>(null)

  const { pagination, onPaginationChange, ensurePageInRange } =
    useTableUrlState({
      search,
      navigate: route.useNavigate(),
      pagination: { defaultPage: 1, defaultPageSize: 20 },
      globalFilter: { enabled: false },
    })

  const { data, isLoading, isFetching } = useQuery({
    queryKey: [
      'audit-request-logs',
      pagination.pageIndex + 1,
      pagination.pageSize,
      search.startTime,
      search.endTime,
      search.ip,
      search.path,
      search.method,
      search.statusCodes?.join(','),
      search.result,
      search.username,
      search.userId,
      search.requestId,
    ],
    queryFn: async () => {
      const result = await getAuditRequestLogs({
        p: pagination.pageIndex + 1,
        page_size: pagination.pageSize,
        start_timestamp: search.startTime
          ? Math.floor(search.startTime / 1000)
          : undefined,
        end_timestamp: search.endTime
          ? Math.floor(search.endTime / 1000)
          : undefined,
        ip: search.ip,
        path: search.path,
        method: search.method,
        status_code: search.statusCodes,
        result: search.result,
        username: search.username,
        user_id: search.userId,
        request_id: search.requestId,
      })
      if (!result.success) {
        const disabled = isAuditRequestDisabledMessage(result.message)
        if (!disabled) {
          toast.error(result.message || t('Failed to load audit request logs'))
        }
        return { items: [], total: 0, disabled }
      }
      return {
        items: result.data?.items || [],
        total: result.data?.total || 0,
        disabled: false,
      }
    },
    placeholderData: (previousData) => previousData,
  })

  const logs = data?.items || []

  const { table } = useDataTable({
    data: logs,
    columns,
    pagination,
    onPaginationChange,
    enableRowSelection: false,
    manualPagination: true,
    manualFiltering: true,
    totalCount: data?.total || 0,
    ensurePageInRange,
    getRowId: (row) => `${row.request_id}:${row.created_at}`,
  })

  return (
    <>
      <DataTablePage
        table={table}
        columns={columns}
        isLoading={isLoading}
        isFetching={isFetching}
        emptyTitle={
          data?.disabled
            ? t('Request audit is not enabled')
            : t('No audit request logs')
        }
        emptyDescription={
          data?.disabled
            ? t('Set AUDIT_SQL_DSN to enable request audit logging.')
            : t('No requests match the current filters.')
        }
        skeletonKeyPrefix='audit-request-logs-skeleton'
        applyHeaderSize
        toolbar={<AuditRequestLogsFilterBar />}
        getRowClassName={() => 'cursor-pointer'}
        renderRow={(row) => (
          <DataTableRow
            key={row.id}
            row={row}
            className='hover:bg-muted/40 cursor-pointer'
            onClick={() => setSelected(row.original)}
            onKeyDown={(event) => {
              if (event.key === 'Enter' || event.key === ' ') {
                event.preventDefault()
                setSelected(row.original)
              }
            }}
            tabIndex={0}
            role='button'
            aria-label={t('View request details')}
          />
        )}
      />
      <AuditRequestDetailDialog
        log={selected}
        open={selected != null}
        onOpenChange={(open) => {
          if (!open) setSelected(null)
        }}
      />
    </>
  )
}
