package tytd_go

import (
	"strings"
	"testing"
)

func TestGodIslandResourceStatus(t *testing.T) {
	p := &EventPlayer{Running: true}
	p.GiEvents[GIEventPeak] = &giEvent{
		ActivityID: 9, Name: "Thiên Ma", Location: 6002,
		PowerKnown: true, PowerCur: 80, PowerMax: 100,
		BallKnown: true, BallCur: 45, BallMax: 20,
	}
	got := p.GodIslandResourceStatus(GIEventPeak)
	for _, want := range []string{"Thiên Ma", "thành 6002", "80/100", "45", "20"} {
		if !strings.Contains(got, want) {
			t.Fatalf("status %q does not contain %q", got, want)
		}
	}
}

func TestGodIslandResourceStatusUnknownAndStopped(t *testing.T) {
	p := &EventPlayer{}
	if got := p.GodIslandResourceStatus(GIEventPeak); got != "⏹️ chưa login" {
		t.Fatalf("stopped status = %q", got)
	}
	p.Running = true
	if got := p.GodIslandResourceStatus(GIEventPeak); !strings.Contains(got, "chưa nhận được dữ liệu") {
		t.Fatalf("unknown status = %q", got)
	}
}
