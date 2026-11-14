import { describe, expect, it } from 'vitest'
import { DOC_TYPE_CREDIT_NOTE, DOC_TYPE_FACTURA } from '#core/sunat-doc-type.ts'
import { periodBounds, type ISalesBookDocument } from '../books/books'
import { PURCHASE_SOURCE_EXPENSE, PURCHASE_SOURCE_ORDER, type IPurchaseBookDocument } from '../books/books.purchases'
import {
  buildAccountingOperations, buildTrialBalance, bulkClassificationTargets,
  classificationTargetsOf, entrySides, groupOperations, inferSameDayPayments, referencedRecordIDs,
  summarizeOperations, OPERATION_CREDIT_NOTE, OPERATION_STATUS_CLASSIFIED,
  OPERATION_STATUS_INCOMPLETE, OPERATION_STATUS_UNCLASSIFIED,
  type IAccountingOperation, type IAccountingSources, type IOperationAsset,
  type IOperationsContext,
} from './accounts-consolidation'
import { splitByWeights } from './accounts-consolidation.utils'

const FIRST_DAY = 20697 // 2026-09-01
const DAY = FIRST_DAY + 3
const SALE_ID = 7_000_101     // series 01
const UNINVOICED_SALE_ID = 7_000_200
const MADE_PRODUCT = 41
const BOUGHT_PRODUCT = 42
const SUPPLY = 43 // default account 6032 → stock 252

const context: IOperationsContext = {
  seriesByID: new Map([
    [1, { SeriesID: 1, DocType: DOC_TYPE_FACTURA, SeriesCode: "F001" }],
    [2, { SeriesID: 2, DocType: DOC_TYPE_CREDIT_NOTE, SeriesCode: "FC01" }],
  ]),
}

const makeResponse = (fields: Partial<IAccountingSources> = {}): IAccountingSources => ({
  FirstDay: FIRST_DAY, LastDay: FIRST_DAY + 29,
  SalesDocuments: [], Sales: [], PurchaseOrders: [], Expenses: [], ReferencedSales: [], ReferencedPurchases: [],
  Assets: [], StockMovements: [], CashBanks: [{ ID: 1, Name: "Caja", Type: 1, CurrencyType: 1 }], CashMovements: [],
  SellRates: Array(30).fill(3750),
  Products: [
    { ID: MADE_PRODUCT, ss: 1, Origin: 2, AccountCode: 0 },
    { ID: BOUGHT_PRODUCT, ss: 1, Origin: 1, AccountCode: 0 },
    { ID: SUPPLY, ss: 2, Origin: 0, AccountCode: 6032 },
  ],
  ...fields,
})

const makeSaleMovement = (Type: number, ProductID: number, MonetaryValue: number, DocumentID = SALE_ID) =>
  ({ ID: Type * 100 + ProductID, Date: DAY, Type, DocumentID, ProductID, MonetaryValue })

const makeInvoice = (fields: Partial<ISalesBookDocument> = {}): ISalesBookDocument => ({
  ID: SALE_ID, IssueDate: DAY, Correlativo: 15, ClientSnapshotID: 0, Currency: 1,
  TaxableAmount: 10000, TaxAmount: 1800, ExemptAmount: 0, UnaffectedAmount: 0, TotalAmount: 11800,
  AffectedDocID: 0, State: 3,
  ...fields,
})

const makePurchase = (fields: Partial<IPurchaseBookDocument> = {}): IPurchaseBookDocument => ({
  Source: PURCHASE_SOURCE_EXPENSE, SourceID: 5, Name: "Alquiler", ProviderID: 9, ProviderSnapshotID: 0,
  DueDate: 0, RecordAmount: 11800, AccountCode: 6352, ExpenseType: 1,
  DocType: 1, DocSeries: "F002", DocNumber: 88, DocIssueDate: DAY,
  TaxableAmount: 10000, TaxAmount: 1800, UntaxedAmount: 0, OtherAmount: 0, CurrencyType: 1, ExchangeRate: 0,
  ...fields,
})

