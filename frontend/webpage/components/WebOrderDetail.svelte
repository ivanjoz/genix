<script lang="ts">
  import StepsTimeline from '#components/navigation/StepsTimeline.svelte';
  import { notifyFailure } from '@genix/ui/notify';
  import { unpackQuantityLine } from '#core/quantity.ts';
  import { WebStore, downloadWebOrderPDF, type IWebOrder } from "#ecommerce/services/web-store.svelte.ts";
  import {
    DELIVERY_PICKUP, formatSoles, webOrderPaymentLabel, webOrderProgressStep, webOrderProgressSteps,
    webPaymentMethodLabel,
  } from "#ecommerce/services/web-checkout.ts";

  // The detail of one "Mis Pedidos" order, shown in the side layer of its card: progress, delivery,
  // payment and every line with its amount.
  interface IProps {
    order: IWebOrder;
    productNames: Record<number, string>;
    cityName: (cityID: number) => string;
  }

  let { order, productNames, cityName }: IProps = $props();

  const isPickup = $derived(order.DeliveryType === DELIVERY_PICKUP);
  const isPaid = $derived(order.ss === 2 || order.ss === 4);
  const pickupSite = $derived((WebStore.company.PickupSites || []).find((site) => site.ID === order.PickupSiteID));
  const contactName = $derived(`${order.Shipping?.FirstName || ''} ${order.Shipping?.LastName || ''}`.trim());
</script>

{#snippet sectionTitle(title: string)}
  <div class="ff-bold text-gray-800 mb-6">{title}</div>
{/snippet}

<div class="flex flex-col gap-18 text-[14px] text-gray-700">
  <!-- Progress, read-only for the shopper; an annulled order has no steps. -->
  {#if order.ss === 0}
    <div class="px-12 py-8 rounded-[8px] bg-red-50 text-red-700 ff-semibold">Este pedido fue anulado.</div>
  {:else}
    <StepsTimeline steps={webOrderProgressSteps(order.DeliveryType)} currentID={webOrderProgressStep(order)} hideMarker />
  {/if}

  <div>
    {@render sectionTitle(isPickup ? "Recojo en tienda" : "Entrega a domicilio")}
    {#if isPickup}
      <div>{pickupSite?.Name || "Tienda de recojo"}</div>
      {#if pickupSite}
        <div class="text-gray-500">{[pickupSite.Address, cityName(pickupSite.CityID)].filter(Boolean).join(" · ")}</div>
      {/if}
    {:else}
      <div>{order.Shipping?.Address}</div>
      <div class="text-gray-500">{[cityName(order.Shipping?.CityID), order.Shipping?.Reference].filter(Boolean).join(" · ")}</div>
    {/if}
  </div>

  <div>
    <!-- Who receives the order on the left, how it is paid on the right. -->
    <div class="grid grid-cols-[1fr_auto] gap-x-16 items-start px-14 py-10 rounded-[8px] bg-gray-100">
      <div class="flex gap-8 min-w-0">
        <i class="icon-[fa--user] shrink-0 mt-3 text-gray-500"></i>
        <div class="min-w-0">
          <div class="ff-semibold text-gray-800">{contactName || "Sin nombre"}</div>
          {#if order.Shipping?.Phone}<div class="text-gray-500">{order.Shipping.Phone}</div>{/if}
        </div>
      </div>
      <div class="flex flex-wrap items-center justify-end gap-x-8 text-right">
        {#if webOrderPaymentLabel(order.ss)}
          <span class="ff-semibold {isPaid ? 'text-green-700' : 'text-amber-700'}">{webOrderPaymentLabel(order.ss)}</span>
          <span class="w-5 h-5 rounded-full bg-gray-300"></span>
        {/if}
        <span>{webPaymentMethodLabel(order.PaymentMethod, order.DeliveryType)}</span>
      </div>
    </div>
    <!-- The Orden de Compra is what the shopper shows to collect an order paid later. -->
    {#if order.ss === 1 && order.DebtAmount > 0}
      <button class="mt-6 text-[#4c55d5] hover:underline cursor-pointer"
        onclick={() => downloadWebOrderPDF(order.ID).catch((error) => notifyFailure(error.message))}>
        <i class="icon-[fa--download] mr-4"></i>Orden de compra (PDF)
      </button>
    {/if}
  </div>

  <div>
    {@render sectionTitle("Productos")}
    <div class="grid grid-cols-[1fr_auto] gap-x-16 gap-y-6 items-baseline">
      {#each order.Lines || [] as line}
        {@const units = unpackQuantityLine(line.Quantity).units}
        <div class="min-w-0">
          <div>{productNames[line.ProductID] || `Producto ${line.ProductID}`}</div>
          <div class="text-gray-500">{units} × {formatSoles(line.Price)}</div>
        </div>
        <div class="text-right ff-semibold">{formatSoles(units * line.Price)}</div>
      {/each}
    </div>
    <!-- TotalAmount already includes the shipping, so the subtotal is what is left without it. -->
    <div class="flex flex-col gap-4 mt-12 pt-10 border-t border-gray-200">
      <div class="flex justify-between"><span>Subtotal</span><span>{formatSoles(order.TotalAmount - order.ShippingAmount)}</span></div>
      {#if !isPickup}
        <div class="flex justify-between"><span>Envío</span><span>{formatSoles(order.ShippingAmount)}</span></div>
      {/if}
      <div class="flex justify-between ff-bold text-gray-800 text-[16px]"><span>Total</span><span>{formatSoles(order.TotalAmount)}</span></div>
    </div>
  </div>
</div>
