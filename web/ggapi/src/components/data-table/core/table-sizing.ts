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
import type { Table as TanstackTable } from '@tanstack/react-table'
import type * as React from 'react'

import { getColumnMinContributionForColumn } from './column-track-style'

export function getTableSizeStyle<TData>(
  table: TanstackTable<TData>
): React.CSSProperties {
  // Sum preferred/min widths so the table scrolls horizontally instead of
  // crushing badges, names, or action controls when the viewport is narrow.
  // table-layout: auto keeps nowrap headers/cells in the intrinsic track
  // (fixed layout would ignore content and clip or overlap).
  const width = table
    .getVisibleLeafColumns()
    .reduce(
      (total, column) =>
        total + getColumnMinContributionForColumn(table, column),
      0
    )

  return {
    minWidth: width > 0 ? `max(100%, ${width}px)` : '100%',
    tableLayout: 'auto',
    width: '100%',
  }
}
