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
*/
import assert from 'node:assert/strict'
import { describe, test } from 'node:test'

import { getAffiliateCommissionSourceLabelKey } from '../source.ts'

describe('affiliate commission source labels', () => {
  test('maps every supported commission source to its translation key', () => {
    assert.equal(getAffiliateCommissionSourceLabelKey('topup'), 'Top-up')
    assert.equal(
      getAffiliateCommissionSourceLabelKey('subscription'),
      'Subscription'
    )
    assert.equal(
      getAffiliateCommissionSourceLabelKey('redemption'),
      'Redemption Code'
    )
  })

  test('does not present an unknown source as a subscription', () => {
    assert.equal(
      getAffiliateCommissionSourceLabelKey('future-source'),
      'Unknown'
    )
  })
})
