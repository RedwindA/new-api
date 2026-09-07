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
import assert from 'node:assert/strict'

import { describe, test } from 'vitest'

import {
  AUDIT_REQUEST_DISABLED_MESSAGE,
  buildAuditRequestLogQuery,
  isAuditRequestDisabledMessage,
  statusCodesToSearch,
} from '../lib/query-params'

describe('buildAuditRequestLogQuery', () => {
  test('omits empty optional filters and keeps user_id 0', () => {
    const query = buildAuditRequestLogQuery({
      p: 2,
      page_size: 20,
      ip: '',
      path: '/api/user/login',
      user_id: 0,
      status_code: [401],
    })

    assert.equal(query.get('p'), '2')
    assert.equal(query.get('page_size'), '20')
    assert.equal(query.get('path'), '/api/user/login')
    assert.equal(query.get('user_id'), '0')
    assert.equal(query.get('status_code'), '401')
    assert.equal(query.has('ip'), false)
  })

  test('joins multiple status codes with commas and omits an empty list', () => {
    const query = buildAuditRequestLogQuery({ status_code: [200, 401] })
    assert.equal(query.get('status_code'), '200,401')

    const emptyQuery = buildAuditRequestLogQuery({ status_code: [] })
    assert.equal(emptyQuery.has('status_code'), false)
  })

  test('treats the backend disabled message as the not-enabled state', () => {
    assert.equal(AUDIT_REQUEST_DISABLED_MESSAGE, 'Request audit is not enabled')
    assert.equal(
      isAuditRequestDisabledMessage(AUDIT_REQUEST_DISABLED_MESSAGE),
      true
    )
    assert.equal(
      isAuditRequestDisabledMessage('Failed to load audit request logs'),
      false
    )
    assert.equal(isAuditRequestDisabledMessage(undefined), false)
  })
})

describe('statusCodesToSearch', () => {
  const available = ['200', '401', '500']

  test('returns undefined when every available status code is selected', () => {
    assert.equal(
      statusCodesToSearch(['500', '200', '401'], available),
      undefined
    )
  })

  test('returns undefined when nothing is selected', () => {
    assert.equal(statusCodesToSearch([], available), undefined)
  })

  test('returns the numeric subset when only some codes are selected', () => {
    assert.deepEqual(statusCodesToSearch(['401', '500'], available), [401, 500])
  })

  test('keeps a selection that is not in the available list while options load', () => {
    assert.deepEqual(statusCodesToSearch(['404'], []), [404])
  })
})
