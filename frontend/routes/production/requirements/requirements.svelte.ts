import { SvelteSet } from 'svelte/reactivity'
import { tr } from '#core/store.svelte.ts'
import { POST } from '#libs/ui-runtime.svelte.ts'
import { PurchaseOrdersService, PurchaseOrderStatus } from '#routes/logistics/purchase-orders/purchase_order.svelte.ts'
import { ProductSupplyService } from '#routes/logistics/purchase-management/supply-management.svelte.ts'
import { expandTemplateToOrderLines } from '#routes/production/orders/production-order.ts'
import { ProductionCatalogs } from '#routes/production/orders/production-order.svelte.ts'
import { PRODUCTION_ORDER_STATUS } from '#services/production/production-orders.ts'
import { postProductionOrder } from '#services/production/production-orders.svelte.ts'
import {
  buildPurchaseSuggestions, buildSemiFinishedDrafts, computeRequirements, groupPurchaseOrdersByProvider,
  type IDemandBatch, type IProviderPrice, type IRequirementSupply, type ISupplyRequirement,
} from './requirements.ts'

// The page's inputs, the services it reads (no endpoint of its own, §7) and the two writes it makes
// through the existing routes: purchase orders per provider, and draft production orders.
export class RequirementsState {
  // Orders, templates, made products, supplies, stock rows, warehouses and the providers (workshops).
  catalogs = new ProductionCatalogs(true)
  productSupplies = new ProductSupplyService(true)
  confirmedPurchaseOrders = new PurchaseOrdersService(PurchaseOrderStatus.CONFIRMED, true)

  form = $state({ WarehouseID: 0, TemplateID: 0, Series: 0, ReplenishToMinimum: false })
  selectedOrderIDs = new SvelteSet<number>()
  isSaving = $state(false)
  // The plan the last write was made for: its button stays off until the plan changes, so a second
  // click does not order the same supplies twice.
  purchasedPlan = $state.raw<ISupplyRequirement[]>()
  draftedPlan = $state.raw<ISupplyRequirement[]>()

  draftOrders = $derived((this.catalogs.orders?.records || []).filter((order) => order.ss === PRODUCTION_ORDER_STATUS.DRAFT))

  // Each picked draft keeps its own supply adjustment; the template × series has none.
  demandBatches = $derived.by((): IDemandBatch[] => {
    const batches: IDemandBatch[] = this.draftOrders.filter((order) => this.selectedOrderIDs.has(order.ID))
      .map((order) => ({ lines: order.Lines || [], adjustmentBasisPoints: order.SupplyAdjustmentBasisPoints || 0 }))
    const template = this.catalogs.templates.recordsMap.get(this.form.TemplateID)
    if (template && this.form.Series > 0) batches.push({ lines: expandTemplateToOrderLines(template.Lines, this.form.Series), adjustmentBasisPoints: 0 })
    return batches
  })

  providerOffersBySupplyID = $derived(new Map<number, IProviderPrice[]>(
    this.productSupplies.records.map((productSupply) => [productSupply.ProductID, productSupply.ProviderSupply || []])))

  requirements = $derived.by((): ISupplyRequirement[] => {
    if (!(this.form.WarehouseID > 0) || !this.demandBatches.length) return []
    return computeRequirements({
      batches: this.demandBatches,
      warehouseID: this.form.WarehouseID,
      recipeProductsByID: this.catalogs.recipeProductsByID,
      suppliesByID: this.catalogs.supplies.recordsMap as Map<number, IRequirementSupply>,
      stockRows: this.catalogs.stockRows.records,
      purchaseOrders: this.confirmedPurchaseOrders.records,
      minimumStockBySupplyID: this.form.ReplenishToMinimum
        ? new Map(this.productSupplies.records.map((productSupply) => [productSupply.ProductID, productSupply.MinimunStock || 0]))
        : undefined,
    })
  })

  // Rebuilt when the plan changes; the page edits units, provider and price on these rows in place.
  purchaseSuggestions = $derived(buildPurchaseSuggestions(this.requirements, this.providerOffersBySupplyID))
  semiFinishedShortfalls = $derived(this.requirements.filter((requirement) => requirement.IsSemiFinished && requirement.ShortfallUnits > 0))

  // "Crear órdenes de compra": one POST purchase-orders per provider, created Pending. Answers the IDs
  // created; a failure stops there and is thrown with what was already created.
  async createPurchaseOrders(): Promise<number[]> {
    const createdOrderIDs: number[] = []
    this.isSaving = true
    try {
      for (const purchaseOrder of groupPurchaseOrdersByProvider(this.purchaseSuggestions, this.form.WarehouseID)) {
        const savedOrder = await POST({ data: purchaseOrder, route: 'purchase-orders', refreshRoutes: ['purchase-orders'] })
        createdOrderIDs.push(savedOrder?.ID || 0)
      }
      this.purchasedPlan = this.requirements
    } catch (postError) {
      console.error('[requirements] purchase orders error', { createdOrderIDs, postError })
      throw createdOrderIDs.length ? `${postError} ${tr(`(created before: ${createdOrderIDs.join(', ')})|(ya creadas: ${createdOrderIDs.join(', ')})`)}` : postError
    } finally {
      this.isSaving = false
    }
    return createdOrderIDs
  }

  // "Crear órdenes": one draft production order per semi-finished shortfall, through POST
  // production-order. The drafts then show in the order list, ready to be picked here too.
  async createSemiFinishedDrafts(): Promise<number[]> {
    const orders = this.catalogs.orders!
    const createdOrderIDs: number[] = []
    this.isSaving = true
    try {
      for (const draft of buildSemiFinishedDrafts(this.requirements, this.form.WarehouseID, orders.records)) {
        const savedOrder = await postProductionOrder(orders, draft)
        createdOrderIDs.push(savedOrder?.ID || 0)
      }
      this.draftedPlan = this.requirements
    } catch (postError) {
      console.error('[requirements] production drafts error', { createdOrderIDs, postError })
      throw createdOrderIDs.length ? `${postError} ${tr(`(created before: ${createdOrderIDs.join(', ')})|(ya creadas: ${createdOrderIDs.join(', ')})`)}` : postError
    } finally {
      this.isSaving = false
    }
    return createdOrderIDs
  }
}
