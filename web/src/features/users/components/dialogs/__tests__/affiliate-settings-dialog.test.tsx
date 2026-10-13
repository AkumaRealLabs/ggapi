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
import { cleanup, render, screen, waitFor } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { AxiosError, type AxiosResponse } from 'axios'
import i18n from 'i18next'
import { Toaster } from 'sonner'
import {
  afterAll,
  afterEach,
  beforeAll,
  beforeEach,
  describe,
  expect,
  test,
  vi,
} from 'vitest'

import zhCN from '@/i18n/locales/zh.json'
import { api } from '@/lib/api'

import type { User } from '../../../types'
import { AffiliateSettingsDialog } from '../affiliate-settings-dialog'

const user = { id: 7, username: 'invitee' } as User

const detail = {
  id: 7,
  username: 'invitee',
  aff_code: 'invitee7',
  inviter: null,
  default_commission_rate: 10,
  custom_commission_rate: null,
  effective_commission_rate: 10,
  aff_quota: 0,
  aff_history_quota: 0,
  invite_count: 0,
}

const originalGetAnimations = Object.getOwnPropertyDescriptor(
  HTMLElement.prototype,
  'getAnimations'
)

function mockDialogLoad(loadDetail: () => Promise<unknown>) {
  vi.spyOn(api, 'get').mockImplementation((async (url: string) => {
    if (url === '/api/affiliate/users/7') return loadDetail()
    if (url.startsWith('/api/affiliate/commissions?')) {
      return { data: { success: true, data: { items: [], total: 0 } } }
    }
    throw new Error(`Unexpected GET ${url}`)
  }) as typeof api.get)
}

function renderDialog() {
  render(
    <>
      <Toaster />
      <AffiliateSettingsDialog
        open
        onOpenChange={() => {}}
        user={user}
        onSuccess={() => {}}
      />
    </>
  )
}

beforeAll(() => {
  Object.defineProperty(HTMLElement.prototype, 'getAnimations', {
    configurable: true,
    value: () => [],
  })
})

afterAll(() => {
  if (originalGetAnimations) {
    Object.defineProperty(
      HTMLElement.prototype,
      'getAnimations',
      originalGetAnimations
    )
    return
  }
  Reflect.deleteProperty(HTMLElement.prototype, 'getAnimations')
})

beforeEach(async () => {
  i18n.addResourceBundle('zhCN', 'translation', zhCN.translation, true, true)
  await i18n.changeLanguage('zhCN')
})

afterEach(async () => {
  cleanup()
  vi.restoreAllMocks()
  await i18n.changeLanguage('en')
})

describe('AffiliateSettingsDialog server errors', () => {
  test('shows the translated server reason when saving is rejected', async () => {
    mockDialogLoad(async () => ({ data: { success: true, data: detail } }))
    vi.spyOn(api, 'put').mockResolvedValue({
      data: {
        success: false,
        message: 'Invitation code is already in use',
        message_key: 'Invitation code is already in use',
      },
    })
    renderDialog()

    const save = await screen.findByRole('button', { name: '保存' })
    await waitFor(() => expect(save).toBeEnabled())
    await userEvent.click(save)

    expect(await screen.findByText('邀请码已被使用')).toBeInTheDocument()
  })

  test('shows the translated server reason when loading fails with an HTTP error', async () => {
    const response = {
      status: 400,
      data: {
        success: false,
        message: 'Invalid user ID',
        message_key: 'Invalid user ID',
      },
    } as AxiosResponse
    mockDialogLoad(() =>
      Promise.reject(
        new AxiosError(
          'Request failed with status code 400',
          'ERR_BAD_REQUEST',
          undefined,
          undefined,
          response
        )
      )
    )
    renderDialog()

    expect(await screen.findByText('无效的用户 ID')).toBeInTheDocument()
  })
})
