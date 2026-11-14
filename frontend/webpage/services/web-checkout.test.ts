import { describe, expect, it } from 'bun:test';
import {
  DELIVERY_HOME, DELIVERY_PICKUP, FLAG_RECEIPT_AUTOMATIC, FLAG_REQUIRE_CONTACT_NUMBER, NEW_ADDRESS_ID,
  PAYMENT_CARD, PAYMENT_DEFERRED, buildWebCheckoutRequest, cartTotalCents, defaultAddressID,
  formatWebOrderNumber, mostExpensiveOrderLines, requiresDocumentNumber, validateCustomerDraft,
  validateDelivery, webOrderPaymentLabel, webOrderProgress, webOrderProgressStep, webPaymentMethodLabel,
  type ICustomerDraft,
} from './web-checkout';
import { QUANTITY_LINE_SCALE } from '#core/quantity.ts';

describe('Mis Pedidos card', () => {
  it('pads the order number to 4 digits', () => {
    expect(formatWebOrderNumber(1)).toBe('0001');
    expect(formatWebOrderNumber(12345)).toBe('12345');
  });

  it('lists the lines with the highest amount first', () => {
    const line = (ProductID: number, units: number, Price: number) => ({ ProductID, Price, Quantity: units * QUANTITY_LINE_SCALE });
    const topLines = mostExpensiveOrderLines([line(1, 10, 100), line(2, 1, 5000), line(3, 3, 700), line(4, 1, 50)], 3);
    expect(topLines.map(topLine => topLine.ProductID)).toEqual([2, 3, 1]);
  });

  it('labels the payment apart from the delivery', () => {
    expect(webOrderPaymentLabel(1)).toBe('Pendiente de pago');
    expect(webOrderPaymentLabel(3)).toBe('Pendiente de pago');
    expect(webOrderPaymentLabel(4)).toBe('Pagado');
    expect(webOrderPaymentLabel(0)).toBe('');
  });

  it('badges the progress, a delivery outranking the fulfillment label', () => {
    const order = (ss: number, FulfillmentStatus: number, DeliveryType = DELIVERY_HOME) => ({ ss, FulfillmentStatus, DeliveryType });
    expect(webOrderProgress(order(2, 0)).label).toBe('Recibido');
    expect(webOrderProgress(order(2, 1)).label).toBe('En Preparación');
    expect(webOrderProgress(order(2, 2)).label).toBe('En Camino');
    expect(webOrderProgress(order(2, 2, DELIVERY_PICKUP)).label).toBe('Para Recoger');
    expect(webOrderProgress(order(4, 2)).label).toBe('Entregado');
    expect(webOrderProgress(order(0, 1)).label).toBe('Anulado');
    expect(webOrderProgressStep(order(3, 1))).toBe(3);
    expect(webOrderProgressStep(order(1, 0))).toBe(0);
  });

  it('names the payment method, cash on delivery by where it is paid', () => {
    expect(webPaymentMethodLabel(PAYMENT_CARD, DELIVERY_HOME)).toBe('Tarjeta');
    expect(webPaymentMethodLabel(PAYMENT_DEFERRED, DELIVERY_HOME)).toBe('Pago contra entrega');
    expect(webPaymentMethodLabel(PAYMENT_DEFERRED, DELIVERY_PICKUP)).toBe('Pago en tienda');
  });
});

const completeDraft: ICustomerDraft = {
  Email: 'ana@example.com', FirstName: 'Ana', LastName: 'Pérez', Phone: '', DocumentNumber: '',
};

describe('requiresDocumentNumber', () => {
  it('asks the DNI from S/ 700 only when receipts are automatic', () => {
    expect(requiresDocumentNumber([FLAG_RECEIPT_AUTOMATIC], 70000)).toBe(true);
    expect(requiresDocumentNumber([FLAG_RECEIPT_AUTOMATIC], 69999)).toBe(false);
    expect(requiresDocumentNumber([], 90000)).toBe(false);
  });
});

