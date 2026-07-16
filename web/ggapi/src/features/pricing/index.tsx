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
import { useNavigate } from '@tanstack/react-router'
import { useCallback, useMemo } from 'react'
import { useTranslation } from 'react-i18next'

import {
  PublicLayout,
  PublicPageHeader,
  PublicPageShell,
} from '@/components/layout'

import {
  LoadingSkeleton,
  EmptyState,
  PricingTable,
  PricingToolbar,
  PricingSidebar,
  ModelCardGrid,
} from './components'
import {
  EXCLUDED_GROUPS,
  PRICING_FILTER_LAYOUT_CLASS,
  PRICING_PAGE_SHELL_CLASS,
  PRICING_SIDEBAR_STICKY_CLASS,
  VIEW_MODES,
} from './constants'
import { useFilters } from './hooks/use-filters'
import { usePricingData } from './hooks/use-pricing-data'
import { hasDistinctRechargePrice } from './lib/price'

export function Pricing() {
  const { t } = useTranslation()
  const navigate = useNavigate({ from: '/pricing/' })

  const {
    models,
    vendors,
    groupRatio,
    usableGroup,
    isLoading,
    priceRate,
    usdExchangeRate,
  } = usePricingData()

  const {
    searchInput,
    sortBy,
    vendorFilter,
    groupFilter,
    quotaTypeFilter,
    endpointTypeFilter,
    tagFilter,
    viewMode,
    showRechargePrice: rechargePriceFromUrl,
    setSearchInput,
    setSortBy,
    setVendorFilter,
    setGroupFilter,
    setQuotaTypeFilter,
    setEndpointTypeFilter,
    setTagFilter,
    setViewMode,
    setShowRechargePrice,
    filteredModels,
    hasActiveFilters,
    activeFilterCount,
    availableTags,
    routeSearch: filtersRouteSearch,
    clearFilters,
    clearSearch,
  } = useFilters(models || [])

  const rechargeDistinct = hasDistinctRechargePrice(priceRate, usdExchangeRate)
  // Single truth: only honor recharge when modes actually differ.
  const showRechargePrice = rechargeDistinct && rechargePriceFromUrl
  // Drop no-op recharge from navigable search (no effect write-back needed).
  const routeSearch = useMemo(
    () => ({
      ...filtersRouteSearch,
      rechargePrice: showRechargePrice || undefined,
    }),
    [filtersRouteSearch, showRechargePrice]
  )

  const handleModelClick = useCallback(
    (modelName: string) => {
      navigate({
        to: '/pricing/$modelId',
        params: { modelId: modelName },
        search: routeSearch,
      })
    },
    [navigate, routeSearch]
  )

  const availableGroups = useMemo(
    () =>
      Object.keys(usableGroup || {}).filter(
        (g) => !EXCLUDED_GROUPS.includes(g)
      ),
    [usableGroup]
  )

  const handleClearAll = useCallback(() => {
    clearFilters()
    clearSearch()
  }, [clearFilters, clearSearch])

  const filterSidebarProps = {
    quotaTypeFilter,
    endpointTypeFilter,
    vendorFilter,
    groupFilter,
    tagFilter,
    onQuotaTypeChange: setQuotaTypeFilter,
    onEndpointTypeChange: setEndpointTypeFilter,
    onVendorChange: setVendorFilter,
    onGroupChange: setGroupFilter,
    onTagChange: setTagFilter,
    vendors: vendors || [],
    groups: availableGroups,
    groupRatios: groupRatio,
    tags: availableTags,
    models: models || [],
    hasActiveFilters,
    onClearFilters: clearFilters,
  }

  // Build list body only when data is ready (and avoid nested ternaries for lint).
  let listContent = null
  if (!isLoading) {
    if (filteredModels.length === 0) {
      listContent = (
        <EmptyState
          searchQuery={searchInput}
          hasActiveFilters={hasActiveFilters}
          onClearFilters={handleClearAll}
        />
      )
    } else if (viewMode === VIEW_MODES.CARD) {
      listContent = (
        <ModelCardGrid
          models={filteredModels}
          onModelClick={handleModelClick}
          priceRate={priceRate}
          usdExchangeRate={usdExchangeRate}
          showRechargePrice={showRechargePrice}
          selectedGroup={groupFilter}
        />
      )
    } else {
      listContent = (
        <PricingTable
          models={filteredModels}
          priceRate={priceRate}
          usdExchangeRate={usdExchangeRate}
          showRechargePrice={showRechargePrice}
          selectedGroup={groupFilter}
          onModelClick={handleModelClick}
        />
      )
    }
  }

  return (
    <PublicLayout showMainContainer={false}>
      <PublicPageShell className={PRICING_PAGE_SHELL_CLASS}>
        {isLoading ? (
          <LoadingSkeleton viewMode={viewMode} />
        ) : (
          <>
            <PublicPageHeader
              title={t('Models & pricing')}
              description={
                <>
                  {t(
                    'Discover curated AI models, compare pricing and capabilities, and choose the right model for every scenario.'
                  )}{' '}
                  {t('This site currently has {{count}} models enabled', {
                    count: models?.length || 0,
                  })}
                </>
              }
            />

            <div className={PRICING_FILTER_LAYOUT_CLASS}>
              <PricingSidebar
                {...filterSidebarProps}
                className={PRICING_SIDEBAR_STICKY_CLASS}
              />

              <main className='min-w-0 space-y-4'>
                <PricingToolbar
                  {...filterSidebarProps}
                  searchInput={searchInput}
                  onSearchChange={setSearchInput}
                  onClearSearch={clearSearch}
                  filteredCount={filteredModels.length}
                  totalCount={models?.length}
                  sortBy={sortBy}
                  onSortChange={setSortBy}
                  showRechargePrice={showRechargePrice}
                  onRechargePriceChange={setShowRechargePrice}
                  priceRate={priceRate}
                  usdExchangeRate={usdExchangeRate}
                  viewMode={viewMode}
                  onViewModeChange={setViewMode}
                  activeFilterCount={activeFilterCount}
                />

                {listContent}
              </main>
            </div>
          </>
        )}
      </PublicPageShell>
    </PublicLayout>
  )
}
