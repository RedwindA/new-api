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
import { RefreshIcon } from '@hugeicons/core-free-icons'
import { HugeiconsIcon } from '@hugeicons/react'
import { useQuery } from '@tanstack/react-query'
import { useTranslation } from 'react-i18next'

import { Dialog } from '@/components/dialog'
import {
  Alert,
  AlertAction,
  AlertDescription,
  AlertTitle,
} from '@/components/ui/alert'
import { Button } from '@/components/ui/button'
import { formatTimestampToDate } from '@/lib/format'

import { getAuditRequestLogDetail } from '../api'
import {
  formatAuditBody,
  isAuditRequestBodyTruncated,
  isAuditResponseBodyTruncated,
} from '../lib/pretty-body'
import { auditResultLabelKey } from '../lib/result'
import type { AuditRequestLog } from '../types'

function BodyBlock(props: {
  label: string
  value: string
  truncated: boolean
}) {
  const { t } = useTranslation()
  return (
    <div className='space-y-1.5'>
      <div className='flex items-center gap-2'>
        <p className='text-muted-foreground text-xs font-medium'>
          {props.label}
        </p>
        {props.truncated ? (
          <span className='text-warning text-xs'>{t('Truncated')}</span>
        ) : null}
      </div>
      <pre className='bg-muted/50 max-h-64 overflow-auto rounded-md p-3 font-mono text-xs whitespace-pre-wrap'>
        {props.value || '-'}
      </pre>
    </div>
  )
}

export function AuditRequestDetailDialog(props: {
  log: AuditRequestLog | null
  open: boolean
  onOpenChange: (open: boolean) => void
}) {
  const { t } = useTranslation()
  const requestId = props.log?.request_id ?? ''
  const createdAt = props.log?.created_at ?? 0

  const detailQuery = useQuery({
    queryKey: ['audit-request-log-detail', requestId, createdAt],
    queryFn: async () => {
      const result = await getAuditRequestLogDetail({
        requestId,
        createdAt,
      })
      if (!result.success || !result.data) {
        throw new Error(result.message || t('Failed to load request details'))
      }
      return result.data
    },
    enabled: props.open && requestId !== '',
  })

  const detail = detailQuery.data
  const title = t('Request details')
  const loadError =
    detailQuery.error instanceof Error
      ? detailQuery.error.message
      : t('Failed to load request details')

  let body = (
    <p className='text-muted-foreground text-sm'>{t('Loading details...')}</p>
  )
  if (detailQuery.isError) {
    body = (
      <Alert variant='destructive'>
        <AlertTitle>{t('Failed to load request details')}</AlertTitle>
        <AlertDescription>{loadError}</AlertDescription>
        <AlertAction>
          <Button
            type='button'
            variant='outline'
            size='xs'
            disabled={detailQuery.isFetching}
            onClick={() => {
              void detailQuery.refetch()
            }}
          >
            <HugeiconsIcon
              icon={RefreshIcon}
              strokeWidth={2}
              data-icon='inline-start'
            />
            {t('Retry')}
          </Button>
        </AlertAction>
      </Alert>
    )
  } else if (detail) {
    body = (
      <>
        <dl className='grid grid-cols-1 gap-2 text-sm sm:grid-cols-2'>
          <div>
            <dt className='text-muted-foreground text-xs'>{t('Time')}</dt>
            <dd className='font-mono text-xs'>
              {formatTimestampToDate(detail.created_at, 'seconds')}
            </dd>
          </div>
          <div>
            <dt className='text-muted-foreground text-xs'>{t('IP')}</dt>
            <dd className='font-mono text-xs'>{detail.ip || '-'}</dd>
          </div>
          <div>
            <dt className='text-muted-foreground text-xs'>{t('Method')}</dt>
            <dd className='font-mono text-xs'>{detail.method}</dd>
          </div>
          <div>
            <dt className='text-muted-foreground text-xs'>{t('Status')}</dt>
            <dd className='font-mono text-xs'>{detail.status_code}</dd>
          </div>
          <div>
            <dt className='text-muted-foreground text-xs'>{t('Result')}</dt>
            <dd className='font-mono text-xs'>
              {t(auditResultLabelKey(detail.result))}
            </dd>
          </div>
          <div className='sm:col-span-2'>
            <dt className='text-muted-foreground text-xs'>{t('Path')}</dt>
            <dd className='font-mono text-xs break-all'>{detail.path}</dd>
          </div>
          <div className='sm:col-span-2'>
            <dt className='text-muted-foreground text-xs'>{t('Route')}</dt>
            <dd className='font-mono text-xs break-all'>
              {detail.route || '-'}
            </dd>
          </div>
          <div className='sm:col-span-2'>
            <dt className='text-muted-foreground text-xs'>{t('Query')}</dt>
            <dd className='font-mono text-xs break-all'>
              {detail.query || '-'}
            </dd>
          </div>
          <div className='sm:col-span-2'>
            <dt className='text-muted-foreground text-xs'>{t('User Agent')}</dt>
            <dd className='font-mono text-xs break-all'>
              {detail.user_agent || '-'}
            </dd>
          </div>
        </dl>
        <BodyBlock
          label={t('Request Body')}
          value={formatAuditBody(detail.request_body ?? '')}
          truncated={isAuditRequestBodyTruncated(detail.body_truncated)}
        />
        <BodyBlock
          label={t('Response Body')}
          value={formatAuditBody(detail.response_body ?? '')}
          truncated={isAuditResponseBodyTruncated(detail.body_truncated)}
        />
      </>
    )
  }

  return (
    <Dialog
      open={props.open}
      onOpenChange={props.onOpenChange}
      title={title}
      description={requestId}
      contentClassName='sm:max-w-3xl'
      bodyClassName='space-y-4'
    >
      {body}
    </Dialog>
  )
}
