package handlers

import (
	"crypto/sha256"
	"fmt"
	"math"

	"cs2-demo-service/models"
	"github.com/markus-wa/demoinfocs-golang/v5/pkg/demoinfocs/common"
	"github.com/markus-wa/demoinfocs-golang/v5/pkg/demoinfocs/constants"
	"github.com/markus-wa/demoinfocs-golang/v5/pkg/demoinfocs/events"
	"github.com/markus-wa/demoinfocs-golang/v5/pkg/demoinfocs/msg"
	st "github.com/markus-wa/demoinfocs-golang/v5/pkg/demoinfocs/sendtables"
)

const identityInventory = "m_pWeaponServices.m_hMyWeapons"

type identityEntity struct {
	alias, kind string
	entity      st.Entity
	frame       int
	alive       bool
}
type economyIdentityTracker struct {
	ctx                  *models.DemoContext
	ledger               *models.EconomyNativeIdentity
	sequence, frame      int
	entities             map[st.Entity]*identityEntity
	live                 map[int]*identityEntity
	itemIDs              map[int64]string
	ownershipFingerprint string
	entryFingerprint     string
}

func identityEpoch(previous *models.EconomyIdentityClock, tick *uint32, demoTick int) (int, string) {
	if previous == nil {
		return 1, "first_sample"
	}
	if previous.NetworkTick == nil || tick == nil {
		return previous.Epoch + 1, "missing_tick_boundary"
	}
	if demoTick < previous.DemoTick {
		return previous.Epoch + 1, "demo_tick_regression"
	}
	if *tick < *previous.NetworkTick {
		return previous.Epoch + 1, "network_tick_regression"
	}
	return previous.Epoch, "continuous"
}

func (t *economyIdentityTracker) clock(m *msg.CNETMsg_Tick) {
	t.sequence++
	var previous *models.EconomyIdentityClock
	if len(t.ledger.Clocks) > 0 {
		previous = &t.ledger.Clocks[len(t.ledger.Clocks)-1]
	}
	var tick *uint32
	if m.Tick != nil {
		v := *m.Tick
		tick = &v
	}
	epoch, reason := identityEpoch(previous, tick, t.ctx.Parser.GameState().IngameTick())
	t.ledger.Clocks = append(t.ledger.Clocks, models.EconomyIdentityClock{Sequence: t.sequence, DemoTick: t.ctx.Parser.GameState().IngameTick(), Frame: t.frame, NetworkTick: tick, Epoch: epoch, EpochReason: reason})
}

func (t *economyIdentityTracker) append(kind, source string, e *identityEntity, player *common.Player, value any) {
	t.sequence++
	r := models.EconomyIdentityRow{Sequence: t.sequence, DemoTick: t.ctx.Parser.GameState().IngameTick(), Frame: t.frame, Kind: kind, Source: source, Value: value}
	if n := len(t.ledger.Clocks); n > 0 {
		seq := t.ledger.Clocks[n-1].Sequence
		r.ClockSequence = &seq
	}
	if e != nil {
		alias := e.alias
		r.EntityAlias = &alias
	}
	if player != nil && player.SteamID64 != 0 {
		if alias := t.ctx.EconomyObservations.ActorAliases[player.SteamID64]; alias != "" {
			r.ActorAlias = &alias
		}
		side := economySide(player)
		if side == "CT" || side == "T" {
			r.Side = &side
		}
	}
	if rounds := t.ctx.EconomyObservations.Rounds; len(rounds) > 0 && !t.ctx.Parser.GameState().IsWarmupPeriod() {
		last := rounds[len(rounds)-1]
		if r.DemoTick >= last.StartTick {
			instance := last.Instance
			r.Instance = &instance
		}
	}
	t.ledger.Rows = append(t.ledger.Rows, r)
}

