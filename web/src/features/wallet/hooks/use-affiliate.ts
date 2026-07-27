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
import i18next from 'i18next'
import { useState, useEffect, useCallback } from 'react'
import { toast } from 'sonner'

import type { AffiliateUserDetail } from '@/features/affiliate/types'
import { useCopyToClipboard } from '@/hooks/use-copy-to-clipboard'

import {
  bindAffiliateInviter,
  getAffiliateSelf,
  transferAffiliateQuota,
} from '../api'
import { generateAffiliateLink } from '../lib'

// ============================================================================
// Affiliate Hook
// ============================================================================

export function useAffiliate() {
  const [affiliateCode, setAffiliateCode] = useState<string>('')
  const [affiliateLink, setAffiliateLink] = useState<string>('')
  const [affiliate, setAffiliate] = useState<AffiliateUserDetail | null>(null)
  const [loading, setLoading] = useState(true)
  const [transferring, setTransferring] = useState(false)
  const [binding, setBinding] = useState(false)
  const { copyToClipboard } = useCopyToClipboard()

  // Fetch affiliate code
  const fetchAffiliateCode = useCallback(async () => {
    try {
      setLoading(true)
      const response = await getAffiliateSelf()

      if (response.success && response.data) {
        setAffiliate(response.data)
        setAffiliateCode(response.data.aff_code)
        const link = generateAffiliateLink(response.data.aff_code)
        setAffiliateLink(link)
      }
    } catch (error) {
      // eslint-disable-next-line no-console
      console.error('Failed to fetch affiliate code:', error)
    } finally {
      setLoading(false)
    }
  }, [])

  const bindInviter = useCallback(
    async (affCode: string): Promise<boolean> => {
      try {
        setBinding(true)
        const response = await bindAffiliateInviter(affCode)
        if (!response.success) {
          toast.error(response.message || i18next.t('Failed to bind inviter'))
          return false
        }
        toast.success(i18next.t('Inviter bound successfully'))
        await fetchAffiliateCode()
        return true
      } catch {
        toast.error(i18next.t('Failed to bind inviter'))
        return false
      } finally {
        setBinding(false)
      }
    },
    [fetchAffiliateCode]
  )

  // Copy affiliate link
  const copyAffiliateLink = useCallback(() => {
    copyToClipboard(affiliateLink)
  }, [affiliateLink, copyToClipboard])

  // Transfer affiliate quota to balance
  const transferQuota = useCallback(
    async (quota: number): Promise<boolean> => {
      try {
        setTransferring(true)
        const response = await transferAffiliateQuota({ quota })

        if (response.success) {
          toast.success(response.message || i18next.t('Transfer successful'))
          await fetchAffiliateCode()
          return true
        }

        toast.error(response.message || i18next.t('Transfer failed'))
        return false
      } catch {
        toast.error(i18next.t('Transfer failed'))
        return false
      } finally {
        setTransferring(false)
      }
    },
    [fetchAffiliateCode]
  )

  useEffect(() => {
    fetchAffiliateCode()
  }, [fetchAffiliateCode])

  return {
    affiliateCode,
    affiliateLink,
    affiliate,
    loading,
    transferring,
    binding,
    copyAffiliateLink,
    transferQuota,
    bindInviter,
    refetch: fetchAffiliateCode,
  }
}
