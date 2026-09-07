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
  AUDIT_RESULT_FAILURE,
  AUDIT_RESULT_FILTER_ALL,
  AUDIT_RESULT_SUCCESS,
  AUDIT_RESULT_UNKNOWN,
  auditResultLabelKey,
  auditResultVariant,
  resultFilterToSearch,
  resultSearchToFilter,
} from '../lib/result'

describe('auditResultLabelKey', () => {
  test('maps success and failure to their i18n keys and everything else to Unknown', () => {
    assert.equal(auditResultLabelKey(AUDIT_RESULT_SUCCESS), 'Success')
    assert.equal(auditResultLabelKey(AUDIT_RESULT_FAILURE), 'Failed')
    assert.equal(auditResultLabelKey(AUDIT_RESULT_UNKNOWN), 'Unknown')
    assert.equal(auditResultLabelKey(99), 'Unknown')
  })
})

describe('auditResultVariant', () => {
  test('renders failure red, success green and unknown grey', () => {
    assert.equal(auditResultVariant(AUDIT_RESULT_FAILURE), 'red')
    assert.equal(auditResultVariant(AUDIT_RESULT_SUCCESS), 'green')
    assert.equal(auditResultVariant(AUDIT_RESULT_UNKNOWN), 'grey')
  })
})

describe('result filter round trip', () => {
  test('all and unknown values mean no result filter', () => {
    assert.equal(resultFilterToSearch(AUDIT_RESULT_FILTER_ALL), undefined)
    assert.equal(resultFilterToSearch(''), undefined)
    assert.equal(resultFilterToSearch('7'), undefined)
    assert.equal(resultSearchToFilter(undefined), AUDIT_RESULT_FILTER_ALL)
    assert.equal(
      resultSearchToFilter(AUDIT_RESULT_UNKNOWN),
      AUDIT_RESULT_FILTER_ALL
    )
  })

  test('success and failure survive the search round trip', () => {
    assert.equal(resultFilterToSearch('1'), AUDIT_RESULT_SUCCESS)
    assert.equal(resultFilterToSearch('2'), AUDIT_RESULT_FAILURE)
    assert.equal(resultSearchToFilter(AUDIT_RESULT_SUCCESS), '1')
    assert.equal(resultSearchToFilter(AUDIT_RESULT_FAILURE), '2')
  })
})
