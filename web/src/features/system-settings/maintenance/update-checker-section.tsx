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
import { zodResolver } from '@hookform/resolvers/zod'
import { useQueryClient } from '@tanstack/react-query'
import { useForm } from 'react-hook-form'
import { useTranslation } from 'react-i18next'
import { toast } from 'sonner'
import * as z from 'zod'

import {
  Form,
  FormControl,
  FormDescription,
  FormField,
  FormItem,
  FormLabel,
  FormMessage,
} from '@/components/ui/form'
import { Input } from '@/components/ui/input'
import { SystemUpdateAction } from '@/features/system-update/system-update-action'
import { SYSTEM_UPDATE_QUERY_KEY } from '@/features/system-update/use-system-update'
import { useStatus } from '@/hooks/use-status'
import { formatTimestamp } from '@/lib/format'

import { SettingsForm } from '../components/settings-form-layout'
import { SettingsPageFormActions } from '../components/settings-page-context'
import { SettingsSection } from '../components/settings-section'
import { useResetForm } from '../hooks/use-reset-form'
import { useUpdateOption } from '../hooks/use-update-option'

// ggapi GG-004: the server checks this release source for the update notice.
const DEFAULT_UPDATE_CHECK_API_URL =
  'https://api.github.com/repos/AkumaRealLabs/ggapi/releases/latest'

const createUpdateCheckSchema = (t: (key: string) => string) =>
  z.object({
    UpdateCheckRepoAPIURL: z.string().refine((value) => {
      const trimmed = value.trim()
      if (!trimmed) return true
      return /^https?:\/\//i.test(trimmed)
    }, t('Provide a valid URL starting with http:// or https://')),
    UpdateCheckGitHubToken: z.string(),
  })

type UpdateCheckFormValues = z.infer<ReturnType<typeof createUpdateCheckSchema>>

type UpdateCheckerSectionProps = {
  currentVersion?: string | null
  startTime?: number | null
  defaultValues?: UpdateCheckFormValues
}

export function UpdateCheckerSection(props: UpdateCheckerSectionProps) {
  const { t } = useTranslation()
  const { status } = useStatus()
  const uptime = props.startTime
    ? formatTimestamp(props.startTime)
    : t('Unknown')
  const version = status?.version || props.currentVersion || t('Unknown')

  return (
    <SettingsSection title={t('System maintenance')}>
      <div className='space-y-6'>
        <div className='grid gap-4 md:grid-cols-2'>
          <div className='rounded-lg border p-4'>
            <div className='text-muted-foreground text-sm'>
              {t('Current version')}
            </div>
            <div className='text-lg font-semibold break-all'>{version}</div>
          </div>
          <div className='rounded-lg border p-4'>
            <div className='text-muted-foreground text-sm'>
              {t('Uptime since')}
            </div>
            <div className='text-lg font-semibold'>{uptime}</div>
          </div>
        </div>
        <SystemUpdateAction compact={false} />
        {props.defaultValues && (
          <UpdateCheckSourceForm defaultValues={props.defaultValues} />
        )}
      </div>
    </SettingsSection>
  )
}

function UpdateCheckSourceForm(props: {
  defaultValues: UpdateCheckFormValues
}) {
  const { t } = useTranslation()
  const queryClient = useQueryClient()
  const updateOption = useUpdateOption()
  const schema = createUpdateCheckSchema(t)
  const { defaultValues } = props

  const form = useForm<UpdateCheckFormValues>({
    resolver: zodResolver(schema),
    defaultValues,
  })

  useResetForm(form, defaultValues)

  const onSubmit = async (values: UpdateCheckFormValues) => {
    const sanitizedUrl = values.UpdateCheckRepoAPIURL.trim()
    const sanitizedToken = values.UpdateCheckGitHubToken.trim()
    const initialUrl = defaultValues.UpdateCheckRepoAPIURL.trim()
    const initialToken = defaultValues.UpdateCheckGitHubToken.trim()

    const updates: Array<{ key: string; value: string }> = []

    if (sanitizedUrl !== initialUrl) {
      updates.push({
        key: 'UpdateCheckRepoAPIURL',
        value: sanitizedUrl || DEFAULT_UPDATE_CHECK_API_URL,
      })
    }

    // Token is never returned by GET /api/option/ (suffix Token). Only push when
    // the admin typed a non-empty new value (same pattern as SMTPToken).
    if (sanitizedToken && sanitizedToken !== initialToken) {
      updates.push({
        key: 'UpdateCheckGitHubToken',
        value: sanitizedToken,
      })
    }

    if (updates.length === 0) {
      toast.message(t('No changes to save'))
      return
    }

    for (const update of updates) {
      await updateOption.mutateAsync(update)
    }

    form.setValue('UpdateCheckGitHubToken', '')
    // A new source or token invalidates the cached release snapshot.
    void queryClient.invalidateQueries({ queryKey: SYSTEM_UPDATE_QUERY_KEY })
  }

  return (
    <Form {...form}>
      <SettingsForm onSubmit={form.handleSubmit(onSubmit)} autoComplete='off'>
        <SettingsPageFormActions
          onSave={form.handleSubmit(onSubmit)}
          isSaving={updateOption.isPending}
          saveLabel='Save update check settings'
        />

        <FormField
          control={form.control}
          name='UpdateCheckRepoAPIURL'
          render={({ field }) => (
            <FormItem>
              <FormLabel>{t('Update check release API URL')}</FormLabel>
              <FormControl>
                <Input
                  type='url'
                  inputMode='url'
                  placeholder={DEFAULT_UPDATE_CHECK_API_URL}
                  autoComplete='off'
                  {...field}
                />
              </FormControl>
              <FormDescription>
                {t(
                  'GitHub Releases API URL (…/repos/owner/repo/releases/latest). Used by the server when checking for updates. Leave empty to reset to the default ggapi repo.'
                )}
              </FormDescription>
              <FormMessage />
            </FormItem>
          )}
        />

        <FormField
          control={form.control}
          name='UpdateCheckGitHubToken'
          render={({ field }) => (
            <FormItem>
              <FormLabel>{t('GitHub PAT for update check')}</FormLabel>
              <FormControl>
                <Input
                  type='password'
                  placeholder={t('Enter new token to update')}
                  autoComplete='new-password'
                  {...field}
                />
              </FormControl>
              <FormDescription>
                {t(
                  'Required for private repositories. Stored server-side only and never returned by the options API. Leave blank to keep the existing token.'
                )}
              </FormDescription>
              <FormMessage />
            </FormItem>
          )}
        />
      </SettingsForm>
    </Form>
  )
}