// A depreciation entry travels in the Expenses key: no comprobante, its asset and its own date.
const makeDepreciation = (AssetID: number, Date: number, RecordAmount: number): IPurchaseBookDocument => makePurchase({
  SourceID: 90, Name: "", ExpenseType: 4, AssetID, Date, RecordAmount, AccountCode: 0,
  DocType: 0, DocSeries: "", DocNumber: 0, DocIssueDate: 0, TaxableAmount: 0, TaxAmount: 0,
})

const makeAsset = (fields: Partial<IOperationAsset> = {}): IOperationAsset => ({
  ID: 3, Name: "Laptop", SupplierID: 9, PurchaseAmount: 0, AcquisitionValue: 300000, AcquisitionDate: DAY,
  DisposalDate: 0, AccumulatedDepreciation: 0, AccountCode: 3361, ss: 1, LastDepreciationDate: 0,
  DocType: 0, DocSeries: "", DocNumber: 0, DocIssueDate: 0, TaxableAmount: 0, TaxAmount: 0,
  UntaxedAmount: 0, OtherAmount: 0, CurrencyType: 1, ExchangeRate: 0,
  ...fields,
})

// The entry as "account: debit/credit" pairs, so an assertion reads like the Diario.
const entryOf = (operation: IAccountingOperation): string[] =>
  operation.Lines.map(line => `${line.AccountCode}:${line.Debit}/${line.Credit}`)

const onlyOperation = (response: IAccountingSources): IAccountingOperation => {
  const operations = buildAccountingOperations(response, context)
  expect(operations).toHaveLength(1)
  return operations[0]
}

describe('splitByWeights', () => {
  it('puts the rounding remainder on the last share', () => {
    expect(splitByWeights(1000, [1, 1, 1])).toEqual([333, 333, 334])
  })
  it('gives everything to the last share when no weight is positive', () => {
    expect(splitByWeights(500, [0, 0])).toEqual([0, 500])
  })
})

describe('sales', () => {
  it('splits a factura between 7012 and 7022 by the products origin', () => {
    const operation = onlyOperation(makeResponse({
      SalesDocuments: [makeInvoice()],
      Sales: [{
        ID: SALE_ID, Date: DAY, ss: 2, CurrencyType: 1, ExchangeRate: 0, TotalAmount: 11800, TaxAmount: 1800,
        Lines: [{ ProductID: MADE_PRODUCT, Amount: 2950 }, { ProductID: BOUGHT_PRODUCT, Amount: 8850 }],
      }],
    }))
    expect(entryOf(operation)).toEqual(["1212:11800/0", "7022:0/2500", "7012:0/7500", "40111:0/1800"])
    expect(operation.DocumentLabel).toBe("F001-15")
    expect(operation.InOfficialBook).toBe(true)
    expect(operation.Status).toBe(OPERATION_STATUS_CLASSIFIED)
  })

  it('reverses the sale entry on a credit note', () => {
    const operation = onlyOperation(makeResponse({
      SalesDocuments: [makeInvoice({ ID: 7_000_302, AffectedDocID: SALE_ID, IssueDate: DAY + 1 })],
      ReferencedSales: [{ ID: SALE_ID, Date: DAY - 40, ss: 0, CurrencyType: 1, ExchangeRate: 0, TotalAmount: 11800,
        TaxAmount: 1800, Lines: [{ ProductID: BOUGHT_PRODUCT, Amount: 11800 }] }],
    }))
    expect(operation.Kind).toBe(OPERATION_CREDIT_NOTE)
    expect(entryOf(operation)).toEqual(["1212:0/11800", "7012:10000/0", "40111:1800/0"])
  })

  it('declares nothing for a voided comprobante', () => {
    expect(buildAccountingOperations(makeResponse({ SalesDocuments: [makeInvoice({ State: 0 })] }), context))
      .toHaveLength(0)
  })

  it('books a sale without comprobante outside the official Diario', () => {
    const operation = onlyOperation(makeResponse({
      Sales: [{ ID: UNINVOICED_SALE_ID, Date: DAY, ss: 1, CurrencyType: 1, ExchangeRate: 0, TotalAmount: 590,
        TaxAmount: 90, Lines: [{ ProductID: BOUGHT_PRODUCT, Amount: 590 }] }],
    }))
    expect(operation.InOfficialBook).toBe(false)
    expect(entryOf(operation)).toEqual(["1212:590/0", "7012:0/500", "40111:0/90"])
  })

  it('flags a sale whose made products wait for the production close as incomplete', () => {
    const operation = onlyOperation(makeResponse({
      SalesDocuments: [makeInvoice()],
      Sales: [{
        ID: SALE_ID, Date: DAY, ss: 4, CurrencyType: 1, ExchangeRate: 0, TotalAmount: 11800, TaxAmount: 1800,
        Lines: [{ ProductID: MADE_PRODUCT, Amount: 11800 }], PendingStockMovement: 1,
      }],
    }))
    expect(operation.Status).toBe(OPERATION_STATUS_INCOMPLETE)
    expect(operation.Problem).toContain('por regularizar')
  })
})

