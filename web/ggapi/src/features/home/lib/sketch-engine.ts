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
 * Port of tech-visual-explainer template-sketch.html render helpers.
 * rough.js optional: clean geometry fallback when unavailable.
 */
import rough from 'roughjs'

export const SKETCH = { seed: 7, roughness: 1.2, bowing: 1.1 }
export const NODE_W = 150
export const NODE_H = 52

export type SketchNode = {
  id: string
  x: number
  y: number
  w?: number
  h?: number
  label: string
  sub?: string
  color?: string
}

export type SketchEdge = {
  id: string
  from: string
  to: string
  label?: string
  curve?: number
  color?: string
}

export type SketchStep = {
  label: string
  title: string
  desc: string
  nodes: string[]
  edges: string[]
  kv?: [string, string][]
}

const NS = 'http://www.w3.org/2000/svg'

export function svgEl(
  tag: string,
  attrs: Record<string, string | number> = {}
): SVGElement {
  const el = document.createElementNS(NS, tag)
  for (const [k, v] of Object.entries(attrs)) {
    el.setAttribute(k, String(v))
  }
  return el
}

export function hashId(s: string): number {
  let h = 9
  for (const c of String(s)) h = (h * 31 + c.charCodeAt(0)) % 1000
  return h
}

export function text(
  x: number,
  y: number,
  str: string,
  cls: string
): SVGTextElement {
  const t = svgEl('text', { x, y, class: cls }) as SVGTextElement
  t.textContent = str
  return t
}

type RoughSvg = ReturnType<typeof rough.svg>

/** Drop hard-coded stroke so CSS tokens (var(--c) / var(--ink)) own the ink. */
function clearPaintAttrs(el: Element) {
  el.removeAttribute('stroke')
  el.removeAttribute('fill')
  el.querySelectorAll('path, line, rect, circle, ellipse, polyline, polygon').forEach(
    (child) => {
      child.removeAttribute('stroke')
      // Keep fill="none" so shapes stay hollow; drop solid fills from rough.
      if (child.getAttribute('fill') && child.getAttribute('fill') !== 'none') {
        child.removeAttribute('fill')
      }
    }
  )
}

