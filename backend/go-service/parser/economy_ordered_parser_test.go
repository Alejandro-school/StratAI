package parser

import (
	"reflect"
	"testing"

	dem "github.com/markus-wa/demoinfocs-golang/v5/pkg/demoinfocs"
	"github.com/markus-wa/demoinfocs-golang/v5/pkg/demoinfocs/events"
	"github.com/markus-wa/demoinfocs-golang/v5/pkg/demoinfocs/msg"
	dp "github.com/markus-wa/godispatch"
)

type economyUnorderedParser struct {
	dem.Parser
	events, messages []reflect.Value
}

func (p *economyUnorderedParser) RegisterEventHandler(h any) dp.HandlerIdentifier {
	p.events = append(p.events, reflect.ValueOf(h))
	return new(int)
}

func (p *economyUnorderedParser) RegisterNetMessageHandler(h any) dp.HandlerIdentifier {
	p.messages = append(p.messages, reflect.ValueOf(h))
	return new(int)
}

func TestEconomyDependentObserversSurviveReversedSDKDispatch(t *testing.T) {
	sdk := &economyUnorderedParser{}
	p := newEconomyOrderedParser(sdk)
	instance, boundaries, trace, links := 0, 0, 0, 0
	p.RegisterEventHandler(func(events.RoundStart) { instance++ })
	p.RegisterEventHandler(func(events.RoundStart) { boundaries += instance })
	p.RegisterNetMessageHandler(func(*msg.CMsgSource1LegacyGameEvent) { trace++ })
	p.RegisterNetMessageHandler(func(*msg.CMsgSource1LegacyGameEvent) { links += trace })
	if len(sdk.events) != 1 || len(sdk.messages) != 1 {
		t.Fatal("dependent consumers exposed separately to unordered SDK")
	}
	for i := len(sdk.events) - 1; i >= 0; i-- {
		sdk.events[i].Call([]reflect.Value{reflect.ValueOf(events.RoundStart{})})
	}
	for i := len(sdk.messages) - 1; i >= 0; i-- {
		sdk.messages[i].Call([]reflect.Value{reflect.ValueOf(&msg.CMsgSource1LegacyGameEvent{})})
	}
	if boundaries != 1 || links != 1 {
		t.Fatal("dependent observation lost")
	}
}

func TestCashflowAndTraceCountersShareFrameDoneBeforeIdentityAdvances(t *testing.T) {
	sdk := &economyUnorderedParser{}
	p := newEconomyOrderedParser(sdk)
	traceFrame, identityFrame, cuts := 0, 0, 0
	p.RegisterEventHandler(func(events.FrameDone) { traceFrame++ })
	p.RegisterEventHandler(func(events.FrameDone) {
		if traceFrame != identityFrame+1 {
			t.Fatal("trace cut is not completed frame plus one")
		}
		cuts++
		identityFrame++
	})
	for i := 0; i < 3; i++ {
		sdk.events[0].Call([]reflect.Value{reflect.ValueOf(events.FrameDone{})})
	}
	if cuts != 3 || traceFrame != 3 || identityFrame != 3 {
		t.Fatal("frame delivery lost")
	}
}

func TestEconomyOrderedObserverRegistrationAndRemovalDuringDelivery(t *testing.T) {
	sdk := &economyUnorderedParser{}
	p := newEconomyOrderedParser(sdk)
	var got []int
	var second dp.HandlerIdentifier
	p.RegisterEventHandler(func(events.FrameDone) {
		got = append(got, 1)
		p.UnregisterEventHandler(second)
	})
	second = p.RegisterEventHandler(func(events.FrameDone) { got = append(got, 2) })
	for i := 0; i < 2; i++ {
		sdk.events[0].Call([]reflect.Value{reflect.ValueOf(events.FrameDone{})})
	}
	if !reflect.DeepEqual(got, []int{1, 2, 1}) {
		t.Fatal(got)
	}
}
