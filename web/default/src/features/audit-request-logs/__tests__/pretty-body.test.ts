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
  formatAuditBody,
  isAuditRequestBodyTruncated,
  isAuditResponseBodyTruncated,
} from '../lib/pretty-body'

describe('formatAuditBody', () => {
  test('pretty-prints JSON objects for the detail dialog', () => {
    assert.equal(
      formatAuditBody('{"username":"alice","password":"***"}'),
      '{\n  "username": "alice",\n  "password": "***"\n}'
    )
  })

  test('keeps unparseable placeholders as stored text', () => {
    const raw = '<unparseable body omitted, 27 bytes>'
    assert.equal(formatAuditBody(raw), raw)
  })

  test('returns empty string for blank bodies', () => {
    assert.equal(formatAuditBody(''), '')
    assert.equal(formatAuditBody('   '), '')
  })

  test('keeps stored text when a JSON integer is outside JS safe range', () => {
    const raw = '{"quota":9007199254740993}'
    assert.equal(formatAuditBody(raw), raw)
  })
})

describe('audit body truncation flags', () => {
  test('reads request and response bits independently', () => {
    assert.equal(isAuditRequestBodyTruncated(0), false)
    assert.equal(isAuditResponseBodyTruncated(0), false)
    assert.equal(isAuditRequestBodyTruncated(1), true)
    assert.equal(isAuditResponseBodyTruncated(1), false)
    assert.equal(isAuditRequestBodyTruncated(2), false)
    assert.equal(isAuditResponseBodyTruncated(2), true)
    assert.equal(isAuditRequestBodyTruncated(3), true)
    assert.equal(isAuditResponseBodyTruncated(3), true)
  })
})
