package tytd_go

import (
	"fmt"
	"time"
)

// GodIslandResourceStatus returns a read-only snapshot of one island's
// combat/resource state. BallMax is the per-draw ceiling, not the reservoir
// capacity; keep those labels distinct so the status cannot mislead users.
func (p *EventPlayer) GodIslandResourceStatus(which int) string {
	if !p.Running {
		return "⏹️ chưa login"
	}

	p.GodIslandMu.Lock()
	ev := p.GiEvents[which]
	if ev == nil || ev.ActivityID == 0 {
		p.GodIslandMu.Unlock()
		return "⏳ chưa nhận được dữ liệu đảo"
	}

	name := ev.Name
	location, routeTo, moving := ev.Location, ev.RouteTo, ev.Moving
	leftOut := ev.LeftOut
	left, dead := ev.reviveLeft()
	hpKnown, hpCur, hpMax := ev.PowerKnown, ev.PowerCur, ev.PowerMax
	ballKnown, ballCur, ballDrawMax := ev.BallKnown, ev.BallCur, ev.BallMax
	p.GodIslandMu.Unlock()

	if name == "" {
		name = giEventLabel(which)
	}
	state := "đang hoạt động"
	switch {
	case leftOut:
		state = "đã bị loại khỏi đảo"
	case dead:
		state = fmt.Sprintf("đang hồi sinh sau %dm%02ds", int(left/time.Minute), int(left/time.Second)%60)
	case moving && routeTo != 0:
		state = fmt.Sprintf("đang hành quân tới thành %d", routeTo)
	case moving:
		state = "đang hành quân"
	case location != 0:
		state = fmt.Sprintf("đang ở thành %d", location)
	}

	hp := "chưa có dữ liệu"
	if hpKnown {
		hp = fmt.Sprintf("%d/%d", hpCur, hpMax)
	}
	ball := "chưa có dữ liệu"
	if ballKnown {
		ball = fmt.Sprintf("%d (giới hạn mỗi lần dùng: %d)", ballCur, ballDrawMax)
	}
	return fmt.Sprintf("🏝️ %s — %s\n❤️ Máu quân: %s\n🔮 Linh lực cầu: %s",
		name, state, hp, ball)
}
