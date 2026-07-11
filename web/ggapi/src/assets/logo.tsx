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
import { type SVGProps } from 'react'

import { cn } from '@/lib/utils'

/** Compact ink-style mark for the ggapi shell (paper-sketch). */
export function Logo({ className, ...props }: SVGProps<SVGSVGElement>) {
  return (
    <svg
      id='ggapi-logo'
      viewBox='0 0 24 24'
      xmlns='http://www.w3.org/2000/svg'
      height='24'
      width='24'
      fill='none'
      stroke='currentColor'
      strokeWidth='1.7'
      strokeLinecap='round'
      strokeLinejoin='round'
      className={cn('size-6', className)}
      {...props}
    >
      <title>New API</title>
      {/* notebook page */}
      <path d='M5.2 3.8h11.2c1.1 0 1.8.9 1.8 1.9v12.6c0 1.1-.8 1.9-1.8 1.9H7.1c-1.2 0-2.1-.8-2.1-2V5.6c0-1 .8-1.8 2.1-1.8' />
      {/* binding dots */}
      <path d='M8 7.2h7.5M8 11h7.5M8 14.8h5.2' />
      {/* pen tip accent */}
      <path d='M16.2 16.5l3.2 3.1' />
    </svg>
  )
}