export function createSketchContext(svg: SVGSVGElement, hasRough: boolean) {
  let rcHolder: RoughSvg | null = null
  if (hasRough) {
    try {
      rcHolder = rough.svg(svg)
    } catch {
      rcHolder = null
    }
  }

  function sketchRect(
    x: number,
    y: number,
    w: number,
    h: number,
    opts: { seed?: number; dash?: boolean; fill?: boolean } = {}
  ): SVGElement {
    if (!rcHolder) {
      const r = svgEl('rect', { x, y, width: w, height: h, rx: 8 })
      if (opts.dash) r.setAttribute('stroke-dasharray', '8 6')
      return r
    }
    const drawn = rcHolder.rectangle(x, y, w, h, {
      roughness: SKETCH.roughness,
      bowing: SKETCH.bowing,
      seed: opts.seed ?? SKETCH.seed,
      strokeWidth: 1.6,
      ...(opts.dash ? { strokeLineDash: [8, 6] } : {}),
      ...(opts.fill
        ? {
            fill: '#000',
            fillStyle: 'hachure',
            fillWeight: 0.6,
            hachureGap: 7,
          }
        : {}),
    })
    clearPaintAttrs(drawn)
    return drawn
  }

  function sketchPath(d: string, opts: { seed?: number } = {}): SVGElement {
    if (!rcHolder) {
      const p = svgEl('path', { d, class: 'shaft' })
      return p
    }
    const g = rcHolder.path(d, {
      roughness: Math.min(SKETCH.roughness, 1),
      bowing: 0.8,
      seed: opts.seed ?? SKETCH.seed,
      strokeWidth: 1.6,
    })
    g.classList.add('shaft')
    clearPaintAttrs(g)
    return g
  }

  function nodeBox(n: SketchNode): SVGElement {
    const w = n.w ?? NODE_W
    const h = n.h ?? NODE_H
    const g = svgEl('g', { class: 'node', 'data-id': n.id })
    if (n.color) g.style.setProperty('--c', n.color)

    if (rcHolder) {
      const hf = svgEl('g', { class: 'hfill' })
      const hatch = rcHolder.rectangle(n.x, n.y, w, h, {
        seed: hashId(n.id),
        stroke: 'none',
        fill: '#000',
        fillStyle: 'hachure',
        fillWeight: 0.5,
        hachureGap: 8,
        roughness: 1,
      })
      // Hachure keeps its own strokes; only drop fill so CSS can recolor stroke.
      hatch.querySelectorAll('path, line').forEach((child) => {
        child.removeAttribute('stroke')
      })
      hf.appendChild(hatch)
      g.appendChild(hf)
    }

    const shape = svgEl('g', { class: 'shape' })
    shape.appendChild(sketchRect(n.x, n.y, w, h, { seed: hashId(n.id) }))
    g.appendChild(shape)

    const labelY = n.sub ? n.y + h / 2 - 8 : n.y + h / 2
    g.appendChild(text(n.x + w / 2, labelY, n.label, 'node-label'))
    if (n.sub) {
      g.appendChild(text(n.x + w / 2, n.y + h / 2 + 13, n.sub, 'node-sub'))
    }
    return g
  }

  function edgeLine(e: {
    id: string
    x1: number
    y1: number
    x2: number
    y2: number
    label?: string
    curve?: number
    color?: string
  }): SVGElement {
    const g = svgEl('g', { class: 'edge', 'data-id': e.id })
    if (e.color) g.style.setProperty('--c', e.color)

    let d: string
    let lx: number
    let ly: number
    let ang: number
    const curve = e.curve ?? 0

    if (curve) {
      const dx = e.x2 - e.x1
      const dy = e.y2 - e.y1
      const len = Math.hypot(dx, dy) || 1
      const cx = (e.x1 + e.x2) / 2 - (dy / len) * curve
      const cy = (e.y1 + e.y2) / 2 + (dx / len) * curve
      d = `M ${e.x1} ${e.y1} Q ${cx} ${cy} ${e.x2} ${e.y2}`
      lx = 0.25 * e.x1 + 0.5 * cx + 0.25 * e.x2
      ly = 0.25 * e.y1 + 0.5 * cy + 0.25 * e.y2 - 8
      ang = Math.atan2(e.y2 - cy, e.x2 - cx)
    } else {
      d = `M ${e.x1} ${e.y1} L ${e.x2} ${e.y2}`
      lx = (e.x1 + e.x2) / 2
      ly = (e.y1 + e.y2) / 2 - 8
      ang = Math.atan2(e.y2 - e.y1, e.x2 - e.x1)
    }

    g.dataset.d = d
    const w = svgEl('g')
    w.appendChild(sketchPath(d, { seed: hashId(e.id) }))
    for (const side of [0.5, -0.5] as const) {
      const hx = e.x2 - 11 * Math.cos(ang + side)
      const hy = e.y2 - 11 * Math.sin(ang + side)
      w.appendChild(
        sketchPath(`M ${hx} ${hy} L ${e.x2} ${e.y2}`, {
          seed: hashId(e.id) + 7,
        })
      )
    }
    g.appendChild(w)

    if (e.label) {
      const t = text(lx, ly, e.label, 'edge-label')
      t.dataset.id = e.id
      g.appendChild(t)
    }
    return g
  }

  function groupArea(opts: {
    x: number
    y: number
    w: number
    h: number
    label: string
    color?: string
  }): SVGElement {
    const g = svgEl('g', { class: 'group' })
    if (opts.color) g.style.setProperty('--c', opts.color)
    g.appendChild(
      sketchRect(opts.x, opts.y, opts.w, opts.h, {
        seed: hashId(opts.label),
        dash: true,
      })
    )
    const t = text(opts.x + 14, opts.y + 18, opts.label, '')
    t.setAttribute('text-anchor', 'start')
    g.appendChild(t)
    return g
  }

  return { nodeBox, edgeLine, groupArea, sketchRect, sketchPath }
}

export function rightOf(n: SketchNode) {
  return { x: n.x + (n.w || NODE_W), y: n.y + (n.h || NODE_H) / 2 }
}
export function leftOf(n: SketchNode) {
  return { x: n.x, y: n.y + (n.h || NODE_H) / 2 }
}

