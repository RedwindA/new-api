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
import { useEffect, useMemo, useState } from 'react'
import { useTranslation } from 'react-i18next'

import { MultiSelect } from '@/components/multi-select'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import {
  Select,
  SelectContent,
  SelectGroup,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select'
import { CompactDateTimeRangePicker } from '@/features/usage-logs/components/compact-date-time-range-picker'

import { getAuditRequestStatusCodes } from '../api'
import { statusCodesToSearch } from '../lib/query-params'
import {
  AUDIT_RESULT_FAILURE,
  AUDIT_RESULT_FILTER_ALL,
  AUDIT_RESULT_SUCCESS,
  resultFilterToSearch,
  resultSearchToFilter,
} from '../lib/result'
import type { AuditRequestLogFilters } from '../types'

const route = getRouteApi('/_authenticated/audit-request-logs/')

const emptyFilters: AuditRequestLogFilters = {
  ip: '',
  path: '',
  method: '',
  statusCodes: [],
  result: AUDIT_RESULT_FILTER_ALL,
  username: '',
  userId: '',
  requestId: '',
}

function filtersFromSearch(
  search: {
    startTime?: number
    endTime?: number
    ip?: string
    path?: string
    method?: string
    statusCodes?: number[]
    result?: number
    username?: string
    userId?: number
    requestId?: string
  },
  availableStatusCodes: string[]
): AuditRequestLogFilters {
  return {
    startTime: search.startTime ? new Date(search.startTime) : undefined,
    endTime: search.endTime ? new Date(search.endTime) : undefined,
    ip: search.ip ?? '',
    path: search.path ?? '',
    method: search.method ?? '',
    statusCodes: search.statusCodes
      ? search.statusCodes.map(String)
      : availableStatusCodes,
    result: resultSearchToFilter(search.result),
    username: search.username ?? '',
    userId: search.userId != null ? String(search.userId) : '',
    requestId: search.requestId ?? '',
  }
}

export function AuditRequestLogsFilterBar() {
  const { t } = useTranslation()
  const search = route.useSearch()
  const navigate = route.useNavigate()
  const statusCodesQuery = useQuery({
    queryKey: ['audit-request-status-codes'],
    queryFn: getAuditRequestStatusCodes,
    staleTime: 60_000,
  })
  const availableStatusCodes = useMemo(
    () => (statusCodesQuery.data?.data ?? []).map(String),
    [statusCodesQuery.data]
  )
  const statusCodeOptions = useMemo(
    () => availableStatusCodes.map((code) => ({ label: code, value: code })),
    [availableStatusCodes]
  )
  const resultOptions = useMemo(
    () => [
      { value: AUDIT_RESULT_FILTER_ALL, label: t('All results') },
      { value: String(AUDIT_RESULT_FAILURE), label: t('Failed') },
      { value: String(AUDIT_RESULT_SUCCESS), label: t('Success') },
    ],
    [t]
  )
  const applied = useMemo(
    () => filtersFromSearch(search, availableStatusCodes),
    [search, availableStatusCodes]
  )
  const [draft, setDraft] = useState<AuditRequestLogFilters>(applied)

  useEffect(() => {
    setDraft(applied)
  }, [applied])
  const hasActiveFilters = Boolean(
    applied.startTime ||
    applied.endTime ||
    applied.ip ||
    applied.path ||
    applied.method ||
    search.statusCodes ||
    search.result != null ||
    applied.username ||
    applied.userId ||
    applied.requestId
  )

  const applyFilters = (next: AuditRequestLogFilters) => {
    const userId = next.userId.trim() ? Number(next.userId) : undefined
    void navigate({
      search: (prev) => ({
        ...prev,
        page: 1,
        startTime: next.startTime ? next.startTime.getTime() : undefined,
        endTime: next.endTime ? next.endTime.getTime() : undefined,
        ip: next.ip.trim() || undefined,
        path: next.path.trim() || undefined,
        method: next.method.trim() || undefined,
        statusCodes: statusCodesToSearch(
          next.statusCodes,
          availableStatusCodes
        ),
        result: resultFilterToSearch(next.result),
        username: next.username.trim() || undefined,
        userId: userId != null && Number.isFinite(userId) ? userId : undefined,
        requestId: next.requestId.trim() || undefined,
      }),
    })
  }

  return (
    <div className='bg-card/50 space-y-2 rounded-lg border p-2.5'>
      <div className='grid grid-cols-1 gap-2 sm:grid-cols-2 lg:grid-cols-4'>
        <CompactDateTimeRangePicker
          start={draft.startTime}
          end={draft.endTime}
          onChange={(range) =>
            setDraft((current) => ({
              ...current,
              startTime: range.start,
              endTime: range.end,
            }))
          }
        />
        <Input
          value={draft.ip}
          onChange={(event) =>
            setDraft((current) => ({ ...current, ip: event.target.value }))
          }
          placeholder={t('IP')}
          aria-label={t('IP')}
          autoComplete='off'
          className='h-8 text-sm'
        />
        <Input
          value={draft.path}
          onChange={(event) =>
            setDraft((current) => ({ ...current, path: event.target.value }))
          }
          placeholder={t('Path')}
          aria-label={t('Path')}
          autoComplete='off'
          className='h-8 text-sm'
        />
        <Input
          value={draft.method}
          onChange={(event) =>
            setDraft((current) => ({ ...current, method: event.target.value }))
          }
          placeholder={t('Method')}
          aria-label={t('Method')}
          autoComplete='off'
          className='h-8 text-sm'
        />
        <MultiSelect
          options={statusCodeOptions}
          selected={draft.statusCodes}
          onChange={(values) =>
            setDraft((current) => ({ ...current, statusCodes: values }))
          }
          placeholder={t('Status')}
          disabled={statusCodesQuery.isLoading}
          className='text-sm'
        />
        <Select
          items={resultOptions}
          value={draft.result}
          onValueChange={(value) =>
            setDraft((current) => ({
              ...current,
              result: value ?? AUDIT_RESULT_FILTER_ALL,
            }))
          }
        >
          <SelectTrigger
            size='sm'
            className='w-full text-sm'
            aria-label={t('Result')}
          >
            <SelectValue />
          </SelectTrigger>
          <SelectContent alignItemWithTrigger={false}>
            <SelectGroup>
              {resultOptions.map((option) => (
                <SelectItem key={option.value} value={option.value}>
                  {option.label}
                </SelectItem>
              ))}
            </SelectGroup>
          </SelectContent>
        </Select>
        <Input
          value={draft.username}
          onChange={(event) =>
            setDraft((current) => ({
              ...current,
              username: event.target.value,
            }))
          }
          placeholder={t('Username')}
          aria-label={t('Username')}
          autoComplete='off'
          className='h-8 text-sm'
        />
        <Input
          value={draft.userId}
          onChange={(event) =>
            setDraft((current) => ({ ...current, userId: event.target.value }))
          }
          placeholder={t('User ID')}
          aria-label={t('User ID')}
          inputMode='numeric'
          autoComplete='off'
          className='h-8 text-sm'
        />
        <Input
          value={draft.requestId}
          onChange={(event) =>
            setDraft((current) => ({
              ...current,
              requestId: event.target.value,
            }))
          }
          placeholder={t('Request ID')}
          aria-label={t('Request ID')}
          autoComplete='off'
          className='h-8 text-sm'
        />
      </div>
      <div className='flex justify-end gap-2'>
        <Button
          type='button'
          variant='ghost'
          disabled={!hasActiveFilters}
          onClick={() => {
            const reset = {
              ...emptyFilters,
              statusCodes: availableStatusCodes,
            }
            setDraft(reset)
            applyFilters(reset)
          }}
        >
          {t('Reset')}
        </Button>
        <Button type='button' onClick={() => applyFilters(draft)}>
          {t('Search')}
        </Button>
      </div>
    </div>
  )
}