describe('purchases', () => {
  it('posts a factura expense to its account with IGV credit and an issued payable', () => {
    const operation = onlyOperation(makeResponse({ Expenses: [makePurchase()] }))
    expect(entryOf(operation)).toEqual(["6352:10000/0", "40111:1800/0", "4212:0/11800"])
    expect(operation.Lines[0].IsNature).toBe(true)
    expect(operation.Status).toBe(OPERATION_STATUS_CLASSIFIED)
  })

  it('leaves an expense with no account unclassified, without IGV when it has no comprobante', () => {
    const operation = onlyOperation(makeResponse({
      Expenses: [makePurchase({ AccountCode: 0, DocType: 0, TaxableAmount: 0, TaxAmount: 0, RecordAmount: 5000 })],
    }))
    expect(entryOf(operation)).toEqual(["0:5000/0", "4699:0/5000"])
    expect(operation.Status).toBe(OPERATION_STATUS_UNCLASSIFIED)
    expect(operation.InOfficialBook).toBe(false)
  })

  it('splits an order base across its lines, a product to 6011 and a supply to its account', () => {
    const operation = onlyOperation(makeResponse({
      PurchaseOrders: [makePurchase({
        Source: PURCHASE_SOURCE_ORDER, SourceID: 12, Name: "", AccountCode: 0, ExpenseType: 0,
        Lines: [
          { ProductID: BOUGHT_PRODUCT, Quantity: 1, Price: 5900 },
          { ProductID: 77, Quantity: 2, Price: 2950, AccountCode: 6032 },
        ],
      })],
    }))
    expect(entryOf(operation)).toEqual(["6011:5000/0", "6032:5000/0", "40111:1800/0", "4212:0/11800"])
    expect(operation.Lines.map(line => line.IsNature)).toEqual([false, true, false, false])
  })

  it('flags a dollar factura with no exchange rate as incomplete', () => {
    const operation = onlyOperation(makeResponse({ Expenses: [makePurchase({ CurrencyType: 2, ExchangeRate: 0 })] }))
    expect(operation.Status).toBe(OPERATION_STATUS_INCOMPLETE)
  })

  it('values an older order reception at its cost on the pair of its stamped purchase account', () => {
    const operation = onlyOperation(makeResponse({
      ReferencedPurchases: [makePurchase({ Source: PURCHASE_SOURCE_ORDER, SourceID: 12, DocIssueDate: DAY - 40 })],
      StockMovements: [{ ID: 1, Date: DAY, Type: 3, DocumentID: 12, ProductID: BOUGHT_PRODUCT, MonetaryValue: 5000,
        PurchaseAccountCode: 6011 }],
    }))
    expect(entryOf(operation)).toEqual(["2011:5000/0", "6111:0/5000"])
    expect(operation.DocumentLabel).toBe("F002-88")
  })

  it('leaves out a reception whose purchase account enters no stock', () => {
    expect(buildAccountingOperations(makeResponse({
      StockMovements: [{ ID: 1, Date: DAY, Type: 4, DocumentID: 5, ProductID: SUPPLY, MonetaryValue: 5000,
        PurchaseAccountCode: 6352 }],
    }), context)).toHaveLength(0)
  })
})

