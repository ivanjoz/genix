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
)
