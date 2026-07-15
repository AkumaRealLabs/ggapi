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
import {
  afterAll,
  beforeAll,
  beforeEach,
  describe,
  expect,
  mock,
  test,
} from 'bun:test'
import fs from 'node:fs/promises'
import path from 'node:path'

import type { ReactNode } from 'react'
import { renderToStaticMarkup } from 'react-dom/server'

mock.module('react-i18next', () => ({
  useTranslation: () => ({
    t: (key: string, values?: Record<string, unknown>) =>
      Object.entries(values || {}).reduce(
        (message, [name, value]) =>
          message.replaceAll(`{{${name}}}`, String(value)),
        key
      ),
  }),
}))

mock.module('@/components/dialog', () => ({
  Dialog: ({ children }: { children: ReactNode }) => <div>{children}</div>,
}))

const apiPost = mock(
  async (_url: string, _data: unknown, _config?: unknown) => ({
    data: { success: false, message: 'test-error' },
  })
)
const apiPut = mock(
  async (_url: string, _data: unknown, _config?: unknown) => ({
    data: { success: false, message: 'test-error' },
  })
)
const apiPatch = mock(
  async (_url: string, _data: unknown, _config?: unknown) => ({
    data: { success: false, message: 'test-error' },
  })
)
const apiDelete = mock(async (_url: string, _config?: unknown) => ({
  data: { success: false, message: 'test-error' },
}))

mock.module('@/lib/api', () => ({
  api: {
    post: apiPost,
    put: apiPut,
    patch: apiPatch,
    delete: apiDelete,
  },
}))

const { useSystemConfigStore } =
  await import('../src/stores/system-config-store')
const { formatCurrencyFromUSD, formatLocalCurrencyAmount } =
  await import('../src/lib/currency')
const { getBalancePurchaseSuccessKey } =
  await import('../src/features/subscriptions/lib')
const {
  paySubscriptionStripe,
  paySubscriptionCreem,
  paySubscriptionWaffoPancake,
  paySubscriptionBalance,
  paySubscriptionEpay,
  createPlan,
  updatePlan,
  patchPlanStatus,
  createUserSubscription,
  invalidateUserSubscription,
  deleteUserSubscription,
  resetUserSubscriptionsByPlan,
  resetPlanSubscriptions,
} = await import('../src/features/subscriptions/api')
const { SubscriptionPurchaseDialog } =
  await import('../src/features/subscriptions/components/dialogs/subscription-purchase-dialog')

const originalSystemConfigState = useSystemConfigStore.getState()

beforeAll(() => {
  useSystemConfigStore.setState({
    config: {
      ...originalSystemConfigState.config,
      currency: {
        displayInCurrency: true,
        quotaDisplayType: 'CUSTOM',
        quotaPerUnit: 1,
        usdExchangeRate: 1,
        customCurrencySymbol: '¥',
        customCurrencyExchangeRate: 1,
      },
    },
  })
})

afterAll(() => {
  useSystemConfigStore.setState(originalSystemConfigState)
})

beforeEach(() => {
  apiPost.mockClear()
  apiPut.mockClear()
  apiPatch.mockClear()
  apiDelete.mockClear()
})

describe('subscription purchase dialog', () => {
  test('renders the plan amount with the configured local currency symbol', () => {
    const markup = renderToStaticMarkup(
      <SubscriptionPurchaseDialog
        open
        onOpenChange={() => undefined}
        plan={{
          plan: {
            id: 1,
            title: 'test-plan',
            subtitle: '',
            price_amount: 19.9,
            currency: 'CNY',
            duration_unit: 'month',
            duration_value: 1,
            quota_reset_period: 'never',
            enabled: true,
            sort_order: 0,
            allow_balance_pay: true,
            allow_wallet_overflow: true,
            membership_only: true,
            max_purchase_per_user: 0,
            total_amount: 0,
            upgrade_group: 'test-member',
          },
        }}
      />
    )

    expect(markup).toContain('¥19.90')
    expect(markup).not.toContain('$19.90')
  })

  test('explains renewal queueing for active members', () => {
    const markup = renderToStaticMarkup(
      <SubscriptionPurchaseDialog
        open
        onOpenChange={() => undefined}
        hasActiveMembership
        plan={{
          plan: {
            id: 2,
            title: 'renewal-plan',
            price_amount: 19.9,
            currency: 'CNY',
            duration_unit: 'month',
            duration_value: 1,
            quota_reset_period: 'never',
            enabled: true,
            sort_order: 0,
            allow_balance_pay: true,
            allow_wallet_overflow: true,
            membership_only: true,
            max_purchase_per_user: 0,
            total_amount: 0,
            upgrade_group: 'test-member',
          },
        }}
      />
    )

    expect(markup).toContain(
      'This membership will be queued and activated after your existing membership benefits end.'
    )
  })

  test('uses the membership queue limit returned by the API', () => {
    const markup = renderToStaticMarkup(
      <SubscriptionPurchaseDialog
        open
        onOpenChange={() => undefined}
        scheduledMembershipCount={2}
        membershipQueueLimit={2}
        plan={{
          plan: {
            id: 3,
            title: 'full-queue-plan',
            price_amount: 19.9,
            currency: 'CNY',
            duration_unit: 'month',
            duration_value: 1,
            quota_reset_period: 'never',
            enabled: true,
            sort_order: 0,
            allow_balance_pay: true,
            allow_wallet_overflow: true,
            membership_only: true,
            max_purchase_per_user: 0,
            total_amount: 0,
            upgrade_group: 'test-member',
          },
        }}
      />
    )

    expect(markup).toContain(
      'Membership queue is full. You can have up to 2 queued memberships.'
    )
  })

  test('blocks another membership purchase while an order is pending', () => {
    const markup = renderToStaticMarkup(
      <SubscriptionPurchaseDialog
        open
        onOpenChange={() => undefined}
        membershipQueueLimit={3}
        pendingMembershipOrderCount={1}
        plan={{
          plan: {
            id: 4,
            title: 'pending-order-plan',
            price_amount: 19.9,
            currency: 'CNY',
            duration_unit: 'month',
            duration_value: 1,
            quota_reset_period: 'never',
            enabled: true,
            sort_order: 0,
            allow_balance_pay: true,
            allow_wallet_overflow: true,
            membership_only: true,
            max_purchase_per_user: 0,
            total_amount: 0,
            upgrade_group: 'test-member',
          },
        }}
      />
    )

    expect(markup).toContain('A pending membership order already exists.')
    expect(markup).toContain('disabled')
  })
})