describe('cost of sales', () => {
  it('posts each delivery at its stamped cost on the accounts its product origin names', () => {
    const operation = onlyOperation(makeResponse({
      StockMovements: [
        makeSaleMovement(8, BOUGHT_PRODUCT, 3000),
        makeSaleMovement(8, MADE_PRODUCT, 1200),
        makeSaleMovement(8, SUPPLY, 400),
      ],
    }))
    expect(entryOf(operation)).toEqual([
      "6912:3400/0", "2011:0/3000", "6922:1200/0", "2111:0/1200", "252:0/400",
    ])
    expect(operation.InOfficialBook).toBe(true)
    expect(operation.Status).toBe(OPERATION_STATUS_CLASSIFIED)
  })

  it('reverses the cost on an annulment return, and follows an undocumented sale out of the Diario', () => {
    const operation = onlyOperation(makeResponse({
      StockMovements: [makeSaleMovement(9, BOUGHT_PRODUCT, 3000, UNINVOICED_SALE_ID)],
    }))
    expect(entryOf(operation)).toEqual(["6912:0/3000", "2011:3000/0"])
    expect(operation.InOfficialBook).toBe(false)
  })

  it('flags a delivery that left with no cost', () => {
    const operation = onlyOperation(makeResponse({ StockMovements: [makeSaleMovement(8, BOUGHT_PRODUCT, 0)] }))
    expect(operation.Status).toBe(OPERATION_STATUS_INCOMPLETE)
  })
})

describe('production close', () => {
  const CLOSE_ID = 129
  const AUXILIARY_SUPPLY = 44 // account 6031 → stock 251, variation 6131
  const STANDARD_COST_PRODUCT = 45
  const makeCloseMovement = (Type: number, ProductID: number, MonetaryValue: number) =>
    ({ ID: Type * 1000 + ProductID, Date: DAY, Type, DocumentID: CLOSE_ID, ProductID, MonetaryValue })
  const products = [
    ...makeResponse().Products,
    { ID: AUXILIARY_SUPPLY, ss: 2, Origin: 0, AccountCode: 6031 },
    { ID: STANDARD_COST_PRODUCT, ss: 1, Origin: 2, AccountCode: 0 },
  ]

  it('consumes each supply into its variation and enters the products as stored production', () => {
    const operation = onlyOperation(makeResponse({
      Products: products,
      StockMovements: [
        makeCloseMovement(15, SUPPLY, 750),
        makeCloseMovement(15, AUXILIARY_SUPPLY, 250),
        // The recipe product at its supplies (1000) plus extra cost; the other at its standard cost.
        makeCloseMovement(16, MADE_PRODUCT, 1100),
        makeCloseMovement(16, STANDARD_COST_PRODUCT, 600),
      ],
    }))
    expect(operation.Key).toBe(`PR-${CLOSE_ID}-${DAY}`)
    expect(entryOf(operation)).toEqual([
      "6132:750/0", "252:0/750", "6131:250/0", "251:0/250", "2111:1700/0", "7111:0/1700",
    ])
    expect(operation.InOfficialBook).toBe(true)
    expect(operation.Status).toBe(OPERATION_STATUS_CLASSIFIED)
  })

  it('flags a consumed supply whose account has no stock account', () => {
    const operation = onlyOperation(makeResponse({
      Products: [...products, { ID: 46, ss: 2, Origin: 0, AccountCode: 656 }],
      StockMovements: [makeCloseMovement(15, 46, 300), makeCloseMovement(16, MADE_PRODUCT, 300)],
    }))
    expect(operation.Status).toBe(OPERATION_STATUS_INCOMPLETE)
  })

  it('shows a movement type no entry handles instead of skipping it', () => {
    const operation = onlyOperation(makeResponse({ StockMovements: [makeSaleMovement(13, BOUGHT_PRODUCT, 500)] }))
    expect(operation.Status).toBe(OPERATION_STATUS_INCOMPLETE)
    expect(operation.Problem).toContain("13")
  })
})

