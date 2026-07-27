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
import { History, Share2 } from 'lucide-react'
import { useState } from 'react'
import { useTranslation } from 'react-i18next'

import { CopyButton } from '@/components/copy-button'
import { Button } from '@/components/ui/button'
import { Card, CardContent } from '@/components/ui/card'
import { IconBadge } from '@/components/ui/icon-badge'
import { Input } from '@/components/ui/input'
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
      <Card data-card-hover='false' className='bg-muted/20 py-0'>
        <CardContent className='grid gap-4 p-3 sm:p-4 lg:grid-cols-[minmax(220px,1fr)_minmax(220px,0.72fr)_minmax(320px,1.15fr)] lg:items-center'>
          <div>
            <Skeleton className='h-5 w-32' />
            <Skeleton className='mt-2 h-4 w-48' />
          </div>
          <Skeleton className='h-14 rounded-lg' />
          <Skeleton className='h-10 rounded-lg' />
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
    <Card data-card-hover='false' className='bg-muted/20 py-0'>
      <CardContent className='grid gap-3 p-3 sm:gap-4 sm:p-4 lg:grid-cols-[minmax(200px,1fr)_minmax(180px,0.65fr)_minmax(280px,1fr)] lg:items-center'>
        <div className='flex min-w-0 items-center gap-2.5'>
          <IconBadge tone='chart-3'>
            <Share2 />
          </IconBadge>
          <div className='min-w-0'>
            <h3 className='truncate text-sm font-semibold'>
              {t('Referral Program')}
            </h3>
            <p className='text-muted-foreground line-clamp-1 text-xs'>
              {t(
                'Earn rewards when users join through your referral link. Transfer accumulated rewards to your balance anytime.'
              )}
            </p>
          </div>
        </div>

        <div className='grid grid-cols-2 gap-1.5 text-center sm:grid-cols-4 lg:grid-cols-2'>
          {stats.map(([label, value]) => (
            <div key={label}>
              <div className='text-muted-foreground truncate text-[10px] font-medium tracking-wider uppercase'>
                {label}
              </div>
              <div className='mt-0.5 truncate text-sm font-semibold tabular-nums'>
                {value}
              </div>
            </div>
          ))}
        </div>

        <div className='flex min-w-0 flex-col gap-2'>
          <div className='flex items-center gap-2'>
            <Input
              value={props.affiliateLink}
              readOnly
              className='border-muted bg-background/70 h-9 min-w-0 flex-1 font-mono text-xs'
            />
            <CopyButton
              value={props.affiliateLink}
              variant='outline'
              className='bg-background size-9 shrink-0'
              iconClassName='size-4'
              tooltip={t('Copy referral link')}
              aria-label={t('Copy referral link')}
            />
            <Button
              variant='outline'
              size='icon'
              className='bg-background size-9 shrink-0'
              onClick={props.onViewCommissions}
              aria-label={t('Commission Records')}
            >
              <History />
            </Button>
            {hasRewards && (
              <Button
                onClick={props.onTransfer}
                disabled={props.complianceConfirmed === false}
                className='h-9 shrink-0 px-3'
                size='sm'
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
        {props.complianceConfirmed === false ? (
          <p className='text-muted-foreground text-xs lg:col-span-3'>
            {t(
              'Referral reward transfer is disabled until the administrator confirms compliance terms.'
            )}
          </p>
        ) : null}
      </CardContent>
    </Card>
  )
}
