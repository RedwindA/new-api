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
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { cleanup, render, screen, waitFor } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { afterEach, describe, expect, test, vi } from 'vitest'

import type { AuditRequestLog } from '../../types'

const i18n = (await import('i18next')).default
const { I18nextProvider, initReactI18next } = await import('react-i18next')

await i18n.use(initReactI18next).init({
  lng: 'en',
  resources: {
    en: {
      translation: {
        'Request details': 'Request details',
        'Loading details...': 'Loading details...',
        'Failed to load request details': 'Failed to load request details',
        'Request Body': 'Request Body',
        'Response Body': 'Response Body',
        Truncated: 'Truncated',
        Retry: 'Retry',
        Time: 'Time',
        IP: 'IP',
        Method: 'Method',
        Status: 'Status',
        Result: 'Result',
        Failed: 'Failed',
        Path: 'Path',
        Route: 'Route',
        Query: 'Query',
        'User Agent': 'User Agent',
      },
    },
  },
})

const detailLog: AuditRequestLog = {
  created_at: 1700000000,
  request_id: 'req-login-fail',
  ip: '1.2.3.4',
  user_agent: 'curl/8.0',
  method: 'POST',
  route: '/api/user/login',
  path: '/api/user/login',
  query: '',
  status_code: 200,
  result: 2,
  latency_ms: 12,
  user_id: 0,
  username: '',
  user_role: 0,
  request_body: '{"username":"alice","password":"***"}',
  response_body: '{"success":false}',
  body_truncated: 1,
}

const getAuditRequestLogDetail = vi.fn()

vi.mock('../../api', () => ({
  getAuditRequestLogDetail: (...args: unknown[]) =>
    getAuditRequestLogDetail(...args),
}))

const { AuditRequestDetailDialog } =
  await import('../audit-request-detail-dialog')

function renderDialog() {
  const client = new QueryClient({
    defaultOptions: { queries: { retry: false } },
  })
  return render(
    <I18nextProvider i18n={i18n}>
      <QueryClientProvider client={client}>
        <AuditRequestDetailDialog
          log={detailLog}
          open
          onOpenChange={() => undefined}
        />
      </QueryClientProvider>
    </I18nextProvider>
  )
}

describe('AuditRequestDetailDialog', () => {
  afterEach(() => {
    cleanup()
    getAuditRequestLogDetail.mockReset()
  })

  test('shows pretty-printed request body and truncation marker', async () => {
    getAuditRequestLogDetail.mockResolvedValue({
      success: true,
      data: detailLog,
    })
    renderDialog()

    await waitFor(() => {
      expect(screen.getByText('Request Body')).toBeTruthy()
    })
    expect(screen.getByText('Truncated')).toBeTruthy()
    expect(screen.getByText('Result')).toBeTruthy()
    expect(screen.getByText('Failed')).toBeTruthy()
    expect(screen.getByText(/"username": "alice"/)).toBeTruthy()
    expect(screen.getByText(/"password": "\*\*\*"/)).toBeTruthy()
    expect(screen.queryByText('plain-secret')).toBeNull()
  })

  test('shows error and retry when the detail request fails', async () => {
    getAuditRequestLogDetail.mockRejectedValue(new Error('network down'))
    const user = userEvent.setup()
    renderDialog()

    await waitFor(() => {
      expect(screen.getByText('Failed to load request details')).toBeTruthy()
    })
    expect(screen.getByText('network down')).toBeTruthy()
    expect(screen.queryByText('Loading details...')).toBeNull()

    getAuditRequestLogDetail.mockResolvedValue({
      success: true,
      data: detailLog,
    })
    await user.click(screen.getByRole('button', { name: 'Retry' }))
    await waitFor(() => {
      expect(screen.getByText('Request Body')).toBeTruthy()
    })
  })

  test('shows error when the detail API returns success false', async () => {
    getAuditRequestLogDetail.mockResolvedValue({
      success: false,
      message: 'record missing',
    })
    renderDialog()

    await waitFor(() => {
      expect(screen.getByText('Failed to load request details')).toBeTruthy()
    })
    expect(screen.getByText('record missing')).toBeTruthy()
    expect(screen.getByRole('button', { name: 'Retry' })).toBeTruthy()
    expect(screen.queryByText('Loading details...')).toBeNull()
  })
})
