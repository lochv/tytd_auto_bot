package tytd_go

import (
	"fmt"
	"log"
	"time"

	"tytd_go/internal/proto"

	pb "google.golang.org/protobuf/proto"
)

const (
	giPeakBallInfoTicks = 5
	giPeakFruitFeedGap  = 30 * time.Second
	giPeakCondSweepGap  = 10 * time.Minute
)

// giPeakAutoFeedBall is deliberately conservative: Thiên Ma only, one fruit
// per request, and only when the server-synchronized ball reservoir is known
// to be empty. BallMax is a per-draw ceiling, not the reservoir capacity.
func (p *EventPlayer) giPeakAutoFeedBall(ev *giEvent) {
	now := time.Now()
	p.GodIslandMu.Lock()
	if ev == nil || ev.ActivityID == 0 || !ev.BallKnown || ev.BallCur > 0 ||
		(!ev.BallFeedLast.IsZero() && now.Sub(ev.BallFeedLast) < giPeakFruitFeedGap) {
		p.GodIslandMu.Unlock()
		return
	}
	activityID := ev.ActivityID
	ev.BallFeedLast = now
	p.GodIslandMu.Unlock()

	p.giSend(ev, GOD_ISLAND_USE_FRUIT_CMD, SERVER_GOD_ISLAND_USE_FRUIT,
		&proto.GodIslandUseFruitReq{ActivityId: activityID, UseCount: 1})
	log.Printf("[%s] [GodIsland/%s] gửi yêu cầu tự bồi dưỡng Linh Lực: 1 quả (cầu đã cạn)",
		p.Username, ev.Name)
}

// giPeakConditionSweep uses only the condition rows returned by this
// activity's own 21003 detail. Only guild-recharge rows (type 84 or
// conditionId 2xxxx) are eligible; the server decides which rewards are ready.
// Each row is claimed through 21045, matching the SeaWar/RebornTrial engines.
func (p *EventPlayer) giPeakConditionSweep(ev *giEvent) {
	now := time.Now()
	p.GodIslandMu.Lock()
	if ev == nil || ev.ActivityID == 0 {
		p.GodIslandMu.Unlock()
		return
	}
	cfgs := append([]*proto.ActivityConditionConfig(nil), ev.CondCfgs...)
	if len(cfgs) == 0 || now.Sub(ev.CondSweepLast) < giPeakCondSweepGap {
		p.GodIslandMu.Unlock()
		return // the existing garden/shop 21003 pull will populate CondCfgs
	}
	activityID := ev.ActivityID
	ev.CondSweepLast = now
	p.GodIslandMu.Unlock()

	seenCross := map[int32]bool{}
	for _, cfg := range cfgs {
		if cfg == nil || (cfg.GetType() != 84 && cfg.GetConditionId() < 20000) {
			continue
		}
		conditionID := cfg.GetConditionId()
		if conditionID <= 0 || seenCross[conditionID] {
			continue
		}
		seenCross[conditionID] = true
		actID := cfg.GetActivityId()
		if actID == 0 {
			actID = activityID
		}
		body, err := pb.Marshal(&proto.ReceiveCrossUnionRechargeRewardReq{
			ActivityId: pb.Int32(actID),
			ConditionId: pb.Int32(conditionID),
		})
		if err != nil {
			continue
		}
		data, err := p.rbtRT(ACTIVITY_CROSS_RECHARGE_CMD, SERVER_ACTIVITY_CROSS_RECHARGE, body)
		if err != nil {
			log.Printf("[%s] [GodIsland/%s] claim guild milestone %d failed: %v",
				p.Username, ev.Name, conditionID, err)
			continue
		}
		p.giHandleConditionReward(ev, data)
	}
}

// giDeliverActivityDetail attributes 21003 only to a request issued by
// the God Island engine. This transport is shared with other event engines,
// so the single-open-event fallback is not safe for this response type.
func (p *EventPlayer) giDeliverActivityDetail(data []byte) bool {
	var resp proto.RspGetActivityDetail
	if pb.Unmarshal(data, &resp) != nil {
		return false
	}
	activityID := resp.GetActivity().GetActivityId()
	if activityID == 0 {
		return false
	}

	p.GodIslandMu.Lock()
	queue := p.giPending[SERVER_ACTIVITY_DETAIL]
	index := -1
	for i, ev := range queue {
		if ev != nil && ev.ActivityID == activityID {
			index = i
			break
		}
	}
	if index < 0 {
		p.GodIslandMu.Unlock()
		return false // a 21003 response belonging to another event engine
	}
	ev := queue[index]
	queue = append(queue[:index], queue[index+1:]...)
	if len(queue) == 0 {
		delete(p.giPending, SERVER_ACTIVITY_DETAIL)
	} else {
		p.giPending[SERVER_ACTIVITY_DETAIL] = queue
	}
	p.GodIslandMu.Unlock()
	p.giHandleActivityDetail(ev, data)
	return true
}

// giHandleConditionReward logs only successful non-empty claims. Empty
// rewards and ret!=0 are routine when a milestone is not ready yet.
func (p *EventPlayer) giHandleConditionReward(ev *giEvent, data []byte) {
	var resp proto.RspGetActivityConditionReward
	if err := pb.Unmarshal(data, &resp); err != nil {
		log.Printf("[%s] [GodIsland/%s] bad condition reward response: %v", p.Username, ev.Name, err)
		return
	}
	if resp.GetRet() != 0 || resp.GetRewards() == "" {
		return
	}
	log.Printf("[%s] [GodIsland/%s] nhận thưởng mốc nhiệm vụ bang/sự kiện: %s",
		p.Username, ev.Name, resp.GetRewards())
	p.forceNotify(fmt.Sprintf("🏝️ [%s] nhận thưởng mốc nạp bang: %s", ev.Name, resp.GetRewards()))
}
