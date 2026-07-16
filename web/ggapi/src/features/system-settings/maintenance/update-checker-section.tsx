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
import { ExternalLinkIcon, RefreshCcwIcon } from 'lucide-react'
import { useState } from 'react'
import { useForm } from 'react-hook-form'
import { useTranslation } from 'react-i18next'
import { toast } from 'sonner'
import * as z from 'zod'

import { Button } from '@/components/design-system/button'
import { Input } from '@/components/design-system/input'
import { Dialog } from '@/components/dialog'
import {
  Form,
  FormControl,
  FormDescription,
  FormField,
  FormItem,
  FormLabel,
  FormMessage,
} from '@/components/ui/form'
import { Markdown } from '@/components/ui/markdown'
import { api } from '@/lib/api'
import { formatTimestamp, formatTimestampToDate } from '@/lib/format'

import { SettingsForm } from '../components/settings-form-layout'
import { SettingsPageFormActions } from '../components/settings-page-context'
import { SettingsSection } from '../components/settings-section'
import { useResetForm } from '../hooks/use-reset-form'
import { useUpdateOption } from '../hooks/use-update-option'

type ReleaseInfo = {
  tag_name: string
  name?: string
  body?: string
  html_url?: string
  published_at?: string
  is_latest?: boolean
  token_configured?: boolean
  source_url?: string
}

type CheckUpdateResponse = {
  success: boolean
  message?: string
  data?: ReleaseInfo
}

const DEFAULT_UPDATE_CHECK_API_URL =
  'https://api.github.com/repos/AkumaRealLabs/ggapi/releases/latest'

const createUpdateCheckSchema = (t: (key: string) => string) =>
  z.object({
    UpdateCheckRepoAPIURL: z
      .string()
      .refine((value) => {
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
  defaultValues: UpdateCheckFormValues
}

export function UpdateCheckerSection({
  currentVersion,
  startTime,
  defaultValues,
}: UpdateCheckerSectionProps) {
  const { t } = useTranslation()
  const updateOption = useUpdateOption()
  const schema = createUpdateCheckSchema(t)
  const [checking, setChecking] = useState(false)
  const [dialogOpen, setDialogOpen] = useState(false)
  const [release, setRelease] = useState<ReleaseInfo | null>(null)

  const form = useForm<UpdateCheckFormValues>({
    resolver: zodResolver(schema),
    defaultValues,
  })

  useResetForm(form, defaultValues)

  const uptime = startTime ? formatTimestamp(startTime) : t('Unknown')
  const version = currentVersion || t('Unknown')

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

    await updateOption.updateMany(updates)

    form.setValue('UpdateCheckGitHubToken', '')
  }

  const handleCheckUpdates = async () => {
    setChecking(true)
    try {
      const res = await api.get<CheckUpdateResponse>(
        '/api/option/check_update',
        {
          // Backend returns success:false with actionable messages; avoid double toasts.
          skipBusinessError: true,
          skipErrorHandler: true,
        }
      )
      const payload = res.data
      if (!payload?.success || !payload.data?.tag_name) {
        throw new Error(
          payload?.message || t('Failed to check for updates')
        )
      }

      const data = payload.data
      if (
        data.is_latest ||
        (currentVersion && data.tag_name === currentVersion)
      ) {
        toast.success(
          t('You are running the latest version ({{version}}).', {
            version: data.tag_name,
          })
        )
        return
      }

      setRelease(data)
      setDialogOpen(true)
    } catch (error: unknown) {
      let message = t('Failed to check for updates')
      if (error && typeof error === 'object' && 'response' in error) {
        const data = (error as { response?: { data?: { message?: string } } })
          .response?.data
        if (data?.message) message = data.message
      } else if (error instanceof Error && error.message) {
        message = error.message
      }
      toast.error(message)
    } finally {
      setChecking(false)
    }
  }

  const goToRelease = () => {
    if (release?.html_url) {
      window.open(release.html_url, '_blank', 'noopener,noreferrer')
    }
  }

  return (
    <>
      <SettingsSection title={t('System maintenance')}>
        <div className='space-y-6'>
          <div className='grid gap-4 md:grid-cols-2'>
            <div className='rounded-lg border p-4'>
              <div className='text-muted-foreground text-sm'>
                {t('Current version')}
              </div>
              <div className='text-lg font-semibold'>{version}</div>
            </div>
            <div className='rounded-lg border p-4'>
              <div className='text-muted-foreground text-sm'>
                {t('Uptime since')}
              </div>
              <div className='text-lg font-semibold'>{uptime}</div>
            </div>
          </div>

          <Form {...form}>
            <SettingsForm
              onSubmit={form.handleSubmit(onSubmit)}
              autoComplete='off'
            >
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

              <div className='flex flex-wrap items-center gap-2 pt-2'>
                <Button
                  type='button'
                  onClick={handleCheckUpdates}
                  disabled={checking || updateOption.isPending}
                >
                  {checking ? (
                    t('Checking updates...')
                  ) : (
                    <>
                      <RefreshCcwIcon className='me-2 h-4 w-4' />
                      {t('Check for updates')}
                    </>
                  )}
                </Button>
              </div>
            </SettingsForm>
          </Form>
        </div>
      </SettingsSection>

      <Dialog
        open={dialogOpen}
        onOpenChange={setDialogOpen}
        title={
          release?.tag_name
            ? t('New version available: {{version}}', {
                version: release.tag_name,
              })
            : t('Release details')
        }
        description={
          release?.published_at
            ? `${t('Published')} ${formatTimestampToDate(
                new Date(release.published_at).getTime(),
                'milliseconds'
              )}`
            : undefined
        }
        contentClassName='max-h-[80vh] overflow-y-auto'
        contentHeight='auto'
        bodyClassName='space-y-4'
        footer={
          <>
            <Button
              type='button'
              variant='secondary'
              onClick={() => setDialogOpen(false)}
            >
              {t('Close')}
            </Button>
            {release?.html_url && (
              <Button type='button' onClick={goToRelease}>
                <ExternalLinkIcon className='me-2 h-4 w-4' />
                {t('Open release')}
              </Button>
            )}
          </>
        }
      >
        <div className='space-y-4'>
          {release?.body ? (
            <Markdown>{release.body}</Markdown>
          ) : (
            <p className='text-muted-foreground text-sm'>
              {t('No release notes provided.')}
            </p>
          )}
        </div>
      </Dialog>
    </>
  )
}
