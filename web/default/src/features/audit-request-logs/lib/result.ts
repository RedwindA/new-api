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
import type { StatusVariant } from '@/components/status-badge'

/**
 * Business outcome recorded by the backend. new-api answers most business
 * failures (wrong password, invalid params) with HTTP 200 and
 * {"success": false}, so the status code alone cannot tell a failed login
 * from a successful one.
 */
export const AUDIT_RESULT_UNKNOWN = 0
export const AUDIT_RESULT_SUCCESS = 1
export const AUDIT_RESULT_FAILURE = 2

export const AUDIT_RESULT_FILTER_ALL = 'all'

/** i18n key for the result badge; consumers must render it through t(). */
export function auditResultLabelKey(result: number): string {
  if (result === AUDIT_RESULT_SUCCESS) return 'Success'
  if (result === AUDIT_RESULT_FAILURE) return 'Failed'
  return 'Unknown'
}

export function auditResultVariant(result: number): StatusVariant {
  if (result === AUDIT_RESULT_SUCCESS) return 'green'
  if (result === AUDIT_RESULT_FAILURE) return 'red'
  return 'grey'
}

/** Converts the result select value into the route search value. */
export function resultFilterToSearch(value: string): number | undefined {
  if (value === String(AUDIT_RESULT_SUCCESS)) return AUDIT_RESULT_SUCCESS
  if (value === String(AUDIT_RESULT_FAILURE)) return AUDIT_RESULT_FAILURE
  return undefined
}

/** Converts the route search value back into the result select value. */
export function resultSearchToFilter(value: number | undefined): string {
  if (value === AUDIT_RESULT_SUCCESS || value === AUDIT_RESULT_FAILURE) {
    return String(value)
  }
  return AUDIT_RESULT_FILTER_ALL
}