describe('validateCustomerDraft', () => {
  it('accepts a complete contact', () => {
    expect(validateCustomerDraft(completeDraft, [], false)).toBe('');
  });
  it('requires the phone only under flag 11', () => {
    expect(validateCustomerDraft(completeDraft, [FLAG_REQUIRE_CONTACT_NUMBER], false)).not.toBe('');
    expect(validateCustomerDraft({ ...completeDraft, Phone: '987654321' }, [FLAG_REQUIRE_CONTACT_NUMBER], false)).toBe('');
  });
  it('requires an 8-digit DNI when asked', () => {
    expect(validateCustomerDraft(completeDraft, [], true)).not.toBe('');
    expect(validateCustomerDraft({ ...completeDraft, DocumentNumber: '1234567' }, [], true)).not.toBe('');
    expect(validateCustomerDraft({ ...completeDraft, DocumentNumber: '12345678' }, [], true)).toBe('');
  });
  it('rejects a malformed email', () => {
    expect(validateCustomerDraft({ ...completeDraft, Email: 'ana@' }, [], false)).not.toBe('');
  });
});

describe('validateDelivery', () => {
  const newAddress = { CityID: 150101, Address: 'Av. Lima 123' };
  it('needs a pickup store on pickup', () => {
    expect(validateDelivery(DELIVERY_PICKUP, 0, NEW_ADDRESS_ID, newAddress)).not.toBe('');
    expect(validateDelivery(DELIVERY_PICKUP, 4, NEW_ADDRESS_ID, newAddress)).toBe('');
  });
  it('checks the typed address only for "Nueva dirección"', () => {
    expect(validateDelivery(DELIVERY_HOME, 0, NEW_ADDRESS_ID, { CityID: 15, Address: 'Av. Lima 123' })).not.toBe('');
    expect(validateDelivery(DELIVERY_HOME, 0, NEW_ADDRESS_ID, newAddress)).toBe('');
    expect(validateDelivery(DELIVERY_HOME, 0, 7, { CityID: 0, Address: '' })).toBe('');
  });
  it('requires a delivery type', () => {
    expect(validateDelivery(0, 0, 0, newAddress)).not.toBe('');
  });
});

describe('defaultAddressID', () => {
  const addresses = [{ ID: 3, CityID: 1, Address: 'a', Reference: '' }, { ID: 9, CityID: 1, Address: 'b', Reference: '' }];
  it('prefers the last used address, then the newest, then a new one', () => {
    expect(defaultAddressID(addresses, 3)).toBe(3);
    expect(defaultAddressID(addresses)).toBe(9);
    expect(defaultAddressID([])).toBe(NEW_ADDRESS_ID);
  });
});

describe('buildWebCheckoutRequest', () => {
  const baseArgs = {
    cartLines: [{ productID: 5, quantity: 2, unitPriceCents: 1500 }],
    draft: { ...completeDraft, Email: ' ana@example.com ' },
    pickupSiteID: 4,
    newAddress: { CityID: 150101, Address: ' Av. Lima 123 ', Reference: 'Frente al parque' },
    culqiToken: 'tkn_test',
  };
  it('sends the typed address on "Nueva dirección" and the token on online payment', () => {
    const request = buildWebCheckoutRequest({
      ...baseArgs, deliveryType: DELIVERY_HOME, selectedAddressID: NEW_ADDRESS_ID, paymentMethod: PAYMENT_CARD,
    });
    expect(request.PickupSiteID).toBe(0);
    expect(request.Shipping.CityID).toBe(150101);
    expect(request.Shipping.Address).toBe('Av. Lima 123');
    expect(request.Shipping.Email).toBe('ana@example.com');
    expect(request.CulqiToken).toBe('tkn_test');
    expect(request.Lines).toEqual([{ ProductID: 5, PresentationID: 0, Quantity: 2 }]);
  });
  it('drops the address and the token on a deferred pickup', () => {
    const request = buildWebCheckoutRequest({
      ...baseArgs, deliveryType: DELIVERY_PICKUP, selectedAddressID: 7, paymentMethod: PAYMENT_DEFERRED,
    });
    expect(request.PickupSiteID).toBe(4);
    expect(request.AddressID).toBe(0);
    expect(request.Shipping.CityID).toBe(0);
    expect(request.CulqiToken).toBe('');
  });
});

describe('cartTotalCents', () => {
  it('sums price × quantity', () => {
    expect(cartTotalCents([{ productID: 1, quantity: 3, unitPriceCents: 250 }, { productID: 2, quantity: 1, unitPriceCents: 100 }])).toBe(850);
  });
});
