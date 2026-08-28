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
import { describe, expect, test } from 'vitest'

import { parseSidebarConfig } from '@/hooks/use-sidebar-config'

import {
  getVisiblePersonalSidebarSections,
  type PersonalSidebarSectionDef,
} from '../personal-sidebar-sections'

const CATALOG: PersonalSidebarSectionDef[] = [
  {
    key: 'chat',
    title: 'Chat Area',
    description: 'Playground and chat functions',
    modules: [
      {
        key: 'playground',
        title: 'Playground',
        description: 'AI model testing environment',
      },
      {
        key: 'chat',
        title: 'Chat',
        description: 'Chat session management',
      },
    ],
  },
  {
    key: 'console',
    title: 'Console Area',
    description: 'Data management and log viewing',
    modules: [
      {
        key: 'detail',
        title: 'Dashboard',
        description: 'System data statistics',
      },
      {
        key: 'token',
        title: 'Token Management',
        description: 'API token management',
      },
      { key: 'log', title: 'Usage Logs', description: 'API usage records' },
      {
        key: 'midjourney',
        title: 'Drawing Logs',
        description: 'Drawing task records',
      },
      { key: 'task', title: 'Task Logs', description: 'System task records' },
    ],
  },
  {
    key: 'personal',
    title: 'Personal Center Area',
    description: 'User personal functions',
    modules: [
      {
        key: 'topup',
        title: 'Wallet Management',
        description: 'Balance and top-up management',
      },
      {
        key: 'personal',
        title: 'Personal Settings',
        description: 'Personal info settings',
      },
    ],
  },
]

function sectionKeys(sections: PersonalSidebarSectionDef[]): string[] {
  return sections.map((section) => section.key)
}

function moduleKeys(
  sections: PersonalSidebarSectionDef[],
  sectionKey: string
): string[] {
  return (
    sections
      .find((section) => section.key === sectionKey)
      ?.modules.map((module) => module.key) ?? []
  )
}

describe('getVisiblePersonalSidebarSections', () => {
  test('omits chat when admin disables the chat section', () => {
    const adminConfig = parseSidebarConfig(
      JSON.stringify({ chat: { enabled: false } })
    )

    const visible = getVisiblePersonalSidebarSections(CATALOG, adminConfig)

    expect(sectionKeys(visible)).toEqual(['console', 'personal'])
  })

  test('keeps the chat section with only the chat module when playground is disabled', () => {
    const adminConfig = parseSidebarConfig(
      JSON.stringify({
        chat: { enabled: true, playground: false, chat: true },
      })
    )

    const visible = getVisiblePersonalSidebarSections(CATALOG, adminConfig)

    expect(sectionKeys(visible)).toEqual(['chat', 'console', 'personal'])
    expect(moduleKeys(visible, 'chat')).toEqual(['chat'])
  })

  test('omits chat when the section is enabled but every listed module is disabled', () => {
    const adminConfig = parseSidebarConfig(
      JSON.stringify({
        chat: { enabled: true, playground: false, chat: false },
      })
    )

    const visible = getVisiblePersonalSidebarSections(CATALOG, adminConfig)

    expect(sectionKeys(visible)).toEqual(['console', 'personal'])
  })

  test('returns the full catalog when admin config is empty or default', () => {
    const visible = getVisiblePersonalSidebarSections(
      CATALOG,
      parseSidebarConfig('')
    )

    expect(sectionKeys(visible)).toEqual(['chat', 'console', 'personal'])
    expect(moduleKeys(visible, 'chat')).toEqual(['playground', 'chat'])
    expect(moduleKeys(visible, 'console')).toEqual([
      'detail',
      'token',
      'log',
      'midjourney',
      'task',
    ])
    expect(moduleKeys(visible, 'personal')).toEqual(['topup', 'personal'])
  })

  test('returns an empty list when chat, console, and personal are all disabled', () => {
    const adminConfig = parseSidebarConfig(
      JSON.stringify({
        chat: { enabled: false },
        console: { enabled: false },
        personal: { enabled: false },
      })
    )

    const visible = getVisiblePersonalSidebarSections(CATALOG, adminConfig)

    expect(visible).toEqual([])
  })

  test('does not mutate the input sectionDefs', () => {
    const catalogCopy: PersonalSidebarSectionDef[] = CATALOG.map((section) => ({
      ...section,
      modules: section.modules.map((module) => ({ ...module })),
    }))
    const chatModules = catalogCopy[0].modules
    const snapshot = structuredClone(catalogCopy)

    getVisiblePersonalSidebarSections(
      catalogCopy,
      parseSidebarConfig(
        JSON.stringify({
          chat: { enabled: true, playground: false, chat: true },
        })
      )
    )

    expect(catalogCopy).toEqual(snapshot)
    expect(catalogCopy[0].modules).toBe(chatModules)
    expect(catalogCopy[0].modules).toHaveLength(2)
  })
})