// Match the full serial as in sendtablescs2.FindEntityByHandle, rather than
// the index-only GameState/Participants helpers. Never resolve from the future.
func (t *economyIdentityTracker) reference(v any, kind string) models.EconomyEntityReference {
	r := models.EconomyEntityReference{Status: "absent"}
	if v == nil {
		return r
	}
	n, ok := sourceInteger(v, math.MaxUint32)
	if !ok {
		r.Status = "unsupported_type"
		return r
	}
	h := uint64(n)
	r.Handle = &h
	index := int(h & constants.EntityHandleIndexMaskSource2)
	if index == constants.EntityHandleIndexMaskSource2 {
		r.Status = "invalid_sentinel"
		return r
	}
	e := t.live[index]
	if e == nil || !e.alive {
		r.Status = "entity_unavailable"
		return r
	}
	if uint64(e.entity.SerialNum()) != h>>constants.MaxEdictBitsSource2 {
		r.Status = "serial_mismatch"
		return r
	}
	if e.kind != kind {
		r.Status = "kind_mismatch"
		return r
	}
	r.Status = "resolved"
	a := e.alias
	r.TargetAlias = &a
	return r
}

func (t *economyIdentityTracker) playerForPawn(pawn *identityEntity) *common.Player {
	if pawn == nil || !pawn.alive || pawn.kind != "pawn" {
		return nil
	}
	var found *common.Player
	for _, p := range t.ctx.Parser.GameState().Participants().All() {
		if p == nil || p.Entity == nil {
			continue
		}
		v, ok := p.Entity.PropertyValue("m_hPlayerPawn")
		if !ok {
			continue
		}
		r := t.reference(v.Any, "pawn")
		if r.TargetAlias != nil && *r.TargetAlias == pawn.alias {
			if found != nil {
				return nil
			}
			found = p
		}
	}
	return found
}

func (t *economyIdentityTracker) actorForReference(r models.EconomyEntityReference) *common.Player {
	if r.Status != "resolved" {
		return nil
	}
	return t.playerForPawn(t.live[int(*r.Handle&constants.EntityHandleIndexMaskSource2)])
}

func (t *economyIdentityTracker) entityCreated(entity st.Entity, kind string) {
	e := &identityEntity{alias: fmt.Sprintf("native-entity-%06d", len(t.entities)+1), kind: kind, entity: entity, frame: t.frame, alive: true}
	t.entities[entity] = e
	t.live[entity.ID()] = e
	t.append("entity_created", entity.ServerClass().Name(), e, nil, map[string]any{"entity_index": entity.ID(), "entity_serial": entity.SerialNum(), "entity_kind": kind})
	phase := func() string {
		if t.frame == e.frame {
			return "creation_frame_or_replay"
		}
		return "later_frame_callback"
	}
	bindRef := func(path, targetKind string) {
		prop := entity.Property(path)
		if prop == nil {
			t.append("probe", path, e, nil, "property_absent")
			return
		}
		capture := func(v any, channel string) {
			ref := t.reference(v, targetKind)
			player := t.playerForPawn(e)
			if targetKind == "pawn" {
				player = t.actorForReference(ref)
			}
			t.append(channel, path, e, player, map[string]any{"phase": phase(), "reference": ref})
		}
		capture(prop.Value().Any, "reference_baseline")
		prop.OnUpdate(func(v st.PropertyValue) { capture(v.Any, "reference_callback") })
	}
	if kind == "equipment" {
		bindRef("m_hOwnerEntity", "pawn")
		// Definition identifies a type, not an instance or purchase.
		prop := entity.Property("m_iItemDefinitionIndex")
		if prop != nil {
			capture := func(v any, k string) {
				n, status := traceScalar(v)
				t.append(k, "m_iItemDefinitionIndex", e, nil, map[string]any{"status": status, "value": n, "phase": phase()})
			}
			capture(prop.Value().Any, "definition_baseline")
			prop.OnUpdate(func(v st.PropertyValue) { capture(v.Any, "definition_callback") })
		}
	} else {
		prop := entity.Property(identityInventory)
		if prop == nil {
			t.append("probe", identityInventory, e, nil, "property_absent")
		} else {
			capture := func(v any, k string) {
				var length *int
				status := "absent"
				if v != nil {
					if a, ok := v.([]any); ok {
						n := len(a)
						length = &n
						status = "observed"
					} else {
						status = "unsupported_type"
					}
				}
				t.append(k, identityInventory, e, t.playerForPawn(e), map[string]any{"status": status, "length": length, "phase": phase()})
			}
			capture(prop.Value().Any, "inventory_length_baseline")
			prop.OnUpdate(func(v st.PropertyValue) { capture(v.Any, "inventory_length_callback") })
		}
		for slot := 0; slot < 64; slot++ {
			bindRef(fmt.Sprintf("%s.%04d", identityInventory, slot), "equipment")
		}
	}
	entity.OnDestroy(func() {
		t.append("entity_destroyed", entity.ServerClass().Name(), e, nil, nil)
		e.alive = false
		if t.live[entity.ID()] == e {
			delete(t.live, entity.ID())
		}
	})
}

