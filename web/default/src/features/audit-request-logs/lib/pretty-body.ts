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
export const AUDIT_BODY_TRUNCATED_REQUEST = 1
export const AUDIT_BODY_TRUNCATED_RESPONSE = 2

export function isAuditRequestBodyTruncated(flags: number): boolean {
  return (flags & AUDIT_BODY_TRUNCATED_REQUEST) !== 0
}

export function isAuditResponseBodyTruncated(flags: number): boolean {
  return (flags & AUDIT_BODY_TRUNCATED_RESPONSE) !== 0
}

// JSON.parse/stringify coerce integers outside Number.MAX_SAFE_INTEGER.
// Keep the stored text when a number token is 16+ digits so the detail
// view matches the backend json.Number-preserving payload.
const unsafeIntegerToken = /(?:^|[^"\d])-?\d{16,}(?:[^"\d.]|$)/

export function formatAuditBody(raw: string): string {
  const trimmed = raw.trim()
  if (!trimmed) {
    return ''
  }
  if (unsafeIntegerToken.test(trimmed)) {
    return raw
  }
  try {
    return JSON.stringify(JSON.parse(trimmed) as unknown, null, 2)
  } catch {
    return raw
  }
}
