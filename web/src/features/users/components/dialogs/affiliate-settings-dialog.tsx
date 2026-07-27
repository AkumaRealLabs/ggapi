/*
Copyright (C) 2023-2026 QuantumNous

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU Affero General Public License as
published by the Free Software Foundation, either version 3 of the
License, or (at your option) any later version.
*/
import { Search } from 'lucide-react'
import { useCallback, useEffect, useMemo, useState } from 'react'
import { useTranslation } from 'react-i18next'
import { toast } from 'sonner'

import { ConfirmDialog } from '@/components/confirm-dialog'
import { Dialog } from '@/components/dialog'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import {
  Select,
  SelectContent,
  SelectGroup,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select'
import { Skeleton } from '@/components/ui/skeleton'
import { CommissionRecordRow } from '@/features/affiliate/components/commission-record-row'
import type {
  AffiliateAdminCommissionRecord,
  AffiliateUserDetail,
  AffiliateUserSummary,
} from '@/features/affiliate/types'
import { formatQuota } from '@/lib/format'

import {
  getAffiliateCommissionRecords,
  getAffiliateUser,
  searchAffiliateUsers,
  updateAffiliateUser,
} from '../../api'
import type { User } from '../../types'

type AffiliateSettingsDialogProps = {
  open: boolean
  onOpenChange: (open: boolean) => void
  user: User
  onSuccess: () => void
}

export function AffiliateSettingsDialog(props: AffiliateSettingsDialogProps) {
  const { t } = useTranslation()
  const [detail, setDetail] = useState<AffiliateUserDetail | null>(null)
  const [records, setRecords] = useState<AffiliateAdminCommissionRecord[]>([])
  const [loading, setLoading] = useState(false)
  const [saving, setSaving] = useState(false)
  const [affCode, setAffCode] = useState('')
  const [rateMode, setRateMode] = useState<'inherit' | 'custom'>('inherit')
  const [customRate, setCustomRate] = useState('')
  const [inviterId, setInviterId] = useState(0)
  const [searchValue, setSearchValue] = useState('')
  const [searching, setSearching] = useState(false)
  const [candidates, setCandidates] = useState<AffiliateUserSummary[]>([])
  const [confirmOpen, setConfirmOpen] = useState(false)

  const loadData = useCallback(async () => {
    if (!props.open) return
    setLoading(true)
    try {
      const [detailResponse, recordsResponse] = await Promise.all([
        getAffiliateUser(props.user.id),
        getAffiliateCommissionRecords({
          page: 1,
          pageSize: 5,
          inviterId: props.user.id,
        }),
      ])
      if (!detailResponse.success || !detailResponse.data) {
        toast.error(
          detailResponse.message || t('Failed to load invitation settings')
        )
        return
      }
      const nextDetail = detailResponse.data
      setDetail(nextDetail)
      setAffCode(nextDetail.aff_code)
      setRateMode(
        nextDetail.custom_commission_rate === null ? 'inherit' : 'custom'
      )
      setCustomRate(
        nextDetail.custom_commission_rate === null
          ? ''
          : String(nextDetail.custom_commission_rate)
      )
      setInviterId(nextDetail.inviter?.id ?? 0)
      setCandidates(nextDetail.inviter ? [nextDetail.inviter] : [])
      setRecords(recordsResponse.data?.items ?? [])
    } catch {
      toast.error(t('Failed to load invitation settings'))
    } finally {
      setLoading(false)
    }
  }, [props.open, props.user.id, t])

  useEffect(() => {
    void loadData()
  }, [loadData])

  const handleSearch = async () => {
    setSearching(true)
    try {
      const response = await searchAffiliateUsers(searchValue.trim())
      if (!response.success) {
        toast.error(response.message || t('Failed to search users'))
        return
      }
      const results = (response.data ?? []).filter(
        (user) => user.id !== props.user.id
      )
      if (
        detail?.inviter &&
        !results.some((candidate) => candidate.id === detail.inviter?.id)
      ) {
        results.unshift(detail.inviter)
      }
      setCandidates(results)
    } catch {
      toast.error(t('Failed to search users'))
    } finally {
      setSearching(false)
    }
  }

  const submit = async () => {
    if (rateMode === 'custom' && !customRate.trim()) {
      toast.error(t('Please enter a valid number'))
      return
    }
    const parsedRate = rateMode === 'inherit' ? null : Number(customRate)
    if (
      parsedRate !== null &&
      (!Number.isFinite(parsedRate) || parsedRate < 0 || parsedRate > 100)
    ) {
      toast.error(t('Commission rate must be between 0 and 100.'))
      return
    }
    setSaving(true)
    try {
      const response = await updateAffiliateUser(props.user.id, {
        aff_code: affCode.trim(),
        custom_commission_rate: parsedRate,
        inviter_id: inviterId,
      })
      if (!response.success) {
        toast.error(response.message || t('Failed to save invitation settings'))
        return
      }
      toast.success(t('Invitation settings saved'))
      props.onSuccess()
      props.onOpenChange(false)
    } catch {
      toast.error(t('Failed to save invitation settings'))
    } finally {
      setSaving(false)
      setConfirmOpen(false)
    }
  }

  const handleSave = () => {
    const originalInviterId = detail?.inviter?.id ?? 0
    if (originalInviterId > 0 && inviterId !== originalInviterId) {
      setConfirmOpen(true)
      return
    }
    void submit()
  }

  // Base UI Select.Value shows the raw item value unless `items` maps value → label.
  // Without this map the closed trigger falls back to English keys like "inherit" / "0".
  const rateModeItems = useMemo(
    () => ({
      inherit: `${t('Inherit default')} (${(detail?.default_commission_rate ?? 0).toFixed(2)}%)`,
      custom: t('Custom rate'),
    }),
    [detail?.default_commission_rate, t]
  )

  const inviterItems = useMemo(() => {
    const items: Record<string, string> = {
      '0': t('No Inviter'),
    }
    for (const candidate of candidates) {
      items[String(candidate.id)] =
        `${candidate.username} (#${candidate.id}, ${candidate.aff_code})`
    }
    return items
  }, [candidates, t])

  return (
    <>
      <Dialog
        open={props.open}
        onOpenChange={props.onOpenChange}
        title={t('Invitation Settings')}
        description={`${props.user.username} (#${props.user.id})`}
        contentClassName='sm:max-w-2xl'
        footer={
          <Button onClick={handleSave} disabled={loading || saving}>
            {saving ? t('Saving...') : t('Save')}
          </Button>
        }
      >
        {loading ? (
          <div className='space-y-3'>
            <Skeleton className='h-16 w-full' />
            <Skeleton className='h-24 w-full' />
            <Skeleton className='h-28 w-full' />
          </div>
        ) : (
          <div className='space-y-5'>
            <div className='grid gap-4 sm:grid-cols-2'>
              <div className='space-y-2'>
                <Label htmlFor='affiliate-code'>{t('Invitation Code')}</Label>
                <Input
                  id='affiliate-code'
                  value={affCode}
                  onChange={(event) => setAffCode(event.currentTarget.value)}
                  maxLength={32}
                />
              </div>
              <div className='space-y-2'>
                <Label>{t('Commission Rate')}</Label>
                <Select
                  items={rateModeItems}
                  value={rateMode}
                  onValueChange={(value) =>
                    value !== null && setRateMode(value as 'inherit' | 'custom')
                  }
                >
                  <SelectTrigger className='w-full'>
                    <SelectValue />
                  </SelectTrigger>
                  <SelectContent alignItemWithTrigger={false}>
                    <SelectGroup>
                      <SelectItem value='inherit'>
                        {rateModeItems.inherit}
                      </SelectItem>
                      <SelectItem value='custom'>
                        {rateModeItems.custom}
                      </SelectItem>
                    </SelectGroup>
                  </SelectContent>
                </Select>
                {rateMode === 'custom' ? (
                  <Input
                    type='number'
                    min={0}
                    max={100}
                    step={0.01}
                    value={customRate}
                    onChange={(event) =>
                      setCustomRate(event.currentTarget.value)
                    }
                    placeholder='0.00'
                  />
                ) : null}
                <p className='text-muted-foreground text-xs'>
                  {t('Effective Rate')}:{' '}
                  {(rateMode === 'inherit'
                    ? (detail?.default_commission_rate ?? 0)
                    : Number.parseFloat(customRate) || 0
                  ).toFixed(2)}
                  %
                </p>
              </div>
            </div>

            <div className='space-y-2'>
              <Label>{t('Inviter')}</Label>
              <div className='flex gap-2'>
                <Input
                  value={searchValue}
                  onChange={(event) =>
                    setSearchValue(event.currentTarget.value)
                  }
                  placeholder={t(
                    'Search by user ID, username, or invitation code'
                  )}
                  onKeyDown={(event) => {
                    if (event.key === 'Enter') {
                      event.preventDefault()
                      void handleSearch()
                    }
                  }}
                />
                <Button
                  variant='outline'
                  size='icon'
                  onClick={handleSearch}
                  disabled={searching}
                  aria-label={t('Search')}
                >
                  <Search />
                </Button>
              </div>
              <Select
                items={inviterItems}
                value={String(inviterId)}
                onValueChange={(value) =>
                  value !== null && setInviterId(Number(value))
                }
              >
                <SelectTrigger className='w-full'>
                  <SelectValue placeholder={t('Select inviter')} />
                </SelectTrigger>
                <SelectContent alignItemWithTrigger={false}>
                  <SelectGroup>
                    <SelectItem value='0'>{inviterItems['0']}</SelectItem>
                    {candidates.map((candidate) => (
                      <SelectItem
                        key={candidate.id}
                        value={String(candidate.id)}
                      >
                        {inviterItems[String(candidate.id)]}
                      </SelectItem>
                    ))}
                  </SelectGroup>
                </SelectContent>
              </Select>
            </div>

            <div className='grid grid-cols-3 gap-3 border-y py-3 text-center'>
              <div>
                <div className='text-muted-foreground text-xs'>
                  {t('Invites')}
                </div>
                <div className='mt-1 font-medium tabular-nums'>
                  {detail?.invite_count ?? 0}
                </div>
              </div>
              <div>
                <div className='text-muted-foreground text-xs'>
                  {t('Pending')}
                </div>
                <div className='mt-1 font-medium tabular-nums'>
                  {formatQuota(detail?.aff_quota ?? 0)}
                </div>
              </div>
              <div>
                <div className='text-muted-foreground text-xs'>
                  {t('Total Earned')}
                </div>
                <div className='mt-1 font-medium tabular-nums'>
                  {formatQuota(detail?.aff_history_quota ?? 0)}
                </div>
              </div>
            </div>

            <div className='space-y-2'>
              <h3 className='text-sm font-medium'>
                {t('Recent Commission Records')}
              </h3>
              {records.length === 0 ? (
                <p className='text-muted-foreground py-4 text-center text-sm'>
                  {t('No commission records')}
                </p>
              ) : (
                records.map((record) => (
                  <CommissionRecordRow
                    key={record.id}
                    record={record}
                    invitee={record.invitee_username}
                  />
                ))
              )}
            </div>
          </div>
        )}
      </Dialog>

      <ConfirmDialog
        open={confirmOpen}
        onOpenChange={setConfirmOpen}
        title={inviterId === 0 ? t('Unbind Inviter') : t('Change Inviter')}
        desc={t(
          'This change only affects future paid orders and cannot be undone automatically.'
        )}
        confirmText={t('Confirm')}
        handleConfirm={submit}
      />
    </>
  )
}