func (t *economyIdentityTracker) traceEvent(source string, weapon *common.Equipment, serverTick *int32) {
	var e *identityEntity
	if weapon != nil {
		e = t.entities[weapon.Entity]
	}
	var traceSequence *int
	var player *common.Player
	rows := t.ctx.EconomyTransactionTrace.Rows
	if len(rows) > 0 {
		last := rows[len(rows)-1]
		if last.Source == source && last.Tick == t.ctx.Parser.GameState().IngameTick() && last.Frame == t.frame {
			seq := last.Sequence
			traceSequence = &seq
			// Copy only already observed aliases below; never create one in v1.
			t.append("trace_event", source, e, player, map[string]any{"trace_sequence": traceSequence, "server_tick": serverTick})
			r := &t.ledger.Rows[len(t.ledger.Rows)-1]
			r.ActorAlias = last.ActorAlias
			r.Side = last.Side
			r.Instance = last.Instance
			return
		}
	}
	t.append("trace_event", source, e, nil, map[string]any{"trace_sequence": nil, "server_tick": serverTick})
}

func (t *economyIdentityTracker) pickupMessage(source string, item *string) {
	t.ledger.UserMessageCounts[source]++
	status := "absent"
	var value *string
	if item != nil {
		if safeTraceItem(*item) {
			s := *item
			value = &s
			status = "observed"
		} else {
			status = "outside_probe"
		}
	}
	t.append("user_message", source, nil, nil, map[string]any{"item": value, "status": status, "actor_status": "not_in_message"})
}

func RegisterEconomyNativeIdentityHandler(ctx *models.DemoContext) {
	registerEconomyNativeIdentity(ctx, false)
}

func RegisterEconomyOwnershipHandler(ctx *models.DemoContext) {
	registerEconomyNativeIdentity(ctx, true)
}

