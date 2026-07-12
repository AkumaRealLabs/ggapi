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
import { Link } from '@tanstack/react-router'
import { useTranslation } from 'react-i18next'

import { Skeleton } from '@/components/ui/skeleton'
import { useSystemConfig } from '@/hooks/use-system-config'

type AuthLayoutProps = {
  children: React.ReactNode
}

/**
 * Notebook-style auth shell for the ggapi paper-sketch product.
 * Full-page paper grid + ink card — not a centered SaaS form on white void.
 */
export function AuthLayout({ children }: AuthLayoutProps) {
  const { t } = useTranslation()
  const { systemName, logo, loading } = useSystemConfig()

  return (
    <div className='gg-auth-shell' data-slot='sketch-auth-shell'>
      <div className='gg-auth-card' data-slot='sketch-auth-card'>
        <Link to='/' className='gg-auth-brand' data-slot='sketch-auth-brand'>
          <div className='relative size-9 shrink-0'>
            {loading ? (
              <Skeleton className='absolute inset-0 rounded-md' />
            ) : (
              <img
                src={logo}
                alt={t('Logo')}
                className='size-9 object-cover'
              />
            )}
          </div>
          {loading ? (
            <Skeleton className='h-6 w-24' />
          ) : (
            <span>{systemName}</span>
          )}
        </Link>
        <p className='text-[var(--ink-soft,var(--muted-foreground))] mb-5 text-[13.5px] leading-relaxed'>
          {t('Sign in on paper — same gateway, quieter surface.')}
        </p>
        <div data-slot='sketch-auth-body' className='space-y-4'>
          {children}
        </div>
      </div>
    </div>
  )
}
