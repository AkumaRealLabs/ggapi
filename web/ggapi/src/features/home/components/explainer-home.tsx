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
 * Default home — faithful tech-visual-explainer sketch page.
 * Layout: title → stepper + diagram | panel → card grid.
 * Not a SaaS marketing landing (no Hero / Stats / CTA chrome).
 */
import { Link } from '@tanstack/react-router'
import { useCallback, useEffect, useMemo, useRef, useState } from 'react'
import { useTranslation } from 'react-i18next'

import { useSystemConfig } from '@/hooks/use-system-config'

import {
  applyStepClasses,
  createSketchContext,
  cssIntro,
  leftOf,
  NODE_H,
  NODE_W,
  replayActiveEdges,
  rightOf,
  svgEl,
  syncFlows,
  type SketchNode,
  type SketchStep,
} from '../lib/sketch-engine'
import '../styles/home-explainer.css'

type ExplainerHomeProps = {
  isAuthenticated?: boolean
}

const VIEWBOX = '0 0 760 280'

export function ExplainerHome(props: ExplainerHomeProps) {
  const { t } = useTranslation()
  const { systemName, logo, loading } = useSystemConfig()
  const brand = systemName || 'ggapi'

  const diagramRef = useRef<HTMLDivElement>(null)
  const rootRef = useRef<HTMLDivElement>(null)
  const svgRef = useRef<SVGSVGElement | null>(null)
  const [step, setStep] = useState(0)

  const reduced = useMemo(
    () =>
      typeof window !== 'undefined' &&
      window.matchMedia('(prefers-reduced-motion: reduce)').matches,
    []
  )

  const nodes: SketchNode[] = useMemo(
    () => [
      {
        id: 'client',
        x: 36,
        y: 112,
        label: t('Client'),
        sub: t('App / SDK'),
        color: 'var(--amber)',
      },
      {
        id: 'auth',
        x: 250,
        y: 48,
        label: t('Auth'),
        sub: t('Token · ACL'),
        color: 'var(--blue)',
      },
      {
        id: 'relay',
        x: 250,
        y: 168,
        label: t('Relay'),
        sub: t('Route · Bill'),
        color: 'var(--teal)',
      },
      {
        id: 'up',
        x: 520,
        y: 112,
        w: 160,
        label: t('Upstream'),
        sub: t('40+ providers'),
        color: 'var(--purple)',
      },
    ],
    [t]
  )

  const groups = useMemo(
    () => [
      {
        x: 16,
        y: 24,
        w: 190,
        h: 232,
        label: t('ENTRY'),
        color: 'var(--amber)',
      },
      {
        x: 220,
        y: 24,
        w: 250,
        h: 232,
        label: 'GGAPI',
        color: 'var(--blue)',
      },
      {
        x: 490,
        y: 24,
        w: 220,
        h: 232,
        label: t('PROVIDERS'),
        color: 'var(--purple)',
      },
    ],
    [t]
  )

  const edgeLabels = useMemo(
    () => ({
      in: t('HTTP · OpenAI-style'),
      pass: t('pass'),
      out: t('provider call'),
    }),
    [t]
  )

  const steps: SketchStep[] = useMemo(
    () => [
      {
        label: t('① Request'),
        title: t('Client hits the gateway'),
        desc: t(
          'Apps keep one base URL and OpenAI-compatible paths. Traffic enters ggapi instead of each vendor SDK.'
        ),
        nodes: ['client', 'auth'],
        edges: ['e-in'],
        kv: [
          [t('Protocol'), 'OpenAI / Claude / Gemini'],
          [t('Auth'), t('API key or JWT')],
        ],
      },
      {
        label: t('② Authorize'),
        title: t('Token, group, and limits'),
        desc: t(
          'Keys, user groups, rate limits and model ACL decide whether the call continues. Failures stay explicit for operators.'
        ),
        nodes: ['auth', 'relay'],
        edges: ['e-auth-relay'],
        kv: [
          [t('Check'), t('Quota · scope · channel')],
          [t('On fail'), t('Reject with clear error')],
        ],
      },
      {
        label: t('③ Relay'),
        title: t('Route and pre-consume'),
        desc: t(
          'The relay maps the model to a channel, may pre-consume quota, then calls the chosen upstream adapter.'
        ),
        nodes: ['relay', 'up'],
        edges: ['e-out'],
        kv: [
          [t('Billing'), t('Pre-consume → settle')],
          [t('Upstream'), t('Provider-specific adapter')],
        ],
      },
      {
        label: t('④ Settle'),
        title: t('Usage trail on paper'),
        desc: t(
          'Tokens and cost land in logs. Operators can read the ink trail without a mystery SaaS meter.'
        ),
        nodes: ['relay', 'up', 'auth'],
        edges: ['e-out', 'e-auth-relay'],
        kv: [
          [t('Log'), t('Usage · model · channel')],
          [t('Surface'), t('Console · export')],
        ],
      },
    ],
    [t]
  )

  const cards = useMemo(
    () => [
      {
        title: t('Channels & models'),
        body: t(
          'Map client model names to upstream channels with groups, weights, and fallbacks.'
        ),
      },
      {
        title: t('Quota path'),
        body: t(
          'Pre-consume, settle, refund — numbers stay auditable for operators.'
        ),
      },
      {
        title: t('Protocol adapters'),
        body: t(
          'One entry door; Claude, Gemini, OpenAI-style and more behind it.'
        ),
      },
      {
        title: t('Self-hosted'),
        body: t(
          'SQLite, MySQL, or PostgreSQL. Your data plane, your notebook.'
        ),
      },
    ],
    [t]
  )

  const renderAll = useCallback(() => {
    const box = diagramRef.current
    if (!box) return null

    const svg = svgEl('svg', {
      viewBox: VIEWBOX,
      role: 'img',
    }) as SVGSVGElement
    const ctx = createSketchContext(svg, true)

    for (const g of groups) {
      svg.appendChild(ctx.groupArea(g))
    }

    const byId = (id: string) => nodes.find((n) => n.id === id)!

    {
      const a = rightOf(byId('client'))
      const b = leftOf(byId('auth'))
      svg.appendChild(
        ctx.edgeLine({
          id: 'e-in',
          x1: a.x + 4,
          y1: a.y,
          x2: b.x - 4,
          y2: b.y,
          label: edgeLabels.in,
          color: 'var(--amber)',
          curve: 28,
        })
      )
    }
    {
      const auth = byId('auth')
      const relay = byId('relay')
      const x = auth.x + (auth.w || NODE_W) / 2
      svg.appendChild(
        ctx.edgeLine({
          id: 'e-auth-relay',
          x1: x,
          y1: auth.y + (auth.h || NODE_H) + 2,
          x2: x,
          y2: relay.y - 2,
          label: edgeLabels.pass,
          color: 'var(--blue)',
        })
      )
    }
    {
      const a = rightOf(byId('relay'))
      const b = leftOf(byId('up'))
      svg.appendChild(
        ctx.edgeLine({
          id: 'e-out',
          x1: a.x + 4,
          y1: a.y,
          x2: b.x - 4,
          y2: b.y,
          label: edgeLabels.out,
          color: 'var(--teal)',
          curve: -24,
        })
      )
    }

    for (const n of nodes) {
      svg.appendChild(ctx.nodeBox(n))
    }

    box.replaceChildren(svg)
    svgRef.current = svg
    return svg
  }, [nodes, groups, edgeLabels])

  const goStep = useCallback(
    (i: number) => {
      const next = Math.max(0, Math.min(i, steps.length - 1))
      setStep(next)
      const root = rootRef.current
      const svg = svgRef.current
      if (!root || !svg) return
      const s = steps[next]
      applyStepClasses(root, s, next)
      syncFlows(svg, s, reduced)
      if (!reduced) replayActiveEdges(svg, s)
    },
    [steps, reduced]
  )

  useEffect(() => {
    const svg = renderAll()
    const root = rootRef.current
    if (!svg || !root) return

    setStep(0)
    applyStepClasses(root, steps[0], 0)
    if (reduced) {
      syncFlows(svg, steps[0], true)
    } else {
      const ms = cssIntro(svg)
      const timer = window.setTimeout(() => {
        syncFlows(svg, steps[0], false)
      }, ms + 100)
      return () => window.clearTimeout(timer)
    }
    // steps[0] tracked via renderAll (lang) + steps identity
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [renderAll, reduced])

  useEffect(() => {
    const root = rootRef.current
    const svg = svgRef.current
    if (!root || !svg) return
    applyStepClasses(root, steps[step], step)
    syncFlows(svg, steps[step], reduced)
  }, [step, steps, reduced])

  useEffect(() => {
    const onKey = (e: KeyboardEvent) => {
      if (e.key === 'ArrowRight') goStep(step + 1)
      if (e.key === 'ArrowLeft') goStep(step - 1)
    }
    document.addEventListener('keydown', onKey)
    return () => document.removeEventListener('keydown', onKey)
  }, [goStep, step])

  const current = steps[step]

  return (
    <div
      className='gg-explainer'
      data-slot='sketch-home'
      ref={rootRef}
    >
      <div className='container'>
        <div className='top-bar anim-in' style={{ ['--i' as string]: 0 }}>
          <Link to='/' className='brand' data-slot='sketch-home-brand'>
            {!loading && logo ? (
              <img src={logo} alt='' />
            ) : (
              <span
                style={{
                  width: 28,
                  height: 28,
                  border: '1.4px solid var(--ink)',
                  borderRadius: 'var(--wobble-b)',
                  display: 'inline-block',
                }}
              />
            )}
            {brand}
          </Link>
          <div className='top-actions'>
            <Link to='/pricing' className='nav-btn'>
              {t('Pricing')}
            </Link>
            {props.isAuthenticated ? (
              <Link to='/dashboard' className='nav-btn is-primary'>
                {t('Dashboard')}
              </Link>
            ) : (
              <Link to='/sign-in' className='nav-btn is-primary'>
                {t('Sign in')}
              </Link>
            )}
          </div>
        </div>

        <header className='anim-in' style={{ ['--i' as string]: 0 }}>
          <h1>
            {brand} · {t('AI API gateway')}
          </h1>
          <p className='subtitle'>
            {t(
              'Hand-drawn walkthrough of how requests enter, authorize, relay, and settle. Same mechanics as the console — no stock SaaS landing chrome.'
            )}
          </p>
        </header>

        <section className='layout-split'>
          <div>
            <div
              className='stepper anim-in'
              style={{ ['--i' as string]: 1 }}
              role='tablist'
              aria-label={t('Request steps')}
            >
              {steps.map((s, i) => (
                <button
                  key={s.label}
                  type='button'
                  className={`step-btn${i === step ? ' is-current' : ''}`}
                  data-i={i}
                  onClick={() => goStep(i)}
                >
                  {s.label}
                </button>
              ))}
            </div>
            <div
              className='diagram anim-in'
              id='diagram'
              ref={diagramRef}
              style={{ ['--i' as string]: 2, marginTop: 12 }}
              role='img'
              aria-label={t('Gateway request flow diagram')}
            />
          </div>
          <aside
            className='panel anim-in'
            id='step-panel'
            style={{ ['--i' as string]: 3 }}
          >
            <div className='panel-swap' key={step}>
              <h3>{current.title}</h3>
              <p>{current.desc}</p>
              {current.kv && current.kv.length > 0 ? (
                <div className='panel-rule'>
                  {current.kv.map(([k, v]) => (
                    <div className='kv' key={k}>
                      <span className='k'>{k}</span>
                      <span className='v'>{v}</span>
                    </div>
                  ))}
                </div>
              ) : null}
            </div>
          </aside>
        </section>

        <section className='anim-in' style={{ ['--i' as string]: 4 }}>
          <h2 style={{ marginBottom: 16 }}>{t('What sits on the paper')}</h2>
          <div className='card-grid'>
            {cards.map((c, i) => (
              <div className='card' key={c.title}>
                <h3>
                  {String(i + 1).padStart(2, '0')} · {c.title}
                </h3>
                <p>{c.body}</p>
              </div>
            ))}
          </div>
        </section>

        <p className='footer-note'>
          {t('Based on')}{' '}
          <a
            href='https://github.com/QuantumNous/new-api'
            target='_blank'
            rel='noopener noreferrer'
          >
            new-api
          </a>
          {' · '}
          QuantumNous
          {' · '}
          {t('Visual system: tech-visual-explainer sketch template')}
        </p>
      </div>
    </div>
  )
}