describe('assets', () => {
  it('books a donated asset against capital', () => {
    const operation = onlyOperation(makeResponse({ Assets: [makeAsset()] }))
    expect(entryOf(operation)).toEqual(["3361:300000/0", "501:0/300000"])
    expect(operation.InOfficialBook).toBe(true)
  })

  it('depreciates and disposes through the 33x pairs', () => {
    const asset = makeAsset({ AcquisitionDate: DAY - 400, DisposalDate: DAY, AccumulatedDepreciation: 100000 })
    const operations = buildAccountingOperations(makeResponse({
      Assets: [asset], Expenses: [makeDepreciation(asset.ID, DAY - 1, 8333)],
    }), context)
    expect(operations.map(entryOf)).toEqual([
      ["68415:8333/0", "39527:0/8333"],
      ["39527:100000/0", "6551:200000/0", "3361:0/300000"],
    ])
  })
})

describe('cash', () => {
  it('posts a sale collection against 1212 and keeps it out of the Diario with its sale', () => {
    const operation = onlyOperation(makeResponse({
      CashMovements: [{ ID: 1, CashBankID: 1, DocumentID: UNINVOICED_SALE_ID, Date: DAY, Type: 8,
        CounterAccountCode: 1212, Amount: 590, CashBankRefID: 0 }],
    }))
    expect(entryOf(operation)).toEqual(["101:590/0", "1212:0/590"])
    expect(operation.InOfficialBook).toBe(false)
  })

  it('posts a supplier payment against its stamped payable, official only on 4212', () => {
    const [documented, undocumented] = buildAccountingOperations(makeResponse({
      CashMovements: [
        { ID: 1, CashBankID: 1, DocumentID: 12, Date: DAY, Type: 6, CounterAccountCode: 4212, Amount: -1000, CashBankRefID: 0 },
        { ID: 2, CashBankID: 1, DocumentID: 13, Date: DAY, Type: 6, CounterAccountCode: 4699, Amount: -500, CashBankRefID: 0 },
      ],
    }), context)
    expect(entryOf(documented)).toEqual(["101:0/1000", "4212:1000/0"])
    expect(documented.InOfficialBook).toBe(true)
    expect(undocumented.InOfficialBook).toBe(false)
  })

  it('infers the payment that settled a document on its own date', () => {
    const sources = makeResponse({
      Sales: [{ ID: SALE_ID, Date: DAY, ss: 2, CurrencyType: 1, ExchangeRate: 0, TotalAmount: 11800, TaxAmount: 1800,
        SameDayCashBankID: 1 }],
      PurchaseOrders: [makePurchase({ Source: PURCHASE_SOURCE_ORDER, SourceID: 12, SameDayCashBankID: 1 })],
      Expenses: [makePurchase({ SourceID: 6, DocType: 0, SameDayCashBankID: 1 })],
      Assets: [makeAsset({ PurchaseAmount: 300000, SameDayCashBankID: 1 })],
    })
    expect(inferSameDayPayments(sources).map(payment => [payment.Type, payment.DocumentID, payment.Amount, payment.CounterAccountCode]))
      .toEqual([[8, SALE_ID, 11800, 1212], [6, 12, -11800, 4212], [9, 6, -11800, 4699], [10, 3, -300000, 4654]])
    const payments = buildAccountingOperations(sources, context).filter(operation => operation.Key.startsWith("CJD-"))
    expect(payments.map(payment => [payment.Key, payment.InOfficialBook])).toEqual([
      ["CJD-10-3", false], ["CJD-6-12", true], ["CJD-8-7000101", true], ["CJD-9-6", false],
    ])
    expect(entryOf(payments[2])).toEqual(["101:11800/0", "1212:0/11800"])
  })

  it('books a transfer once, from its outgoing row', () => {
    const operation = onlyOperation(makeResponse({
      CashBanks: [{ ID: 1, Name: "Caja", Type: 1, CurrencyType: 1 }, { ID: 2, Name: "BCP", Type: 2, CurrencyType: 1 }],
      CashMovements: [
        { ID: 1, CashBankID: 1, DocumentID: 0, Date: DAY, Type: 3, CounterAccountCode: 0, Amount: -30000, CashBankRefID: 2 },
        { ID: 2, CashBankID: 2, DocumentID: 0, Date: DAY, Type: 3, CounterAccountCode: 0, Amount: 30000, CashBankRefID: 1 },
      ],
    }))
    expect(entryOf(operation)).toEqual(["101:0/30000", "1041:30000/0"])
  })

  it('leaves an unspecified movement unclassified until it gets an account', () => {
    const operation = onlyOperation(makeResponse({
      CashMovements: [{ ID: 1, CashBankID: 1, DocumentID: 0, Date: DAY, Type: 1, CounterAccountCode: 0,
        Amount: -2000, CashBankRefID: 0 }],
    }))
    expect(operation.Status).toBe(OPERATION_STATUS_UNCLASSIFIED)
  })
})

