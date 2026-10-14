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
 * The title bar area of the desktop app's window, by which pages without a
 * header drag the window. It has no height outside the desktop app. Place it
 * first in a positioned page container.
 */
export function DesktopDragRegion() {
  return (
    <div
      aria-hidden
      className='absolute inset-x-0 top-0 h-[var(--mygo-titlebar-height,0px)] [--app-region:drag]'
    />
  )
}
