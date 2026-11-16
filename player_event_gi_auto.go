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
	log.Printf("[%s] [GodIsland/%s] tự bồi dưỡng Linh Lực: dùng 1 quả vì cầu đã cạn",
		p.Username, ev.Name)
}

// giPeakConditionSweep uses only the condition rows returned by this
// activity's own 21003 detail. The server decides which rewards are ready.
// Ordinary rows are grouped by type (21033); cross-union recharge rows are
// claimed individually (21045), matching the existing RebornTrial engine.
func (p *EventPlayer) giPeakConditionSweep(ev *giEvent) {
	now := time.Now()
	p.GodIslandMu.Lock()
	if ev == nil || ev.ActivityID == 0 || now.Sub(ev.CondSweepLast) < giPeakCondSweepGap {
		p.GodIslandMu.Unlock()
		return
	}
	activityID := ev.ActivityID
	cfgs := append([]*proto.ActivityConditionConfig(nil), ev.CondCfgs...)
	ev.CondSweepLast = now
	needConfig := len(cfgs) == 0 || now.Sub(ev.CondConfigLast) >= giPeakCondSweepGap
	if needConfig {
		ev.CondConfigLast = now
	}
	p.GodIslandMu.Unlock()

	if needConfig {
		p.giSend(ev, ACTIVITY_GET_DATA_CMD, SERVER_ACTIVITY_DETAIL,
			&proto.ReqGetActivityDetail{ActivityId: activityID})
		if len(cfgs) == 0 {
			return
		}
	}
	if len(cfgs) == 0 {
		return
	}

	seenTypes := make(map[int32]bool)
	seenCross := make(map[int32]bool)
	for _, cfg := range cfgs {
		if cfg == nil {
			continue
		}
		actID := cfg.GetActivityId()
		if actID == 0 {
			actID = activityID
		}
		conditionID := cfg.GetConditionId()
		if conditionID <= 0 {
			continue
		}
		if cfg.GetType() == 84 || conditionID >= 20000 {
			if seenCross[conditionID] {
				continue
			}
			seenCross[conditionID] = true
			p.giSend(ev, ACTIVITY_CROSS_RECHARGE_CMD, SERVER_ACTIVITY_CROSS_RECHARGE,
				&proto.ReceiveCrossUnionRechargeRewardReq{
					ActivityId: pb.Int32(actID),
					ConditionId: pb.Int32(conditionID),
				})
			continue
		}
		typ := cfg.GetType()
		if typ <= 0 || seenTypes[typ] {
			continue
		}
		seenTypes[typ] = true
		p.giSend(ev, ACTIVITY_TOKEN_REWARD_CMD, SERVER_ACTIVITY_TOKEN_REWARD,
			&proto.ReqGetActivityConditionReward{
				ActivityId: pb.Int32(actID),
				ConditionId: pb.Int32(typ),
			})
	}
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
	p.sendNotify(fmt.Sprintf("🏝️ [%s] nhận thưởng mốc nhiệm vụ: %s", ev.Name, resp.GetRewards()))
}
