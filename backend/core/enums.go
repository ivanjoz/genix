package core

// LockAction names a family of distributed locks. It is the first half of a lock's key, the
// identifier being the second, which is what lets two features serialize on the same number
// without ever colliding: sign-up's IP 3405803821 and some future action's company 3405803821 are
// different keys.
//
// The values travel to the Rust daemon as a raw uint16, so they are protocol: never renumber one
// that is deployed. A rolling deploy would leave two processes disagreeing about what a number
// means, which is exactly the collision the namespace exists to prevent. Retire a value instead,
// and take the next free one.
type LockAction uint16

const (
	// ActionSignUpByIP serializes public registration per client IP, so the "N emails per window"
	// count cannot be read by two Lambdas at once.
	ActionSignUpByIP LockAction = 1
	// ActionContactByIP does the same for the public contact form. It is its own action rather than
	// a reuse of the one above so the two forms never queue behind each other: a visitor sending a
	// message while a registration is in flight from the same IP is not the abuse either limit is
	// looking for.
	ActionContactByIP LockAction = 2
	// ActionInvoiceSaleOrder serializes issuing an electronic document for one
	// sale, so two clicks cannot produce two documents for it. Keyed on the sale,
	// so unrelated sales never queue behind each other.
	ActionInvoiceSaleOrder LockAction = 3
	// 4 is retired (ActionAnnulSaleOrder): the annulment locks the sale's row instead.
	// 5 is retired (ActionSaveInvoiceSeries): the series save locks the company's row instead.
	// ActionCompactDataFrame keeps one compaction at a time on a DataFrame partition
	// (genix-orm/dataframe): the reads of other Lambdas that find it due queue behind it and then read
	// its files. Keyed on a hash of the frame's folder and the company. The frame's state and files are
	// written conditioned on their ETags, so a lost lock costs duplicate work only.
	ActionCompactDataFrame LockAction = 6
	// 7 is retired (ActionProjectWrite): project writes lock the project's row instead.
	// ActionWebCustomerByIP serializes the web store's emailed codes and customer creation per
	// client IP, so the "N emails / customers per window" counts are read by one caller at a time.
	ActionWebCustomerByIP LockAction = 8
	// ActionWebCheckoutByCustomer serializes one web customer's checkouts, so the cap on pending
	// (unpaid) orders cannot be passed by placing several at once.
	ActionWebCheckoutByCustomer LockAction = 9
	// ActionOnlineSaleOrderSettle serializes turning one web store order into its SaleOrder (Culqi
	// approval, back-office payment or delivery, annulment). It is taken before the sale's number is
	// minted, which a row lock inside the transaction could not cover: two settlements of one order
	// would otherwise both spend a comprobante number. Keyed on companyID<<32 | order ID.
	ActionOnlineSaleOrderSettle LockAction = 10
	// ActionBuildEcommerceCatalog keeps one build of a company's web store catalog file at a time
	// (business/ecommerce-catalog.go): a delta request that needs the file queues behind the build
	// under way and then reuses its file; one that only noticed the file aging skips. Keyed on the company.
	ActionBuildEcommerceCatalog LockAction = 11
)