describe('classification targets', () => {
  it('offers an order only its supply lines, with their index', () => {
    const response = makeResponse({
      PurchaseOrders: [makePurchase({
        Source: PURCHASE_SOURCE_ORDER, SourceID: 12, AccountCode: 0,
        Lines: [{ ProductID: BOUGHT_PRODUCT, Quantity: 1, Price: 100 }, { ProductID: SUPPLY, Quantity: 1, Price: 100, AccountCode: 6032 }],
      })],
    })
    const [operation] = buildAccountingOperations(response, context)
    expect(classificationTargetsOf(operation, response).map(target => [target.Kind, target.ID, target.Line, target.AccountCode]))
      .toEqual([[3, 12, 1, 6032]])
  })

  it('sends depreciation to its asset, and leaves derived operations without targets', () => {
    const asset = makeAsset({ AcquisitionDate: DAY - 400 })
    const response = makeResponse({
      Assets: [asset], Expenses: [makeDepreciation(asset.ID, DAY, 100)],
      SalesDocuments: [makeInvoice()],
    })
    const operations = buildAccountingOperations(response, context)
    const depreciation = operations.find(operation => operation.Key === "DEP-90")!
    const sale = operations.find(operation => operation.Key.startsWith("V-"))!
    expect(classificationTargetsOf(depreciation, response).map(target => [target.Kind, target.ID])).toEqual([[6, asset.ID]])
    expect(classificationTargetsOf(sale, response)).toEqual([])
  })

  it('bulk-applies only when every target shares one PCGE kind', () => {
    const response = makeResponse({
      Expenses: [makePurchase(), makePurchase({ SourceID: 6, ExpenseType: 2, AccountCode: 6032 })],
    })
    const operations = buildAccountingOperations(response, context)
    expect(bulkClassificationTargets(operations.slice(0, 1), response).pcgeKind).toBe(2)
    expect(bulkClassificationTargets(operations, response).pcgeKind).toBe(0)
  })
})

describe('Diario de gestión and trial balance', () => {
  const response = makeResponse({
    Expenses: [makePurchase(), makePurchase({ SourceID: 6, AccountCode: 0, DocType: 0, RecordAmount: 500 })],
  })
  const operations = buildAccountingOperations(response, context)

  it('splits an entry into its Debe and Haber accounts', () => {
    expect(entrySides(operations[0])).toEqual({
      DebitLines: [{ AccountCode: 6352, Amount: 10000 }, { AccountCode: 40111, Amount: 1800 }],
      CreditLines: [{ AccountCode: 4212, Amount: 11800 }],
    })
  })

  it('rolls the sums up the chart, in tree order, unclassified last', () => {
    const rows = buildTrialBalance(operations)
    expect(rows.map(row => [row.AccountCode, row.Debit, row.Credit])).toEqual([
      [40, 1800, 0], [401, 1800, 0], [4011, 1800, 0], [40111, 1800, 0],
      [42, 0, 11800], [421, 0, 11800], [4212, 0, 11800],
      [46, 0, 500], [469, 0, 500], [4699, 0, 500],
      [63, 10000, 0], [635, 10000, 0], [6352, 10000, 0],
      [0, 500, 0],
    ])
    expect(rows.find(row => row.AccountCode === 6352)?.Depth).toBe(2)
  })
})

