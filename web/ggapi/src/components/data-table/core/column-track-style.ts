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
import type {
  Column,
  ColumnDef,
  Table as TanstackTable,
} from '@tanstack/react-table'
import type * as React from 'react'

import { isContentSizedColumn } from './content-sized-columns'

/** Fallback width for the actions track when size is undeclared. */
export const ACTIONS_COLUMN_FALLBACK_SIZE = 96

type AuthorSizing = {
  size?: number
  minSize?: number
}

/**
 * Read size/minSize from the author-supplied column defs (table.options.columns),
 * not from the merged columnDef (which always includes TanStack defaults of
 * size=150 and minSize=20).
 */
export function getAuthorColumnSizing<TData>(
  table: TanstackTable<TData>,
  columnId: string
): AuthorSizing {
  const original = findAuthorColumnDef(table.options.columns, columnId)
  if (!original) {
    return {}
  }

  return {
    size: typeof original.size === 'number' ? original.size : undefined,
    minSize:
      typeof original.minSize === 'number' ? original.minSize : undefined,
  }
}

function findAuthorColumnDef<TData>(
  columns: ColumnDef<TData, unknown>[] | undefined,
  columnId: string
): ColumnDef<TData, unknown> | undefined {
  if (!columns?.length) {
    return undefined
  }

  for (const column of columns) {
    const id = resolveAuthorColumnId(column)
    if (id === columnId) {
      return column
    }

    const nested = (
      column as ColumnDef<TData, unknown> & {
        columns?: ColumnDef<TData, unknown>[]
      }
    ).columns
    if (nested?.length) {
      const found = findAuthorColumnDef(nested, columnId)
      if (found) {
        return found
      }
    }
  }

  return undefined
}

function resolveAuthorColumnId<TData>(
  column: ColumnDef<TData, unknown>
): string | undefined {
  if (typeof column.id === 'string' && column.id.length > 0) {
    return column.id
  }

  const accessorKey = (column as { accessorKey?: unknown }).accessorKey
  if (typeof accessorKey === 'string' && accessorKey.length > 0) {
    return accessorKey.includes('.')
      ? accessorKey.replaceAll('.', '_')
      : accessorKey
  }

  if (typeof column.header === 'string' && column.header.length > 0) {
    return column.header
  }

  return undefined
}

export function resolveActionsSize(
  columnSize: number,
  authorSize: number | undefined
): number {
  // Prefer an author-declared size (including an explicit 150). Only fall back
  // when the definition omitted `size` and TanStack filled in the default.
  if (typeof authorSize === 'number' && authorSize > 0) {
    return authorSize
  }
  if (columnSize > 0 && typeof authorSize === 'number') {
    // authorSize is 0 or negative — treat as missing.
    return ACTIONS_COLUMN_FALLBACK_SIZE
  }
  return ACTIONS_COLUMN_FALLBACK_SIZE
}

export function getColumnMinContribution(props: {
  columnId: string
  columnSize: number
  authorSize?: number
  authorMinSize?: number
  contentSized?: boolean
}): number {
  const { columnId, columnSize, authorSize, authorMinSize, contentSized } =
    props

  if (columnId === 'actions') {
    return resolveActionsSize(columnSize, authorSize)
  }

  if (isContentSizedColumn(columnId, contentSized)) {
    if (typeof authorMinSize === 'number' && authorMinSize > 0) {
      return authorMinSize
    }
    if (typeof authorSize === 'number' && authorSize > 0) {
      return authorSize
    }
    if (columnSize > 0) {
      return columnSize
    }
    return 80
  }

  return columnSize > 0 ? columnSize : 0
}

/**
 * Style for `<col>` / header / body cells that participate in track sizing.
 *
 * Compact columns use absolute pixel width+minWidth. Content-sized columns
 * with an author minSize use the same width hint (browsers ignore min-width
 * on `<col>`); under table-layout:auto the track can still grow. Actions are
 * a bounded chrome track.
 */
export function getColumnTrackStyle(props: {
  columnId: string
  columnSize: number
  authorSize?: number
  authorMinSize?: number
  contentSized?: boolean
}): React.CSSProperties | undefined {
  const { columnId, columnSize, authorSize, authorMinSize, contentSized } =
    props

  if (columnId === 'actions') {
    const size = resolveActionsSize(columnSize, authorSize)
    return { width: `${size}px`, minWidth: `${size}px` }
  }

  if (isContentSizedColumn(columnId, contentSized)) {
    // Author floor must be a real `width` hint — `min-width` on `<col>` is
    // ignored by browsers for table-column layout. Auto layout may still
    // grow the track when there is leftover space.
    if (typeof authorMinSize === 'number' && authorMinSize > 0) {
      return { width: `${authorMinSize}px`, minWidth: `${authorMinSize}px` }
    }
    return undefined
  }

  if (columnSize > 0) {
    return { width: `${columnSize}px`, minWidth: `${columnSize}px` }
  }

  return undefined
}

export function getColumnTrackStyleForColumn<TData>(
  table: TanstackTable<TData>,
  column: Column<TData, unknown>
): React.CSSProperties | undefined {
  const author = getAuthorColumnSizing(table, column.id)
  return getColumnTrackStyle({
    columnId: column.id,
    columnSize: column.getSize(),
    authorSize: author.size,
    authorMinSize: author.minSize,
    contentSized: column.columnDef.meta?.contentSized,
  })
}

export function getColumnMinContributionForColumn<TData>(
  table: TanstackTable<TData>,
  column: Column<TData, unknown>
): number {
  const author = getAuthorColumnSizing(table, column.id)
  return getColumnMinContribution({
    columnId: column.id,
    columnSize: column.getSize(),
    authorSize: author.size,
    authorMinSize: author.minSize,
    contentSized: column.columnDef.meta?.contentSized,
  })
}
