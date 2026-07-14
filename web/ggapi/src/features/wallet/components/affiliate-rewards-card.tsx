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
import { History } from 'lucide-react'
import { useState } from 'react'
import { useTranslation } from 'react-i18next'

import { CopyButton } from '@/components/copy-button'
import { Button } from '@/components/design-system/button'
import { Input } from '@/components/design-system/input'
import { Card, CardContent } from '@/components/ui/card'
import { Skeleton } from '@/components/ui/skeleton'
import type { AffiliateUserDetail } from '@/features/affiliate/types'
import { formatQuota } from '@/lib/format'

import type { UserWalletData } from '../types'

interface AffiliateRewardsCardProps {
  user: UserWalletData | null
  affiliate: AffiliateUserDetail | null
  affiliateLink: string
  onTransfer: () => void
  onBind: (affCode: string) => Promise<boolean>
  onViewCommissions: () => void
  binding?: boolean
  complianceConfirmed?: boolean
  loading?: boolean
}

export function AffiliateRewardsCard(props: AffiliateRewardsCardProps) {
  const { t } = useTranslation()
  const [bindCode, setBindCode] = useState('')

  if (props.loading) {
    return (
      <Card data-card-hover='false' className='py-0'>
        <CardContent className='space-y-3 p-4 sm:p-5'>
          <Skeleton className='h-5 w-32' />
          <Skeleton className='h-4 w-full max-w-md' />
          <Skeleton className='h-8 w-full' />
        </CardContent>
      </Card>
    )
  }

  const hasRewards = (props.user?.aff_quota ?? 0) > 0
  const stats = [
    [
      t('Pending'),
      formatQuota(props.affiliate?.aff_quota ?? props.user?.aff_quota ?? 0),
    ],
    [
      t('Total Earned'),
      formatQuota(
        props.affiliate?.aff_history_quota ?? props.user?.aff_history_quota ?? 0
      ),
    ],
    [
      t('Invites'),
      String(props.affiliate?.invite_count ?? props.user?.aff_count ?? 0),
    ],
    [
      t('Commission Rate'),
      `${(props.affiliate?.effective_commission_rate ?? 0).toFixed(2)}%`,
    ],
  ]

  const handleBind = async () => {
    if (!bindCode.trim()) return
    if (await props.onBind(bindCode.trim())) {
      setBindCode('')
    }
  }

  return (
    <Card
      data-card-hover='false'
      data-slot='sketch-wallet-card'
      className='gap-0 py-0'
    >
      <CardContent className='flex flex-col gap-4 p-4 sm:p-5 lg:flex-row lg:items-center lg:justify-between'>
        <div className='min-w-0'>
          <h3 className='text-sm font-medium'>{t('Referral Program')}</h3>
          <p className='text-muted-foreground mt-0.5 line-clamp-2 text-xs'>
            {t(
              'Earn rewards when users join through your referral link. Transfer accumulated rewards to your balance anytime.'
            )}
          </p>
          <div className='text-muted-foreground mt-2 flex flex-wrap items-center gap-x-4 gap-y-1 text-xs'>
            {stats.map(([label, value]) => (
              <span key={label} className='inline-flex items-baseline gap-1.5'>
                {label}
                <span className='text-foreground font-medium tabular-nums'>
                  {value}
                </span>
              </span>
            ))}
          </div>
        </div>

        <div className='flex w-full flex-col gap-2 lg:w-auto lg:max-w-xl lg:flex-1 lg:items-end'>
          <div className='flex w-full items-center gap-2 lg:justify-end'>
            <Input
              value={props.affiliateLink}
              readOnly
              className='min-w-0 flex-1 font-mono text-xs lg:max-w-sm'
            />
            <CopyButton
              value={props.affiliateLink}
              variant='outline'
              className='shrink-0'
              iconClassName='size-4'
              tooltip={t('Copy referral link')}
              aria-label={t('Copy referral link')}
            />
            <Button
              variant='outline'
              size='icon'
              onClick={props.onViewCommissions}
              aria-label={t('Commission Records')}
            >
              <History />
            </Button>
            {hasRewards && (
              <Button
                onClick={props.onTransfer}
                disabled={props.complianceConfirmed === false}
                className='shrink-0'
              >
                {t('Transfer to Balance')}
              </Button>
            )}
          </div>
          {props.affiliate?.inviter ? (
            <div className='text-muted-foreground text-xs'>
              {t('Current Inviter')}: {props.affiliate.inviter.username} (#
              {props.affiliate.inviter.id})
            </div>
          ) : (
            <div className='flex w-full items-center gap-2 lg:max-w-sm'>
              <Input
                value={bindCode}
                onChange={(event) => setBindCode(event.currentTarget.value)}
                placeholder={t('Enter invitation code')}
                maxLength={32}
              />
              <Button
                variant='outline'
                onClick={handleBind}
                disabled={!bindCode.trim() || props.binding}
              >
                {props.binding ? t('Binding...') : t('Bind')}
              </Button>
            </div>
          )}
        </div>
      </CardContent>
      {props.complianceConfirmed === false && (
        <div className='text-muted-foreground border-t px-4 py-2.5 text-xs sm:px-5'>
          {t(
            'Referral reward transfer is disabled until the administrator confirms compliance terms.'
          )}
        </div>
      )}
    </Card>
  )
}
