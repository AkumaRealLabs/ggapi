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
import { useMutation, useQueryClient } from '@tanstack/react-query'
import i18next from 'i18next'
import { useCallback, useRef, useState } from 'react'
import { toast } from 'sonner'

import { updateSystemOption } from '../api'
import type { UpdateOptionRequest } from '../types'

// Configuration keys that require status refresh
const STATUS_RELATED_KEYS = new Set([
  'theme.frontend',
  'HeaderNavModules',
  'SidebarModulesAdmin',
  'Notice',
  'LogConsumeEnabled',
  'QuotaPerUnit',
  'USDExchangeRate',
  'DisplayInCurrencyEnabled',
  'DisplayTokenStatEnabled',
  'general_setting.quota_display_type',
  'general_setting.custom_currency_symbol',
  'general_setting.custom_currency_exchange_rate',
])

export type UpdateOptionToastOptions = {
  /**
   * Skip the success toast (e.g. when more option writes follow and the caller
   * toasts once after the full batch).
   */
  quiet?: boolean
  /**
   * Override the default success toast copy (domain-specific save messages).
   * Ignored when `quiet` is true.
   */
  successMessage?: string
}

/**
 * Toggle keys (…Enabled / …enabled). Only *turning on* is deferred: credentials
 * must land first on fresh setup. Turning off stays early so a later failure
 * never leaves a feature enabled after its prerequisites were cleared.
 */
function isEnableOptionKey(key: string): boolean {
  return (
    /Enabled$/i.test(key) ||
    /\.enabled$/i.test(key) ||
    /_enabled$/i.test(key)
  )
}

function isEnableOnWrite(request: UpdateOptionRequest): boolean {
  if (!isEnableOptionKey(request.key)) return false
  const value = request.value
  return value === true || value === 'true' || value === 1 || value === '1'
}

function orderOptionWrites(
  requests: UpdateOptionRequest[]
): UpdateOptionRequest[] {
  const early: UpdateOptionRequest[] = []
  const enableOn: UpdateOptionRequest[] = []
  for (const request of requests) {
    if (isEnableOnWrite(request)) {
      enableOn.push(request)
    } else {
      early.push(request)
    }
  }
  if (enableOn.length === 0) return requests
  return [...early, ...enableOn]
}

export function useUpdateOption() {
  const queryClient = useQueryClient()
  // Direct writes bypass useMutation; refcount so overlapping batches keep
  // isPending true until the last one finishes.
  const batchInflightRef = useRef(0)
  const [batchPending, setBatchPending] = useState(false)

  const beginBatch = useCallback(() => {
    batchInflightRef.current += 1
    setBatchPending(true)
  }, [])

  const endBatch = useCallback(() => {
    batchInflightRef.current = Math.max(0, batchInflightRef.current - 1)
    if (batchInflightRef.current === 0) {
      setBatchPending(false)
    }
  }, [])

  const invalidateForKeys = useCallback(
    (keys: string[]) => {
      queryClient.invalidateQueries({ queryKey: ['system-options'] })

      if (keys.some((key) => STATUS_RELATED_KEYS.has(key))) {
        queryClient.invalidateQueries({ queryKey: ['status'] })
        try {
          window.localStorage.removeItem('status')
        } catch {
          /* empty */
        }
      }
    },
    [queryClient]
  )

  const mutation = useMutation({
    // Failure semantics live in updateSystemOption (throw on success:false).
    mutationFn: updateSystemOption,
    onSuccess: (_data, variables: UpdateOptionRequest) => {
      invalidateForKeys([variables.key])
      toast.success(i18next.t('Setting updated successfully'))
    },
    onError: (error: Error) => {
      toast.error(error.message || i18next.t('Failed to update setting'))
    },
  })

  /**
   * Apply several option writes sequentially with a single success toast.
   *
   * Prefer this over a `for` + `mutateAsync` loop whenever more than one key
   * may change in one save — otherwise each success fires its own toast.
   *
   * Not a transaction: there is no backend multi-option batch API. Writes are
   * fail-fast; keys that already succeeded stay on the server and are not
   * rolled back. On any failure (or full success) we still invalidate caches
   * so the UI can reconcile partial application honestly.
   */
  const updateMany = useCallback(
    async (
      requests: UpdateOptionRequest[],
      options?: UpdateOptionToastOptions
    ) => {
      if (requests.length === 0) return

      // Credentials / whitelist first; enable flags last (fail-fast safe).
      const ordered = orderOptionWrites(requests)
      const keys = ordered.map((request) => request.key)
      beginBatch()
      try {
        for (const request of ordered) {
          await updateSystemOption(request)
        }
        invalidateForKeys(keys)
        if (!options?.quiet) {
          toast.success(
            options?.successMessage ??
              i18next.t('Setting updated successfully')
          )
        }
      } catch (error) {
        // Refresh even on partial failure so form/server state can reconverge.
        // Earlier keys in this batch may already be persisted (not rolled back).
        invalidateForKeys(keys)
        const message =
          error instanceof Error
            ? error.message
            : i18next.t('Failed to update setting')
        toast.error(message)
        throw error instanceof Error ? error : new Error(message)
      } finally {
        endBatch()
      }
    },
    [beginBatch, endBatch, invalidateForKeys]
  )

  /**
   * Single-key write with the same toast / isPending path as updateMany.
   * Use when you need a domain successMessage or quiet flag; otherwise
   * mutate / mutateAsync is fine for default toast copy.
   */
  const updateOne = useCallback(
    (request: UpdateOptionRequest, options?: UpdateOptionToastOptions) =>
      updateMany([request], options),
    [updateMany]
  )

  return {
    ...mutation,
    isPending: mutation.isPending || batchPending,
    updateMany,
    updateOne,
  }
}
