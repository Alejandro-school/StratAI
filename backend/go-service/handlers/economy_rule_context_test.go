package handlers

import (
	"cs2-demo-service/models"
	"encoding/json"
	"github.com/markus-wa/demoinfocs-golang/v5/pkg/demoinfocs/msg"
	"strings"
	"testing"
)

func ruleString(value string) *string { return &value }

func ruleMessage(entries ...*msg.CMsg_CVars_CVar) *msg.CNETMsg_SetConVar {
	return &msg.CNETMsg_SetConVar{Convars: &msg.CMsg_CVars{Cvars: entries}}
}

func TestNativeRuleMessagesKeepPresenceOrderAndImmutablePrefix(t *testing.T) {
	tracker := newEconomyRuleTracker()
	value := "800"
	tracker.convars(ruleMessage(
		&msg.CMsg_CVars_CVar{Name: ruleString("mp_startmoney"), Value: &value},
		&msg.CMsg_CVars_CVar{Name: ruleString("mp_startmoney"), Value: ruleString("0")},
		&msg.CMsg_CVars_CVar{Name: ruleString("mp_free_armor"), Value: ruleString("false")},
		&msg.CMsg_CVars_CVar{Name: ruleString("mp_maxmoney")},
		&msg.CMsg_CVars_CVar{Name: ruleString("mp_buytime"), Value: ruleString("")},
		&msg.CMsg_CVars_CVar{Name: ruleString("hostname"), Value: ruleString("private-name")},
	), 0)
	tracker.boundary(1, 100, "round_start_probe", map[string]models.EconomyFact{})
	value = "9999"
	tracker.convars(ruleMessage(&msg.CMsg_CVars_CVar{Name: ruleString("mp_startmoney"), Value: ruleString("16000")}), 100)
	tracker.boundary(1, 200, "freeze_end_probe", map[string]models.EconomyFact{})
	first, last := tracker.ledger.Boundaries[0], tracker.ledger.Boundaries[1]
	if *first.Convars["mp_startmoney"].Value != "0" || *last.Convars["mp_startmoney"].Value != "16000" || *tracker.ledger.Messages[0].Entries[0].Value != "800" {
		t.Fatal("mutable or future value entered prefix")
	}
	if first.Convars["mp_maxmoney"].Value != nil || first.Convars["mp_maxmoney"].SourceSequence == nil || first.Convars["cash_team_loser_bonus"].SourceSequence != nil || *first.Convars["mp_buytime"].Value != "" {
		t.Fatal("absent, empty and unseen collapsed")
	}
	if *first.Convars["mp_free_armor"].Value != "false" || len(tracker.ledger.Messages[0].Entries) != 5 || first.Sequence != 2 || last.Sequence != 4 {
		t.Fatal("native message order/coverage differs")
	}
	raw, _ := json.Marshal(tracker.ledger)
	if strings.Contains(string(raw), "private-name") || strings.Contains(string(raw), "hostname") {
		t.Fatal("unrelated convar retained")
	}
}

func TestNativeSessionProbeDistinguishesEmptyAbsentAndOpaque(t *testing.T) {
	tracker := newEconomyRuleTracker()
	tracker.session(&msg.CSVCMsg_ServerInfo{}, 0)
	tracker.session(&msg.CSVCMsg_ServerInfo{GameSessionManifest: []byte{}, GameSessionConfig: &msg.CSVCMsg_GameSessionConfiguration{Data: []byte("opaque-private-payload")}}, 1)
	one, two := tracker.ledger.Sessions[0], tracker.ledger.Sessions[1]
	if one.Manifest.Present || one.Manifest.Bytes != nil || one.Manifest.SHA256 != nil || one.ConfigurationPresent {
		t.Fatal("missing session became present")
	}
	if !two.Manifest.Present || *two.Manifest.Bytes != 0 || *two.Manifest.SHA256 != "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855" || !two.ConfigurationPresent || *two.ConfigurationData.Bytes != 22 {
		t.Fatal(two)
	}
	raw, _ := json.Marshal(tracker.ledger)
	if strings.Contains(string(raw), "private-payload") {
		t.Fatal("opaque bytes leaked")
	}
}

func TestNativeRuleProbeDoesNotInventMissingConvars(t *testing.T) {
	tracker := newEconomyRuleTracker()
	tracker.convars(&msg.CNETMsg_SetConVar{}, 0)
	tracker.boundary(1, 1, "round_start_probe", map[string]models.EconomyFact{})
	if len(tracker.ledger.Boundaries[0].Convars) != 56 || len(tracker.ledger.Messages) != 1 {
		t.Fatal("probe coverage lost")
	}
	for _, fact := range tracker.ledger.Boundaries[0].Convars {
		if fact.Value != nil || fact.SourceSequence != nil || fact.SourceTick != nil {
			t.Fatal("default imputed")
		}
	}
}
