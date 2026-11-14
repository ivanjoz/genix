package dining

import (
	"context"
	"fmt"
	"time"

	"app/core"
)

// diningPublishTimeout bounds the nudge so a hung bridge cannot hold a waiter's write for long.
// A bridge that is down refuses the connection at once; only a wedged one waits this out.
const diningPublishTimeout = 2 * time.Second

// diningLayoutTopic names the SSE bridge topic of one floor (DINING_PLAN §8). The company is the
// second segment because that is where the bridge reads it to refuse another tenant's tabs; the
// floor page subscribes with the same string.
func diningLayoutTopic(companyID, layoutID int32) string {
	return fmt.Sprintf("dining.%d.%d", companyID, layoutID)
}

// publishDiningLayoutChange tells every tab watching the floor that a table order changed, after
// the write committed. The message is a nudge, not data: the tab pulls dining-table-orders through
// the delta cache, so a nudge lost here is healed by the next one or by the tab's fallback poll.
// That is why a failure is only logged and never fails the write.
func publishDiningLayoutChange(companyID, layoutID, updated int32) {
	if !core.BridgeConfigured() {
		return
	}
	publishContext, cancelPublish := context.WithTimeout(context.Background(), diningPublishTimeout)
	defer cancelPublish()

	topic := diningLayoutTopic(companyID, layoutID)
	nudge := map[string]any{"Type": "dining", "LayoutID": layoutID, "Updated": updated}
	if _, err := core.BridgePublishTopic(publishContext, topic, nudge); err != nil {
		core.Log("dining live update not published topic::", topic, " err::", err)
	}
}
