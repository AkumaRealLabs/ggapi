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
import type { ReactNode } from 'react'

import { PageTransition } from '@/components/page-transition'
import { cn } from '@/lib/utils'

/** Shared shell for public catalog pages (pricing, rankings, …). */
export const PUBLIC_PAGE_SHELL_CLASS =
  'mx-auto w-full max-w-7xl px-4 pt-24 pb-12 sm:px-6 lg:px-8'

export interface PublicPageShellProps {
  children: ReactNode
  className?: string
}

export function PublicPageShell(props: PublicPageShellProps) {
  return (
    <PageTransition className={cn(PUBLIC_PAGE_SHELL_CLASS, props.className)}>
      <div data-slot='public-page-shell' className='contents'>
        {props.children}
      </div>
    </PageTransition>
  )
}

export interface PublicPageHeaderProps {
  title: ReactNode
  description?: ReactNode
  /** Full-width slot under the title block (tabs, filters, meta). */
  children?: ReactNode
  className?: string
}

/**
 * Shared page header for public catalog surfaces.
 * Title follows the product page-title contract: text-lg / semibold / tight.
 * Title/description width follows the page shell (no second measure here).
 */
export function PublicPageHeader(props: PublicPageHeaderProps) {
  return (
    <header
      data-slot='public-page-header'
      className={cn('mb-8 space-y-6', props.className)}
    >
      <div>
        <h1
          data-slot='public-page-title'
          className='text-lg font-semibold tracking-tight'
        >
          {props.title}
        </h1>
        {props.description != null && props.description !== '' && (
          <p
            data-slot='public-page-subtitle'
            className='text-muted-foreground mt-2 text-sm leading-relaxed'
          >
            {props.description}
          </p>
        )}
      </div>
      {props.children}
    </header>
  )
}
