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
export interface AffiliateUserSummary {
  id: number
  username: string
  aff_code: string
  status: number
}

export interface AffiliateUserDetail {
  id: number
  username: string
  aff_code: string
  inviter: AffiliateUserSummary | null
  default_commission_rate: number
  custom_commission_rate: number | null
  effective_commission_rate: number
  aff_quota: number
  aff_history_quota: number
  invite_count: number
}

export type AffiliateCommissionSourceType =
  | 'topup'
  | 'subscription'
  | 'redemption'

export interface AffiliateCommissionRecord {
  id: number
  source_type: AffiliateCommissionSourceType
  source_order_no: string
  payment_provider: string
  base_quota: number
  commission_rate: number
  commission_quota: number
  settled_at: number
}

export interface AffiliateAdminCommissionRecord extends AffiliateCommissionRecord {
  inviter_id: number
  inviter_username: string
  invitee_id: number
  invitee_username: string
  source_id: number
}

export interface AffiliateCommissionPage<
  T extends AffiliateCommissionRecord = AffiliateCommissionRecord,
> {
  items: T[]
  total: number
  page: number
  page_size: number
}