describe('balance purchase success messages', () => {
  test('distinguishes activation from queued renewal', () => {
    expect(getBalancePurchaseSuccessKey('active')).toBe(
      'Subscription activated.'
    )
    expect(getBalancePurchaseSuccessKey('scheduled')).toBe('Membership queued.')
  })
})

describe('subscription purchase formatting', () => {
  test('keeps payment precision without forcing it on other custom currency displays', () => {
    expect(formatLocalCurrencyAmount(19.9)).toBe('¥19.90')
    expect(formatCurrencyFromUSD(19.9)).toBe('¥19.9')
  })
})

describe('subscription payment requests', () => {
  test('leave business error toasts to the purchase dialog', async () => {
    await paySubscriptionStripe({ plan_id: 1 })
    await paySubscriptionCreem({ plan_id: 1 })
    await paySubscriptionWaffoPancake({ plan_id: 1 })
    await paySubscriptionBalance({ plan_id: 1 })
    await paySubscriptionEpay({ plan_id: 1, payment_method: 'alipay' })

    expect(apiPost).toHaveBeenCalledTimes(5)
    for (const call of apiPost.mock.calls) {
      expect(call[2]).toEqual({ skipBusinessError: true })
    }
  })
})

describe('subscription admin requests', () => {
  test('leave business error toasts to the admin dialogs', async () => {
    await createPlan({ plan: { title: 'test' } })
    await updatePlan(1, { plan: { title: 'test' } })
    await patchPlanStatus(1, true)
    await createUserSubscription(1, { plan_id: 1 })
    await invalidateUserSubscription(1)
    await deleteUserSubscription(1)
    await resetUserSubscriptionsByPlan(1, {
      plan_id: 1,
      advance_reset_time: true,
    })
    await resetPlanSubscriptions(1, { advance_reset_time: true })

    for (const call of apiPost.mock.calls) {
      expect(call[2]).toEqual({ skipBusinessError: true })
    }
    expect(apiPut.mock.calls[0][2]).toEqual({ skipBusinessError: true })
    expect(apiPatch.mock.calls[0][2]).toEqual({ skipBusinessError: true })
    expect(apiDelete.mock.calls[0][1]).toEqual({ skipBusinessError: true })
  })
})

describe('subscription purchase errors', () => {
  test('are translated in all supported locales', async () => {
    const locales = ['en', 'zh', 'zh-TW', 'fr', 'ja', 'ru', 'vi']
    const keys = [
      'A pending order already exists for this plan.',
      'A pending membership order already exists.',
      'Custom reset period must be greater than 0 seconds.',
      'Downgrade group does not exist.',
      'Failed to read payment request.',
      'Invalid ID.',
      'Invalid parameters',
      'Invalid quota unit configuration.',
      'Invalid subscription ID.',
      'Invalid user ID.',
      'Insufficient balance',
      'Membership queue is full.',
      'Membership queue is full. You can have up to {{count}} queued memberships.',
      'Membership queued.',
      'Membership plans do not support quota resets.',
      'Membership plans require an upgrade group.',
      'Payment callback configuration is invalid.',
      'Payment method is unavailable.',
      'Payment request failed',
      'Purchase limit cannot be negative.',
      'Purchase limit reached',
      'Set all API keys to auto or a specific group before purchasing membership.',
      'Subscription activated.',
      'Subscription plan is disabled.',
      'Subscription plan amount is too low.',
      'Subscription plan price cannot be negative.',
      'Subscription plan price cannot exceed 9999.',
      'Subscription plan price exceeds the supported range.',
      'Subscription plan not found.',
      'Subscription plan title is required.',
      'The membership plan type cannot be changed while active or queued subscriptions or pending orders exist.',
      'The selected payment provider is not configured correctly.',
      'The user has no active subscription for this plan.',
      'This membership will be queued and activated after your existing membership benefits end.',
      'This plan does not allow balance redemption',
      'This plan is not configured for the selected payment provider.',
      'Total quota cannot be negative.',
      'Upgrade group does not exist.',
      'User not found.',
      'User group switched to {{group}}.',
    ]

    for (const locale of locales) {
      const filePath = path.resolve('src/i18n/locales', `${locale}.json`)
      const json = JSON.parse(await fs.readFile(filePath, 'utf8')) as {
        translation: Record<string, string>
      }

      for (const key of keys) {
        expect(json.translation[key], `${locale}: ${key}`).toBeTruthy()
      }
    }
  })
})