describe('groupOperations', () => {
  const makeUninvoicedSale = (ID: number, TotalAmount: number, TaxAmount: number, ProductID = BOUGHT_PRODUCT) => ({
    ID, Date: DAY, ss: 1, CurrencyType: 1, ExchangeRate: 0, TotalAmount, TaxAmount,
    Lines: [{ ProductID, Amount: TotalAmount }],
  })

  it('folds the sales that hit the same accounts into one row, summed per account', () => {
    const groups = groupOperations(buildAccountingOperations(makeResponse({
      Sales: [makeUninvoicedSale(UNINVOICED_SALE_ID, 590, 90), makeUninvoicedSale(UNINVOICED_SALE_ID + 100, 1180, 180)],
    }), context))
    expect(groups).toHaveLength(1)
    expect(groups[0].Operations).toHaveLength(2)
    expect(groups[0].DebitLines).toEqual([{ AccountCode: 1212, Amount: 1770 }])
    expect(groups[0].CreditLines).toEqual([{ AccountCode: 7012, Amount: 1500 }, { AccountCode: 40111, Amount: 270 }])
  })

  it('keeps apart a sale that hits another account, and an operation of another status', () => {
    const groups = groupOperations(buildAccountingOperations(makeResponse({
      Sales: [makeUninvoicedSale(UNINVOICED_SALE_ID, 590, 90), makeUninvoicedSale(UNINVOICED_SALE_ID + 100, 590, 90, MADE_PRODUCT)],
      Expenses: [makePurchase(), makePurchase({ SourceID: 6, AccountCode: 0 })],
    }), context))
    expect(groups.map(group => group.Operations.length)).toEqual([1, 1, 1, 1])
  })
})

describe('summarizeOperations', () => {
  it('counts by status and keeps Debe equal to Haber', () => {
    const operations = buildAccountingOperations(makeResponse({
      Expenses: [makePurchase(), makePurchase({ SourceID: 6, AccountCode: 0 })],
    }), context)
    const summary = summarizeOperations(operations)
    expect(summary).toMatchObject({ count: 2, classified: 1, unclassified: 1 })
    expect(summary.debitTotal).toBe(summary.creditTotal)
  })
})

describe('what the period reads outside itself', () => {
  it('asks only for the older sales, orders and expenses the period names', () => {
    expect(referencedRecordIDs(makeResponse({
      Sales: [{ ID: SALE_ID, Date: DAY, ss: 2, CurrencyType: 1, ExchangeRate: 0, TotalAmount: 0, TaxAmount: 0 }],
      SalesDocuments: [makeInvoice({ ID: 7_000_302, AffectedDocID: 7_000_501 }), makeInvoice({ ID: 7_000_402, AffectedDocID: SALE_ID })],
      Expenses: [makePurchase({ SourceID: 5 })],
      StockMovements: [
        { ID: 1, Date: DAY, Type: 3, DocumentID: 12, ProductID: BOUGHT_PRODUCT, MonetaryValue: 1 },
        { ID: 2, Date: DAY, Type: 4, DocumentID: 5, ProductID: SUPPLY, MonetaryValue: 1 },
        { ID: 3, Date: DAY, Type: 4, DocumentID: 8, ProductID: SUPPLY, MonetaryValue: 1 },
      ],
    }))).toEqual({ saleIDs: [7_000_501], orderIDs: [12], expenseIDs: [8] })
  })

  it('bounds a period by its first and last UnixDay', () => {
    expect(periodBounds("202609")).toEqual({ firstDay: FIRST_DAY, lastDay: FIRST_DAY + 29 })
    expect(periodBounds("202602")).toEqual({ firstDay: 20485, lastDay: 20512 })
  })
})