export function flowDots(
  d: string,
  opts: {
    color?: string
    dur?: number
    count?: number
    r?: number
    reduced?: boolean
  } = {}
): SVGElement {
  const {
    color = 'var(--blue)',
    dur = 2.2,
    count = 2,
    r = 3.5,
    reduced = false,
  } = opts
  const g = svgEl('g', { class: 'flow' })
  for (let i = 0; i < count && !reduced; i++) {
    const c = svgEl('circle', {
      r,
      fill: color,
      class: 'flow-dot',
    }) as SVGCircleElement
    c.style.offsetPath = `path('${d}')`
    c.style.animationDuration = `${dur}s`
    c.style.animationDelay = `${(-dur / count) * i}s`
    g.appendChild(c)
  }
  return g
}

export function applyStepClasses(
  root: ParentNode,
  step: SketchStep,
  stepIndex: number
) {
  // Only the diagram stepper — not top-nav links that also use .step-btn look.
  root.querySelectorAll('.stepper .step-btn').forEach((b, i) => {
    b.classList.toggle('is-current', i === stepIndex)
  })
  root.querySelectorAll('.node').forEach((el) => {
    const id = (el as HTMLElement).dataset.id || ''
    const on = step.nodes.includes(id)
    el.classList.toggle('is-active', on)
    el.classList.toggle('is-dimmed', !on)
  })
  root.querySelectorAll('.edge, .edge-label').forEach((el) => {
    const id =
      (el as HTMLElement).dataset.id ||
      el.closest('.edge')?.getAttribute('data-id') ||
      ''
    const on = step.edges.includes(id)
    el.classList.toggle('is-active', on)
    el.classList.toggle('is-dimmed', !on)
  })
}

export function syncFlows(
  svg: SVGSVGElement,
  step: SketchStep,
  reduced: boolean
) {
  svg.querySelectorAll('.flow').forEach((f) => f.remove())
  if (reduced) return
  for (const id of step.edges || []) {
    const eg = svg.querySelector(`.edge[data-id="${id}"]`) as SVGElement | null
    if (!eg?.dataset.d) continue
    const color = eg.style.getPropertyValue('--c') || 'var(--blue)'
    svg.appendChild(flowDots(eg.dataset.d, { color, reduced }))
  }
}

export function cssIntro(svg: SVGSVGElement): number {
  let i = 0
  svg
    .querySelectorAll(
      '.shape path, .edge path, .group path, .shape rect, .group rect'
    )
    .forEach((p) => {
      p.setAttribute('pathLength', '1')
      ;(p as SVGElement).style.setProperty('--di', String(i++))
      p.classList.add('draw-in')
    })
  svg.querySelectorAll('text').forEach((t) => {
    ;(t as SVGElement).style.setProperty('--di', String(i))
    t.classList.add('fade-el')
  })
  svg.querySelectorAll('.hfill').forEach((h) => {
    ;(h as SVGElement).style.setProperty('--di', String(i + 6))
    h.classList.add('fade-el')
  })
  return 650 + i * 15
}

export function replayActiveEdges(svg: SVGSVGElement, step: SketchStep) {
  const sel = step.edges.map((id) => `.edge[data-id="${id}"] path`).join(',')
  if (!sel) return
  const paths = svg.querySelectorAll(sel)
  paths.forEach((p) => p.classList.remove('draw-in'))
  void document.body.offsetWidth
  paths.forEach((p, k) => {
    ;(p as SVGElement).style.setProperty('--di', String(k * 4))
    p.classList.add('draw-in')
  })
}

export function renderGatewayDiagram(
  container: HTMLElement,
  nodes: SketchNode[],
  edges: SketchEdge[],
  groups: {
    x: number
    y: number
    w: number
    h: number
    label: string
    color?: string
  }[],
  viewBox: string,
  hasRough: boolean
): SVGSVGElement {
  const svg = svgEl('svg', {
    viewBox,
    role: 'img',
  }) as SVGSVGElement
  const ctx = createSketchContext(svg, hasRough)

  for (const g of groups) {
    svg.appendChild(ctx.groupArea(g))
  }

  for (const e of edges) {
    const from = nodes.find((n) => n.id === e.from)
    const to = nodes.find((n) => n.id === e.to)
    if (!from || !to) continue
    const a = rightOf(from)
    const b = leftOf(to)
    svg.appendChild(
      ctx.edgeLine({
        ...e,
        x1: a.x + 4,
        y1: a.y,
        x2: b.x - 4,
        y2: b.y,
      })
    )
  }

  for (const n of nodes) {
    svg.appendChild(ctx.nodeBox(n))
  }

  container.replaceChildren(svg)
  return svg
}
