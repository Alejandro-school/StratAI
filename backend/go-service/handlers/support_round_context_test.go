package handlers

import (
	"cs2-demo-service/models"
	"github.com/markus-wa/demoinfocs-golang/v5/pkg/demoinfocs/msg"
	"testing"
)

func TestRoundConfigurationPreservesNullEmptyDuplicatesAndCopies(t *testing.T) {
	tracker := &supportRoundTracker{data: &models.SupportRoundContextObservations{}}
	value := ""
	tracker.convars(ruleMessage(
		&msg.CMsg_CVars_CVar{Name: ruleString("mp_t_default_secondary"), Value: &value},
		&msg.CMsg_CVars_CVar{Name: ruleString("mp_t_default_secondary")},
		&msg.CMsg_CVars_CVar{Name: ruleString("hostname"), Value: ruleString("private")},
	), 10, 1)
	value = "weapon_glock"
	rows := tracker.data.Messages
	if len(rows) != 1 || len(rows[0].Entries) != 2 || *rows[0].Entries[0].Value != "" || rows[0].Entries[1].Value != nil || rows[0].Frame != 10 {
		t.Fatal("lost native configuration presence/order")
	}
	tracker.session++
	tracker.convars(ruleMessage(&msg.CMsg_CVars_CVar{Name: ruleString("mp_give_player_c4"), Value: ruleString("1")}), 11, 2)
	if tracker.data.Messages[1].Session != 1 || tracker.data.Messages[0].Session != 0 {
		t.Fatal("session context changed history")
	}
}
