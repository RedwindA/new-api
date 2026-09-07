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
import type { AuditRequestLogQuery } from '../types'

export const AUDIT_REQUEST_DISABLED_MESSAGE = 'Request audit is not enabled'

export function isAuditRequestDisabledMessage(message?: string): boolean {
  return message === AUDIT_REQUEST_DISABLED_MESSAGE
}

export function buildAuditRequestLogQuery(
  params: AuditRequestLogQuery
): URLSearchParams {
  const query = new URLSearchParams()
  for (const [key, value] of Object.entries(params)) {
    if (value === undefined || value === null || value === '') {
      continue
    }
    if (Array.isArray(value)) {
      if (value.length === 0) continue
      query.set(key, value.join(','))
      continue
    }
    query.set(key, String(value))
  }
  return query
}

/**
 * Converts the status-code multi-select into the route search value.
 * Selecting every known code (or nothing) means "no status filter".
 */
export function statusCodesToSearch(
  selected: string[],
  available: string[]
): number[] | undefined {
  const codes = selected
    .map((code) => Number(code))
    .filter((code) => Number.isInteger(code))
  if (codes.length === 0) return undefined
  const coversAll =
    available.length > 0 && available.every((code) => selected.includes(code))
  if (coversAll) return undefined
  return codes
}
