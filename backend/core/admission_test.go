package core

import (
	"context"
	"testing"
	"time"
)

// A saturated server queues a request, hands it the first slot freed, and refuses it when its
// client gives up instead of holding a slot for nobody.
func TestAdmitRequestQueuesThenRefuses(t *testing.T) {
	previousSlots := admissionSlots
	admissionSlots = make(chan struct{}, 1)
	t.Cleanup(func() { admissionSlots = previousSlots })

	releaseFirst, isAdmitted := AdmitRequest(context.Background())
	if !isAdmitted {
		t.Fatal("the first request found no free slot")
	}

	queued := make(chan bool, 1)
	go func() {
		releaseQueued, isQueuedAdmitted := AdmitRequest(context.Background())
		if isQueuedAdmitted {
			releaseQueued()
		}
		queued <- isQueuedAdmitted
	}()
	time.Sleep(20 * time.Millisecond)
	releaseFirst()
	if !<-queued {
		t.Fatal("the queued request was refused although a slot was freed")
	}

	releaseHolder, _ := AdmitRequest(context.Background())
	defer releaseHolder()
	clientGone, cancel := context.WithCancel(context.Background())
	cancel()
	if _, isAdmitted := AdmitRequest(clientGone); isAdmitted {
		t.Fatal("a request whose client left was admitted into a full server")
	}
}