func registerEconomyNativeIdentity(ctx *models.DemoContext, ownership bool) {
	t := &economyIdentityTracker{ctx: ctx, ledger: &models.EconomyNativeIdentity{Clocks: []models.EconomyIdentityClock{}, Rows: []models.EconomyIdentityRow{}, UserMessageCounts: map[string]int{"CUserMessageItemPickup": 0, "CCSUsrMsg_ItemPickup": 0, "CCSUsrMsg_ItemDrop": 0, "CSVCMsg_UserMessage:economic": 0}}, entities: map[st.Entity]*identityEntity{}, live: map[int]*identityEntity{}, itemIDs: map[int64]string{}}
	ctx.EconomyNativeIdentity = t.ledger
	if ownership {
		ctx.EconomyOwnershipObservations = &models.EconomyOwnershipObservations{Cuts: []models.EconomyOwnershipCut{}, States: []models.EconomyOwnershipState{}}
	}
	ctx.Parser.RegisterNetMessageHandler(t.clock)
	descriptors := map[int32]string{}
	ctx.Parser.RegisterNetMessageHandler(func(m *msg.CMsgSource1LegacyGameEventList) {
		descriptors = map[int32]string{}
		for _, d := range m.Descriptors {
			if d != nil && d.Eventid != nil {
				descriptors[*d.Eventid] = d.GetName()
			}
		}
	})
	ctx.Parser.RegisterNetMessageHandler(func(m *msg.CMsgSource1LegacyGameEvent) {
		if m.Eventid != nil && economyTraceNames[descriptors[*m.Eventid]] {
			t.traceEvent(descriptors[*m.Eventid], nil, m.ServerTick)
		}
	})
	ctx.Parser.RegisterEventHandler(func(e events.ItemPickup) { t.traceEvent("ItemPickup", e.Weapon, nil) })
	ctx.Parser.RegisterEventHandler(func(e events.ItemDrop) { t.traceEvent("ItemDrop", e.Weapon, nil) })
	ctx.Parser.RegisterEventHandler(func(e events.ItemRefund) { t.traceEvent("ItemRefund", e.Weapon, nil) })
	ctx.Parser.RegisterNetMessageHandler(func(m *msg.CUserMessageItemPickup) { t.pickupMessage("CUserMessageItemPickup", m.Itemname) })
	ctx.Parser.RegisterNetMessageHandler(func(m *msg.CCSUsrMsg_ItemPickup) { t.pickupMessage("CCSUsrMsg_ItemPickup", m.Item) })
	ctx.Parser.RegisterNetMessageHandler(func(m *msg.CCSUsrMsg_ItemDrop) {
		t.ledger.UserMessageCounts["CCSUsrMsg_ItemDrop"]++
		var item *string
		if m.Itemid != nil {
			alias := t.itemIDs[*m.Itemid]
			if alias == "" {
				alias = fmt.Sprintf("native-item-%06d", len(t.itemIDs)+1)
				t.itemIDs[*m.Itemid] = alias
			}
			item = &alias
		}
		t.append("user_message", "CCSUsrMsg_ItemDrop", nil, nil, map[string]any{"item_id_alias": item, "death": m.Death, "actor_status": "not_in_message", "entity_binding": "not_accredited"})
	})
	ctx.Parser.RegisterNetMessageHandler(func(m *msg.CSVCMsg_UserMessage) {
		if m.MsgType == nil {
			return
		}
		typ := *m.MsgType
		if typ != int32(msg.EBaseUserMessages_UM_ItemPickup) && typ != int32(msg.ECstrike15UserMessages_CS_UM_ItemPickup) && typ != int32(msg.ECstrike15UserMessages_CS_UM_ItemDrop) {
			return
		}
		t.ledger.UserMessageCounts["CSVCMsg_UserMessage:economic"]++
		var fingerprint *string
		if m.MsgData != nil {
			s := fmt.Sprintf("%x", sha256.Sum256(m.MsgData))
			fingerprint = &s
		}
		t.append("user_message", "CSVCMsg_UserMessage:economic", nil, nil, map[string]any{"message_type": typ, "payload_bytes": len(m.MsgData), "payload_sha256": fingerprint, "actor_status": "not_in_message", "payload_interpretation": "not_decoded"})
	})
	ctx.Parser.RegisterEventHandler(func(events.DataTablesParsed) {
		for _, class := range ctx.Parser.ServerClasses().All() {
			kind := ""
			if class.Name() == "CCSPlayerPawn" {
				kind = "pawn"
			} else {
				index, clip := false, false
				for _, p := range class.PropertyEntries() {
					if p == "m_iItemDefinitionIndex" {
						index = true
					}
					if p == "m_iClip1" {
						clip = true
					}
				}
				if index && clip {
					kind = "equipment"
				}
			}
			if kind != "" {
				class.OnEntityCreated(func(e st.Entity) { t.entityCreated(e, kind) })
			}
		}
	})
	ctx.Parser.RegisterEventHandler(func(events.RoundStart) { t.append("round_boundary", "RoundStart", nil, nil, nil) })
	ctx.Parser.RegisterEventHandler(func(events.RoundFreezetimeEnd) { t.append("round_boundary", "RoundFreezetimeEnd", nil, nil, nil) })
	ctx.Parser.RegisterEventHandler(func(events.RoundEndOfficial) { t.append("round_boundary", "RoundEndOfficial", nil, nil, nil) })
	ctx.Parser.RegisterEventHandler(func(events.FrameDone) {
		if ownership {
			t.captureOwnership()
		}
		t.frame++
	})
}
