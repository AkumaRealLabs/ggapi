/*
Copyright (C) 2023-2026 QuantumNous

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU Affero General Public License as
published by the Free Software Foundation, either version 3 of the
License, or (at your option) any later version.
*/
import { ChevronLeft, ChevronRight, Search } from 'lucide-react'
import { type ReactNode, useCallback, useEffect, useState } from 'react'
import { useTranslation } from 'react-i18next'
import { toast } from 'sonner'

import { StatusBadge } from '@/components/status-badge'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import {
  Select,
  SelectContent,
  SelectGroup,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select'
import { Skeleton } from '@/components/ui/skeleton'
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from '@/components/ui/table'
import { getAffiliateCommissionSourceLabelKey } from '@/features/affiliate/lib/source'
import type { AffiliateAdminCommissionRecord } from '@/features/affiliate/types'
import { formatQuota, formatTimestamp } from '@/lib/format'

import { getAffiliateCommissionRecords } from '../api'

type CommissionFilters = {
  orderNo: string
  sourceType: string
  inviter: string
  invitee: string
}

const emptyFilters: CommissionFilters = {
  orderNo: '',
  sourceType: '',
  inviter: '',
  invitee: '',
}

export function AffiliateCommissionTable() {
  const { t } = useTranslation()
  const [draftFilters, setDraftFilters] = useState(emptyFilters)
  const [filters, setFilters] = useState(emptyFilters)
  const [records, setRecords] = useState<AffiliateAdminCommissionRecord[]>([])
  const [page, setPage] = useState(1)
  const [total, setTotal] = useState(0)
  const [loading, setLoading] = useState(false)
  const pageSize = 20
  const totalPages = Math.max(1, Math.ceil(total / pageSize))

  const fetchRecords = useCallback(async () => {
    setLoading(true)
    try {
      const response = await getAffiliateCommissionRecords({
        page,
        pageSize,
        orderNo: filters.orderNo,
        sourceType: filters.sourceType,
        inviter: filters.inviter,
        invitee: filters.invitee,
      })
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
  }, [filters, page, t])

  useEffect(() => {
    void fetchRecords()
  }, [fetchRecords])

  const applyFilters = () => {
    setPage(1)
    setFilters({ ...draftFilters })
  }

  let tableRows: ReactNode
  if (loading) {
    tableRows = ['one', 'two', 'three', 'four'].map((key) => (
      <TableRow key={key}>
        <TableCell colSpan={8}>
          <Skeleton className='h-7 w-full' />
        </TableCell>
      </TableRow>
    ))
  } else if (records.length === 0) {
    tableRows = (
      <TableRow>
        <TableCell colSpan={8} className='h-32 text-center'>
          <span className='text-muted-foreground text-sm'>
            {t('No commission records')}
          </span>
        </TableCell>
      </TableRow>
    )
  } else {
    tableRows = records.map((record) => (
      <TableRow key={record.id}>
        <TableCell>
          <div className='max-w-48 truncate font-mono text-xs'>
            {record.source_order_no}
          </div>
          <div className='text-muted-foreground mt-1 text-xs'>
            {record.payment_provider}
          </div>
        </TableCell>
        <TableCell>
          <StatusBadge variant='neutral'>
            {t(getAffiliateCommissionSourceLabelKey(record.source_type))}
          </StatusBadge>
        </TableCell>
        <TableCell>
          <div className='font-medium'>{record.inviter_username || '-'}</div>
          <div className='text-muted-foreground text-xs'>
            #{record.inviter_id}
          </div>
        </TableCell>
        <TableCell>
          <div className='font-medium'>{record.invitee_username || '-'}</div>
          <div className='text-muted-foreground text-xs'>
            #{record.invitee_id}
          </div>
        </TableCell>
        <TableCell className='tabular-nums'>
          {formatQuota(record.base_quota)}
        </TableCell>
        <TableCell className='tabular-nums'>
          {record.commission_rate.toFixed(2)}%
        </TableCell>
        <TableCell className='font-medium tabular-nums'>
          {formatQuota(record.commission_quota)}
        </TableCell>
        <TableCell className='text-muted-foreground whitespace-nowrap'>
          {formatTimestamp(record.settled_at)}
        </TableCell>
      </TableRow>
    ))
  }

  return (
    <div className='flex min-h-0 flex-1 flex-col gap-3'>
      <div className='grid gap-2 sm:grid-cols-2 lg:grid-cols-[1.4fr_0.8fr_1fr_1fr_auto]'>
        <Input
          value={draftFilters.orderNo}
          onChange={(event) =>
            setDraftFilters((current) => ({
              ...current,
              orderNo: event.currentTarget.value,
            }))
          }
          placeholder={t('Order Number')}
        />
        <Select
          items={[
            { value: 'all', label: t('All Sources') },
            { value: 'topup', label: t('Top-up') },
            { value: 'subscription', label: t('Subscription') },
            { value: 'redemption', label: t('Redemption Code') },
          ]}
          value={draftFilters.sourceType || 'all'}
          onValueChange={(value) =>
            value !== null &&
            setDraftFilters((current) => ({
              ...current,
              sourceType: value === 'all' ? '' : value,
            }))
          }
        >
          <SelectTrigger>
            <SelectValue />
          </SelectTrigger>
          <SelectContent alignItemWithTrigger={false}>
            <SelectGroup>
              <SelectItem value='all'>{t('All Sources')}</SelectItem>
              <SelectItem value='topup'>{t('Top-up')}</SelectItem>
              <SelectItem value='subscription'>{t('Subscription')}</SelectItem>
              <SelectItem value='redemption'>{t('Redemption Code')}</SelectItem>
            </SelectGroup>
          </SelectContent>
        </Select>
        <Input
          value={draftFilters.inviter}
          onChange={(event) =>
            setDraftFilters((current) => ({
              ...current,
              inviter: event.currentTarget.value,
            }))
          }
          placeholder={t('Inviter ID, username, or code')}
        />
        <Input
          value={draftFilters.invitee}
          onChange={(event) =>
            setDraftFilters((current) => ({
              ...current,
              invitee: event.currentTarget.value,
            }))
          }
          placeholder={t('Invitee ID, username, or code')}
        />
        <Button onClick={applyFilters}>
          <Search />
          {t('Search')}
        </Button>
      </div>

      <div className='min-h-0 flex-1 overflow-auto rounded-lg border'>
        <Table>
          <TableHeader>
            <TableRow>
              <TableHead>{t('Order Number')}</TableHead>
              <TableHead>{t('Source')}</TableHead>
              <TableHead>{t('Inviter')}</TableHead>
              <TableHead>{t('Invitee')}</TableHead>
              <TableHead>{t('Commission Base')}</TableHead>
              <TableHead>{t('Commission Rate')}</TableHead>
              <TableHead>{t('Commission')}</TableHead>
              <TableHead>{t('Settled At')}</TableHead>
            </TableRow>
          </TableHeader>
          <TableBody>{tableRows}</TableBody>
        </Table>
      </div>

      <div className='flex items-center justify-between'>
        <span className='text-muted-foreground text-xs'>
          {t('{{count}} records', { count: total })}
        </span>
        <div className='flex items-center gap-2'>
          <Button
            variant='outline'
            size='icon'
            onClick={() => setPage((value) => Math.max(1, value - 1))}
            disabled={page <= 1 || loading}
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
            onClick={() => setPage((value) => Math.min(totalPages, value + 1))}
            disabled={page >= totalPages || loading}
            aria-label={t('Next page')}
          >
            <ChevronRight />
          </Button>
        </div>
      </div>
    </div>
  )
}
