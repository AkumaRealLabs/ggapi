/*
Copyright (C) 2023-2026 QuantumNous

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU Affero General Public License as
published by the Free Software Foundation, either version 3 of the
License, or (at your option) any later version.
*/
import { useTranslation } from 'react-i18next'

import { StatusBadge } from '@/components/status-badge'
import { formatQuota, formatTimestamp } from '@/lib/format'

import { getAffiliateCommissionSourceLabelKey } from '../lib/source'
import type { AffiliateCommissionRecord } from '../types'

type CommissionRecordRowProps = {
  record: AffiliateCommissionRecord
  invitee?: string
}

export function CommissionRecordRow(props: CommissionRecordRowProps) {
  const { t } = useTranslation()

  return (
    <div className='rounded-lg border p-3'>
      <div className='flex flex-wrap items-center justify-between gap-2'>
        <div className='min-w-0'>
          <div className='truncate font-mono text-xs'>
            {props.record.source_order_no}
          </div>
          <div className='text-muted-foreground mt-1 text-xs'>
            {formatTimestamp(props.record.settled_at)} ·{' '}
            {props.record.payment_provider}
          </div>
        </div>
        <StatusBadge variant='success'>
          +{formatQuota(props.record.commission_quota)}
        </StatusBadge>
      </div>
      <div className='text-muted-foreground mt-2 flex flex-wrap gap-x-4 gap-y-1 text-xs'>
        <span>
          {t('Source')}:{' '}
          {t(getAffiliateCommissionSourceLabelKey(props.record.source_type))}
        </span>
        <span>
          {t('Commission Base')}: {formatQuota(props.record.base_quota)}
        </span>
        <span>
          {t('Commission Rate')}: {props.record.commission_rate.toFixed(2)}%
        </span>
        {props.invitee ? (
          <span>
            {t('Invitee')}: {props.invitee}
          </span>
        ) : null}
      </div>
    </div>
  )
}
