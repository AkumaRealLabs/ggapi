/*
Copyright (C) 2023-2026 QuantumNous

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU Affero General Public License as
published by the Free Software Foundation, either version 3 of the
License, or (at your option) any later version.
*/
import { ChevronLeft, ChevronRight } from 'lucide-react'
import { type ReactNode, useCallback, useEffect, useState } from 'react'
import { useTranslation } from 'react-i18next'
import { toast } from 'sonner'

import { Button } from '@/components/design-system/button'
import { Dialog } from '@/components/dialog'
import { Skeleton } from '@/components/ui/skeleton'
import { CommissionRecordRow } from '@/features/affiliate/components/commission-record-row'
import type { AffiliateCommissionRecord } from '@/features/affiliate/types'

import { getAffiliateCommissions } from '../../api'

type AffiliateCommissionsDialogProps = {
  open: boolean
  onOpenChange: (open: boolean) => void
}

export function AffiliateCommissionsDialog(
  props: AffiliateCommissionsDialogProps
) {
  const { t } = useTranslation()
  const [records, setRecords] = useState<AffiliateCommissionRecord[]>([])
  const [page, setPage] = useState(1)
  const [total, setTotal] = useState(0)
  const [loading, setLoading] = useState(false)
  const pageSize = 10
  const totalPages = Math.max(1, Math.ceil(total / pageSize))

  const fetchRecords = useCallback(async () => {
    if (!props.open) return
    setLoading(true)
    try {
      const response = await getAffiliateCommissions(page, pageSize)
      if (!response.success || !response.data) {
        toast.error(response.message || t('Failed to load commission records'))
        return
      }
      setRecords(response.data.items ?? [])
      setTotal(response.data.total ?? 0)
    } catch {
      toast.error(t('Failed to load commission records'))
    } finally {
      setLoading(false)
    }
  }, [page, props.open, t])

  useEffect(() => {
    void fetchRecords()
  }, [fetchRecords])

  useEffect(() => {
    if (!props.open) setPage(1)
  }, [props.open])

  let content: ReactNode
  if (loading) {
    content = (
      <div className='space-y-2'>
        {['one', 'two', 'three'].map((key) => (
          <Skeleton key={key} className='h-20 w-full' />
        ))}
      </div>
    )
  } else if (records.length === 0) {
    content = (
      <div className='text-muted-foreground py-10 text-center text-sm'>
        {t('No commission records')}
      </div>
    )
  } else {
    content = (
      <div className='max-h-[min(60vh,560px)] space-y-2 overflow-y-auto pr-1'>
        {records.map((record) => (
          <CommissionRecordRow key={record.id} record={record} />
        ))}
      </div>
    )
  }

  return (
    <Dialog
      open={props.open}
      onOpenChange={props.onOpenChange}
      title={t('Commission Records')}
      description={t('View rewards earned from invited users.')}
      contentClassName='sm:max-w-3xl'
      bodyClassName='space-y-3'
    >
      {content}

      {!loading && total > 0 ? (
        <div className='flex items-center justify-between border-t pt-3'>
          <span className='text-muted-foreground text-xs'>
            {t('{{count}} records', { count: total })}
          </span>
          <div className='flex items-center gap-2'>
            <Button
              variant='outline'
              size='icon'
              onClick={() => setPage((value) => Math.max(1, value - 1))}
              disabled={page <= 1}
              aria-label={t('Previous page')}
            >
              <ChevronLeft />
            </Button>
            <span className='text-sm tabular-nums'>
              {page} / {totalPages}
            </span>
            <Button
              variant='outline'
              size='icon'
              onClick={() =>
                setPage((value) => Math.min(totalPages, value + 1))
              }
              disabled={page >= totalPages}
              aria-label={t('Next page')}
            >
              <ChevronRight />
            </Button>
          </div>
        </div>
      ) : null}
    </Dialog>
  )
}
