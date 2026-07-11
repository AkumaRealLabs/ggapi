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
/**
 * Compact tech-visual-explainer style step strip for create drawers.
 * Pure CSS state — no rough.js dependency; works offline.
 * When `activeStep` is driven by form fields, previous steps mark done.
 */
import { useTranslation } from 'react-i18next'

import { cn } from '@/lib/utils'

export type CreateFlowStep = {
  /** English i18n key / source string */
  label: string
  /** Optional short hint (English i18n key) */
  hint?: string
}

type CreateFlowGuideProps = {
  steps: CreateFlowStep[]
  /** 0-based active step; all previous are marked complete */
  activeStep?: number
  className?: string
  title?: string
  /** Optional: jump to a step (e.g. scroll to section) */
  onStepClick?: (index: number) => void
}

export function CreateFlowGuide(props: CreateFlowGuideProps) {
  const { t } = useTranslation()
  const active = props.activeStep ?? 0
  const clickable = typeof props.onStepClick === 'function'

  return (
    <div
      data-slot='create-flow-guide'
      className={cn('shrink-0 px-4 pt-3 sm:px-6', props.className)}
    >
      {props.title ? (
        <p data-slot='create-flow-title' className='mb-2 text-xs font-bold'>
          {t(props.title)}
        </p>
      ) : null}
      <ol
        className='flex flex-wrap gap-2'
        aria-label={t(props.title || 'Steps')}
      >
        {props.steps.map((step, i) => {
          const state =
            i < active ? 'done' : i === active ? 'current' : 'todo'
          const indexLabel = state === 'done' ? '✓' : String(i + 1)
          return (
            <li
              key={step.label}
              data-slot='create-flow-step'
              data-state={state}
              data-clickable={clickable ? 'true' : undefined}
              className='min-w-0'
            >
              {clickable ? (
                <button
                  type='button'
                  className='flex w-full items-start gap-2 text-left'
                  onClick={() => props.onStepClick?.(i)}
                >
                  <span data-slot='create-flow-index' aria-hidden='true'>
                    {indexLabel}
                  </span>
                  <span className='min-w-0'>
                    <span data-slot='create-flow-label'>{t(step.label)}</span>
                    {step.hint ? (
                      <span data-slot='create-flow-hint' className='block'>
                        {t(step.hint)}
                      </span>
                    ) : null}
                  </span>
                </button>
              ) : (
                <div className='flex items-start gap-2'>
                  <span data-slot='create-flow-index' aria-hidden='true'>
                    {indexLabel}
                  </span>
                  <span className='min-w-0'>
                    <span data-slot='create-flow-label'>{t(step.label)}</span>
                    {step.hint ? (
                      <span data-slot='create-flow-hint' className='block'>
                        {t(step.hint)}
                      </span>
                    ) : null}
                  </span>
                </div>
              )}
            </li>
          )
        })}
      </ol>
    </div>
  )
}
