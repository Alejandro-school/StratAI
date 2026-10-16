package handlers

import (
	"fmt"
	"math"
	"strings"

	"cs2-demo-service/models"
	"github.com/markus-wa/demoinfocs-golang/v5/pkg/demoinfocs/common"
	"github.com/markus-wa/demoinfocs-golang/v5/pkg/demoinfocs/events"
	"github.com/markus-wa/demoinfocs-golang/v5/pkg/demoinfocs/msg"
	st "github.com/markus-wa/demoinfocs-golang/v5/pkg/demoinfocs/sendtables"
)

var economyTraceNames = map[string]bool{"item_purchase": true, "item_pickup": true, "item_pickup_slerp": true, "item_remove": true, "item_refund": true}
var economyTraceProperties = []string{"m_pInGameMoneyServices.m_iAccount", "m_pInGameMoneyServices.m_iCashSpentThisRound"}

type economyTraceTracker struct {
	ctx         *models.DemoContext
	ledger      *models.EconomyTransactionTrace
	frame       int
	descriptors map[int32]*msg.CMsgSource1LegacyGameEventListDescriptorT
	entities    map[st.Entity]string
}

func (t *economyTraceTracker) append(channel, source string, player *common.Player, entity *string, value any) {
	row := models.EconomyTraceRow{Sequence: len(t.ledger.Rows) + 1, Tick: t.ctx.Parser.GameState().IngameTick(), Frame: t.frame, Channel: channel, Source: source, EntityAlias: entity, Value: value}
	if player != nil && player.SteamID64 != 0 {
		if alias := t.ctx.EconomyObservations.ActorAliases[player.SteamID64]; alias != "" {
			row.ActorAlias = &alias
		}
		side := economySide(player)
		if side == "CT" || side == "T" {
			row.Side = &side
		}
	}
	// Only the currently observed ledger instance is retained. Readers additionally
	// enforce strict tick frontiers and side membership; repeated round numbers do not join.
	if rounds := t.ctx.EconomyObservations.Rounds; len(rounds) > 0 && !t.ctx.Parser.GameState().IsWarmupPeriod() {
		r := rounds[len(rounds)-1]
		if row.Tick >= r.StartTick {
			instance := r.Instance
			row.Instance = &instance
		}
	}
	t.ledger.Rows = append(t.ledger.Rows, row)
}

func (t *economyTraceTracker) entityAlias(entity st.Entity) *string {
	if entity == nil {
		return nil
	}
	alias := t.entities[entity]
	if alias == "" {
		alias = fmt.Sprintf("trace-entity-%06d", len(t.entities)+1)
		t.entities[entity] = alias
	}
	return &alias
}

func (t *economyTraceTracker) callback(kind string, player *common.Player, weapon *common.Equipment) {
	var entity *string
	var item *string
	if weapon != nil {
		entity = t.entityAlias(weapon.Entity)
		if weapon.Type != common.EqUnknown {
			name := weapon.String()
			item = &name
		}
	}
	t.append("sdk_callback", kind, player, entity, map[string]any{"item": item})
}

// Explicit whitelisting protects identifiers and arbitrary payload text. Event
// scalar zero/false/empty and absent pointers remain distinct; repeated keys stay ordered.
func traceField(ordinal int, descriptor *msg.CMsgSource1LegacyGameEventListKeyT, key *msg.CMsgSource1LegacyGameEventKeyT) models.EconomyTraceField {
	f := models.EconomyTraceField{Ordinal: ordinal, Status: "absent"}
	if descriptor != nil {
		f.Name = descriptor.Name
		f.DescriptorType = descriptor.Type
	}
	if key == nil {
		return f
	}
	f.WireType = key.Type
	if descriptor == nil || descriptor.Name == nil {
		f.Status = "unmapped"
		return f
	}
	name := descriptor.GetName()
	if name == "userid" {
		f.Status = "identity_redacted"
		return f
	}
	allowed := map[string]bool{"item": true, "defindex": true, "silent": true, "inventory": true, "canbuy": true, "team": true}
	if !allowed[name] {
		f.Status = "outside_probe"
		return f
	}
	if key.Type == nil || descriptor.Type == nil || key.GetType() != descriptor.GetType() {
		f.Status = "type_mismatch"
		return f
	}
	switch key.GetType() {
	case 1:
		if key.ValString != nil && (name != "item" || safeTraceItem(*key.ValString)) {
			f.Value = *key.ValString
		}
	case 2:
		if key.ValFloat != nil && !math.IsNaN(float64(*key.ValFloat)) && !math.IsInf(float64(*key.ValFloat), 0) {
			f.Value = *key.ValFloat
		}
	case 3:
		if key.ValLong != nil {
			f.Value = *key.ValLong
		}
	case 4:
		if key.ValShort != nil {
			f.Value = *key.ValShort
		}
	case 5:
		if key.ValByte != nil {
			f.Value = *key.ValByte
		}
	case 6:
		if key.ValBool != nil {
			f.Value = *key.ValBool
		}
	case 7:
		if key.ValUint64 != nil {
			f.Value = fmt.Sprint(*key.ValUint64)
		}
	}
	if f.Value != nil {
		f.Status = "observed"
	}
	return f
}

func safeTraceItem(s string) bool {
	return len(s) <= 96 && strings.IndexFunc(s, func(r rune) bool {
		return !(r >= 'a' && r <= 'z' || r >= 'A' && r <= 'Z' || r >= '0' && r <= '9' || r == '_' || r == '-')
	}) < 0
}

