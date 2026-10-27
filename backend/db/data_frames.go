package db

import (
	"time"

	"github.com/ivanjoz/genix-orm/scylla"
)

// SetDataFrameLock installs the lock a DataFrame compaction takes (core installs fareward's at boot,
// with the file store: the lock's action number lives in core). It names the driver, which is why it
// is not in db.go.
func SetDataFrameLock(acquire func(lockName string, wait, lease time.Duration) (release func(), isTaken bool, err error)) {
	scylla.AcquireFrameLock = acquire
}
