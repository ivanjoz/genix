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
	// ActionAnnulSaleOrder serializes annulling one sale. The annulment reads both ledgers to
	// decide what to give back, so two concurrent clicks would each read a zero reversal and
	// each write one — refunding twice. Keyed on the sale.
	ActionAnnulSaleOrder LockAction = 4
)