describe('payroll', () => {
  // S/ 2,000 salary + S/ 150 movilidad (625); ONP 13 % and a S/ 300 advance; EsSalud 9 %; CTS provision.
  const payslip = {
    ID: 9, Period: 2609, NetPay: 159000, ss: 2,
    Lines: [
      { Kind: 1, AccountCode: 6211, PayableAccountCode: 0, Amount: 200000 },
      { Kind: 1, AccountCode: 625, PayableAccountCode: 0, Amount: 15000 },
      { Kind: 2, AccountCode: 0, PayableAccountCode: 4032, Amount: 26000 },
      { Kind: 2, AccountCode: 0, PayableAccountCode: 1412, Amount: 30000 },
      { Kind: 3, AccountCode: 6271, PayableAccountCode: 4031, Amount: 18000 },
      { Kind: 4, AccountCode: 629, PayableAccountCode: 415, Amount: 19444 },
    ],
  }

  it('posts the payslip and its provisions as two balanced entries on the last day', () => {
    const operations = buildAccountingOperations(makeResponse({ Payslips: [payslip] }), context)
    expect(operations.map(operation => operation.Key)).toEqual(['PL-9', 'PP-9'])
    for (const operation of operations) {
      expect(operation.Date).toBe(FIRST_DAY + 29)
      expect(operation.Status).toBe(OPERATION_STATUS_CLASSIFIED)
      const debit = operation.Lines.reduce((total, line) => total + line.Debit, 0)
      const credit = operation.Lines.reduce((total, line) => total + line.Credit, 0)
      expect(debit).toBe(credit)
    }
    const payroll = operations[0].Lines
    expect(payroll.find(line => line.AccountCode === 4111)?.Credit).toBe(159000)
    expect(payroll.find(line => line.AccountCode === 1412)?.Credit).toBe(30000)
    expect(payroll.find(line => line.AccountCode === 6271)?.Debit).toBe(18000)
  })

  it('charges a gratificación to its provision first and only the excess to 6214', () => {
    const gratificacion = {
      ID: 10, Period: 2607, NetPay: 223450, ss: 2,
      Lines: [
        { Kind: 1, AccountCode: 6214, PayableAccountCode: 0, Amount: 205000, ProvisionAccountCode: 4114, ProvisionedAmount: 205000 },
        { Kind: 1, AccountCode: 6214, PayableAccountCode: 0, Amount: 18450, ProvisionAccountCode: 4114, ProvisionedAmount: 11000 },
      ],
    }
    const [operation] = buildAccountingOperations(makeResponse({ Payslips: [gratificacion] }), context)
    const debitOf = (accountCode: number) => operation.Lines
      .filter(line => line.AccountCode === accountCode).reduce((total, line) => total + line.Debit, 0)
    expect([debitOf(4114), debitOf(6214)]).toEqual([216000, 7450])
    expect(operation.Lines.find(line => line.AccountCode === 4111)?.Credit).toBe(223450)
  })

  it('keeps a CTS deposit owed on 415 and out of the net pay', () => {
    // Salary S/ 2,000 and a CTS of S/ 1,000 charged half to its provision: the net pay is the salary only.
    const month = {
      ID: 11, Period: 2605, NetPay: 200000, ss: 2,
      Lines: [
        { Kind: 1, AccountCode: 6211, PayableAccountCode: 0, Amount: 200000 },
        { Kind: 1, AccountCode: 629, PayableAccountCode: 415, Amount: 100000, ProvisionAccountCode: 415,
          ProvisionedAmount: 50000, OutsideNetPay: true },
      ],
    }
    const [operation] = buildAccountingOperations(makeResponse({ Payslips: [month] }), context)
    const debit = operation.Lines.reduce((total, line) => total + line.Debit, 0)
    const credit = operation.Lines.reduce((total, line) => total + line.Credit, 0)
    expect(debit).toBe(credit)
    expect(operation.Lines.find(line => line.AccountCode === 4111)?.Credit).toBe(200000)
    const onAccount415 = operation.Lines.filter(line => line.AccountCode === 415)
    expect([onAccount415.reduce((total, line) => total + line.Debit, 0), onAccount415.reduce((total, line) => total + line.Credit, 0)])
      .toEqual([50000, 100000])
  })
})