func (t *economyTraceTracker) native(message *msg.CMsgSource1LegacyGameEvent) {
	d := t.descriptors[message.GetEventid()]
	if message.Eventid == nil || d == nil {
		t.ledger.UnmappedNativeEvents++
		return
	}
	if !economyTraceNames[d.GetName()] {
		return
	}
	fields := []models.EconomyTraceField{}
	var player *common.Player
	userIDs := 0
	for i := 0; i < max(len(d.Keys), len(message.Keys)); i++ {
		var dk *msg.CMsgSource1LegacyGameEventListKeyT
		var key *msg.CMsgSource1LegacyGameEventKeyT
		if i < len(d.Keys) {
			dk = d.Keys[i]
		}
		if i < len(message.Keys) {
			key = message.Keys[i]
		}
		fields = append(fields, traceField(i, dk, key))
		if dk != nil && dk.GetName() == "userid" {
			userIDs++
			// itemEvent reads ValShort even for the observed CS2 descriptor type 9.
			// Keep native type untouched; this is the SDK's actor mapping, not a cast.
			if traceUserIDSupported(dk, key) {
				id := int(*key.ValShort)
				if id >= 0 && id <= math.MaxUint16 {
					id &= 0xff
				}
				player = t.ctx.Parser.GameState().Participants().ByUserID()[id]
			}
		}
	}
	if userIDs != 1 {
		player = nil
	}
	t.append("native_event", d.GetName(), player, nil, map[string]any{"server_tick": message.ServerTick, "fields": fields})
}

func traceUserIDSupported(d *msg.CMsgSource1LegacyGameEventListKeyT, key *msg.CMsgSource1LegacyGameEventKeyT) bool {
	return d != nil && d.Type != nil && key != nil && key.Type != nil &&
		(d.GetType() == 4 || d.GetType() == 9) && d.GetType() == key.GetType() && key.ValShort != nil
}

func traceScalar(value any) (any, string) {
	switch v := value.(type) {
	case int:
		return v, "observed"
	case int32:
		return v, "observed"
	case int64:
		return v, "observed"
	case uint32:
		return v, "observed"
	case uint64:
		if v <= math.MaxInt64 {
			return v, "observed"
		}
	case nil:
		return nil, "absent"
	}
	return nil, "unsupported_type"
}

func RegisterEconomyTransactionTraceHandler(ctx *models.DemoContext) {
	t := &economyTraceTracker{ctx: ctx, ledger: &models.EconomyTransactionTrace{Rows: []models.EconomyTraceRow{}}, descriptors: map[int32]*msg.CMsgSource1LegacyGameEventListDescriptorT{}, entities: map[st.Entity]string{}}
	ctx.EconomyTransactionTrace = t.ledger
	ctx.Parser.RegisterNetMessageHandler(func(m *msg.CMsgSource1LegacyGameEventList) {
		t.ledger.NativeDescriptorLists++
		t.descriptors = map[int32]*msg.CMsgSource1LegacyGameEventListDescriptorT{}
		for _, d := range m.GetDescriptors() {
			if d != nil && d.Eventid != nil {
				t.descriptors[*d.Eventid] = d
			}
		}
	})
	ctx.Parser.RegisterNetMessageHandler(t.native)
	ctx.Parser.RegisterEventHandler(func(e events.GenericGameEvent) {
		if economyTraceNames[e.Name] {
			t.append("sdk_generic", e.Name, nil, nil, nil)
		}
	})
	ctx.Parser.RegisterEventHandler(func(e events.ItemPickup) { t.callback("ItemPickup", e.Player, e.Weapon) })
	ctx.Parser.RegisterEventHandler(func(e events.ItemDrop) { t.callback("ItemDrop", e.Player, e.Weapon) })
	ctx.Parser.RegisterEventHandler(func(e events.ItemRefund) { t.callback("ItemRefund", e.Player, e.Weapon) })
	ctx.Parser.RegisterEventHandler(func(events.FrameDone) {
		t.frame++
		if rt := ctx.EconomyCashflowRuntime; rt != nil {
			rt.NextFrame = t.frame
		}
	})
	ctx.Parser.RegisterEventHandler(func(events.DataTablesParsed) {
		class := ctx.Parser.ServerClasses().FindByName("CCSPlayerController")
		if class == nil {
			t.append("probe", "CCSPlayerController", nil, nil, "class_absent")
			return
		}
		class.OnEntityCreated(func(entity st.Entity) {
			alias := t.entityAlias(entity)
			t.cashflowController(entity, *alias)
			for _, path := range economyTraceProperties {
				prop := entity.Property(path)
				player := func() *common.Player { return ctx.Parser.GameState().Participants().ByEntityID()[entity.ID()] }
				if prop == nil {
					t.append("probe", path, player(), alias, "property_absent")
					continue
				}
				// CS2 OnUpdate does NOT send an initial value (unlike its interface
				// comment). Read it explicitly and label it as a snapshot, not an update.
				value, status := traceScalar(prop.Value().Any)
				t.append("property_baseline", path, player(), alias, map[string]any{"status": status, "value": value})
				t.cashflowDelivery(entity)
				prop.OnUpdate(func(v st.PropertyValue) {
					value, status := traceScalar(v.Any)
					t.append("property_update", path, player(), alias, map[string]any{"status": status, "value": value})
					t.cashflowDelivery(entity)
				})
			}
		})
	})
}
