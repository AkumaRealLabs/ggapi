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
import { formatCurrencyFromUSD } from '@/lib/currency'

import { QUOTA_TYPE_VALUES } from '../constants'
import type { PricingModel, PriceType } from '../types'
import { getConfiguredGroupRatio, getDisplayGroupRatio } from './model-helpers'

// ----------------------------------------------------------------------------
// Price Calculation Utilities
// ----------------------------------------------------------------------------

/**
 * Strip trailing zeros from formatted price string while preserving currency symbols
 */
export function stripTrailingZeros(formatted: string): string {
  // Match currency symbol at start, number, and potential 'k' suffix
  const match = formatted.match(/^([^\d-]*)([-\d,]+\.?\d*)(k?)$/)
  if (!match) return formatted

  const [, symbol, number, suffix] = match

  // Remove commas for processing
  const cleanNumber = number.replaceAll(',', '')

  // Convert to number and back to remove trailing zeros
  const parsed = Number.parseFloat(cleanNumber)
  if (Number.isNaN(parsed)) return formatted

  // Convert to string, which automatically removes trailing zeros
  let result = parsed.toString()

  // If the result is in scientific notation, format it properly
  if (result.includes('e')) {
    result = parsed.toFixed(20).replace(/\.?0+$/, '')
  }

  return `${symbol}${result}${suffix}`
}

/**
 * Calculate token price in USD.
 *
 * Returns NaN when the required ratio field is missing/null so callers can
 * skip rendering that price type.
 */
function calculateTokenPrice(
  model: PricingModel,
  type: PriceType,
  ratio: number
): number {
  const base = model.model_ratio * 2 * ratio

  switch (type) {
    case 'input':
      return base
    case 'output':
      return base * model.completion_ratio
    case 'cache':
      return hasRatio(model.cache_ratio)
        ? base * Number(model.cache_ratio)
        : Number.NaN
    case 'create_cache':
      return hasRatio(model.create_cache_ratio)
        ? base * Number(model.create_cache_ratio)
        : Number.NaN
    case 'image':
      return hasRatio(model.image_ratio)
        ? base * Number(model.image_ratio)
        : Number.NaN
    case 'audio_input':
      return hasRatio(model.audio_ratio)
        ? base * Number(model.audio_ratio)
        : Number.NaN
    case 'audio_output':
      return hasRatio(model.audio_ratio) &&
        hasRatio(model.audio_completion_ratio)
        ? base *
            Number(model.audio_ratio) *
            Number(model.audio_completion_ratio)
        : Number.NaN
  }
}

function hasRatio(value: number | null | undefined): boolean {
  return value !== undefined && value !== null && Number.isFinite(Number(value))
}

/**
 * Apply recharge display rate (Standard vs Recharge toggle).
 *
 * priceRate is how much users recharge (in display currency) for 1 USD credit;
 * usdExchangeRate is the real FX rate. Callers format the result with
 * formatCurrencyFromUSD / formatBillingCurrencyFromUSD.
 *
 * Guards zero/non-finite exchange and priceRate so display never divides by
 * zero or produces NaN from bad rates.
 *
 * Examples:
 * - USD display, priceRate=0.5, usdExchangeRate=1 → 1 × 0.5 / 1 = 0.5
 * - CNY display, priceRate=4, usdExchangeRate=7 → 1 × 4 / 7 ≈ 0.571 → ¥4 after FX
 */
export function applyRechargeRate(
  price: number,
  showWithRecharge: boolean,
  priceRate: number,
  usdExchangeRate: number
): number {
  if (!showWithRecharge) return price
  if (!Number.isFinite(usdExchangeRate) || usdExchangeRate === 0) return price
  if (!Number.isFinite(priceRate)) return price
  return (price * priceRate) / usdExchangeRate
}

/**
 * Whether standard vs recharge would change displayed numbers.
 * Mirrors applyRechargeRate: rate/exchange ≠ 1 (and exchange usable).
 */
export function hasDistinctRechargePrice(
  priceRate: number,
  usdExchangeRate: number
): boolean {
  if (!Number.isFinite(priceRate) || !Number.isFinite(usdExchangeRate)) {
    return false
  }
  if (usdExchangeRate === 0) return false
  return Math.abs(priceRate / usdExchangeRate - 1) > 1e-6
}

/**
 * Format token-based price for display (always per 1M tokens).
 */
export function formatPrice(
  model: PricingModel,
  type: PriceType,
  showWithRecharge = false,
  priceRate = 1,
  usdExchangeRate = 1,
  selectedGroup?: string
): string {
  if (model.quota_type === QUOTA_TYPE_VALUES.REQUEST) {
    return '-'
  }

  const displayGroupRatio = getDisplayGroupRatio(model, selectedGroup)

  let priceInUSD = calculateTokenPrice(model, type, displayGroupRatio)
  priceInUSD = applyRechargeRate(
    priceInUSD,
    showWithRecharge,
    priceRate,
    usdExchangeRate
  )

  return formatCurrencyFromUSD(priceInUSD, {
    digitsLarge: 4,
    digitsSmall: 6,
    abbreviate: false,
  })
}

/**
 * Format price for a specific group (token-based, always per 1M tokens).
 */
export function formatGroupPrice(
  model: PricingModel,
  group: string,
  type: PriceType,
  showWithRecharge = false,
  priceRate = 1,
  usdExchangeRate = 1,
  groupRatio: Record<string, number>
): string {
  if (model.quota_type === QUOTA_TYPE_VALUES.REQUEST) {
    return '-'
  }

  const ratio = getConfiguredGroupRatio(groupRatio, group)
  let priceInUSD = calculateTokenPrice(model, type, ratio)

  priceInUSD = applyRechargeRate(
    priceInUSD,
    showWithRecharge,
    priceRate,
    usdExchangeRate
  )

  return formatCurrencyFromUSD(priceInUSD, {
    digitsLarge: 4,
    digitsSmall: 6,
    abbreviate: false,
  })
}

/**
 * Format fixed price for pay-per-request models (with specific group)
 */
export function formatFixedPrice(
  model: PricingModel,
  group: string,
  showWithRecharge = false,
  priceRate = 1,
  usdExchangeRate = 1,
  groupRatio: Record<string, number>
): string {
  if (model.quota_type !== QUOTA_TYPE_VALUES.REQUEST) {
    return '-'
  }

  const ratio = getConfiguredGroupRatio(groupRatio, group)
  let priceInUSD = (model.model_price || 0) * ratio

  priceInUSD = applyRechargeRate(
    priceInUSD,
    showWithRecharge,
    priceRate,
    usdExchangeRate
  )

  return formatCurrencyFromUSD(priceInUSD, {
    digitsLarge: 4,
    digitsSmall: 4,
    abbreviate: false,
  })
}

/**
 * Format fixed price for pay-per-request models (minimum price from all groups)
 */
export function formatRequestPrice(
  model: PricingModel,
  showWithRecharge = false,
  priceRate = 1,
  usdExchangeRate = 1,
  selectedGroup?: string
): string {
  if (model.quota_type !== QUOTA_TYPE_VALUES.REQUEST) {
    return '-'
  }

  const displayGroupRatio = getDisplayGroupRatio(model, selectedGroup)

  let priceInUSD = (model.model_price || 0) * displayGroupRatio

  priceInUSD = applyRechargeRate(
    priceInUSD,
    showWithRecharge,
    priceRate,
    usdExchangeRate
  )

  return formatCurrencyFromUSD(priceInUSD, {
    digitsLarge: 4,
    digitsSmall: 4,
    abbreviate: false,
  })
}
