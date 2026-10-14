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

/**
 * What the desktop app (desktop/main.go) injects into its window. Absent in
 * browsers.
 */
export type DesktopApi = {
  /** The directory of the SQLite database the app's server uses. */
  dataDir?: string
  /** Draws the window controls over the header in the page's theme. */
  setTheme?: (theme: 'dark' | 'light' | 'system') => void
}

export function getDesktopApi(): DesktopApi | undefined {
  if (typeof window === 'undefined') return undefined
  return (window as Window & { newApiDesktop?: DesktopApi }).newApiDesktop
}
