package handlers

import (
	"fmt"
	"sort"
	"strings"

	"cs2-demo-service/models"
	"cs2-demo-service/pkg/playerstate"
	"github.com/markus-wa/demoinfocs-golang/v5/pkg/demoinfocs/common"
	"github.com/markus-wa/demoinfocs-golang/v5/pkg/demoinfocs/constants"
	"github.com/markus-wa/demoinfocs-golang/v5/pkg/demoinfocs/events"
	st "github.com/markus-wa/demoinfocs-golang/v5/pkg/demoinfocs/sendtables"
)

const economyWeaponsProperty = "m_pWeaponServices.m_hMyWeapons"

type economyObservationTracker struct {
	ledger           *models.EconomyObservationLedger
	current          int
	sequence         int
	actors           map[uint64]string
	entities         map[int64]string
	roster           map[string]string
	activity         map[string]bool
	priorDrops       map[string]models.EconomyCallbackEvent
	pending          []string
	initialFrontiers bool
	earlyCallbacks   []models.EconomyEarlyCallback
}

func newEconomyObservationTracker() *economyObservationTracker {
	return &economyObservationTracker{
		ledger:  &models.EconomyObservationLedger{Rounds: []models.EconomyObservedRound{}},
		current: -1, actors: map[uint64]string{}, entities: map[int64]string{},
		roster: map[string]string{}, activity: map[string]bool{}, priorDrops: map[string]models.EconomyCallbackEvent{},
	}
}

// This observer owns its ledger. It never writes AI_EconomyRounds, callbacks'
// players, inventories or the existing replay/canonical export state.
func RegisterEconomyObservationHandler(ctx *models.DemoContext) {
	registerEconomyObservationHandler(ctx, false)
}

func RegisterEconomyInitialFrontierHandler(ctx *models.DemoContext) {
	registerEconomyObservationHandler(ctx, true)
}

func registerEconomyObservationHandler(ctx *models.DemoContext, initialFrontiers bool) {
	tracker := newEconomyObservationTracker()
	tracker.initialFrontiers = initialFrontiers
	tracker.ledger.ActorAliases = tracker.actors
	ctx.EconomyObservations = tracker.ledger
	ctx.Parser.RegisterEventHandler(func(events.RoundStart) {
		gs := ctx.Parser.GameState()
		if gs.IsWarmupPeriod() || gs.IngameTick() < 0 {
			tracker.current = -1
			tracker.pending = nil
			return
		}
		tracker.startRound(gs.TotalRoundsPlayed()+1, gs.IngameTick())
		if initialFrontiers {
			tracker.capture(ctx, "round_start_probe")
			last := len(tracker.round().Snapshots) - 1
			probe := tracker.round().Snapshots[last]
			tracker.round().Snapshots = tracker.round().Snapshots[:last]
			nativeRules := map[string]models.EconomyFact{}
			for key, property := range map[string]string{
				"match_started": "m_pGameRules.m_bHasMatchStarted",
				"warmup":        "m_pGameRules.m_bWarmupPeriod",
				"freeze":        "m_pGameRules.m_bFreezePeriod",
			} {
				nativeRules[key] = nativeEconomyFact(gs.Rules().Entity(), property, gs.IngameTick(), true)
			}
			tracker.ledger.InitialFrontiers = append(tracker.ledger.InitialFrontiers, models.EconomyInitialFrontier{
				Instance: tracker.round().Instance, RoundStartProbe: &probe, NativeRules: nativeRules,
				PreProbeCallbacks: tracker.earlyCallbacksAt(gs.IngameTick()),
			})
		}
	})
	if initialFrontiers {
		ctx.Parser.RegisterEventHandler(func(events.RoundEndOfficial) {
			if tracker.current >= 0 && tracker.current < len(tracker.ledger.InitialFrontiers) {
				tracker.ledger.InitialFrontiers[tracker.current].OfficialEndTick = intPointer(ctx.Parser.GameState().IngameTick())
			}
		})
	}
	ctx.Parser.RegisterEventHandler(func(events.RoundFreezetimeEnd) {
		if tracker.current < 0 {
			return
		}
		tracker.sequence++
		tick := ctx.Parser.GameState().IngameTick()
		tracker.round().FreezeEndTick = intPointer(tick)
		tracker.pending = append(tracker.pending, "freeze_end")
	})
	ctx.Parser.RegisterEventHandler(func(events.RoundEnd) {
		if tracker.current < 0 {
			return
		}
		tracker.sequence++
		tick := ctx.Parser.GameState().IngameTick()
		tracker.round().EndTick = intPointer(tick)
		tracker.pending = append(tracker.pending, "round_end")
	})
	ctx.Parser.RegisterEventHandler(func(e events.ItemPickup) { tracker.observeEvent(ctx, "pickup", e.Player, e.Weapon) })
	ctx.Parser.RegisterEventHandler(func(e events.ItemDrop) { tracker.observeEvent(ctx, "drop", e.Player, e.Weapon) })
	ctx.Parser.RegisterEventHandler(func(e events.ItemRefund) { tracker.observeEvent(ctx, "refund", e.Player, e.Weapon) })
	ctx.Parser.RegisterEventHandler(func(events.FrameDone) {
		if tracker.current < 0 || len(tracker.pending) == 0 {
			return
		}
		pending := tracker.pending
		tracker.pending = nil
		for _, phase := range pending {
			tracker.capture(ctx, phase)
		}
	})
}

func (t *economyObservationTracker) round() *models.EconomyObservedRound {
	return &t.ledger.Rounds[t.current]
}

func (t *economyObservationTracker) startRound(number, tick int) {
	t.sequence++
	t.ledger.Rounds = append(t.ledger.Rounds, models.EconomyObservedRound{
		Instance: len(t.ledger.Rounds) + 1, RoundNumber: number, StartTick: tick,
		Snapshots: []models.EconomySnapshot{}, Events: []models.EconomyCallbackEvent{},
	})
	t.current = len(t.ledger.Rounds) - 1
	t.roster, t.activity, t.priorDrops = map[string]string{}, map[string]bool{}, map[string]models.EconomyCallbackEvent{}
	t.pending = []string{"initial_available"}
}

func (t *economyObservationTracker) actor(id uint64) string {
	if alias, ok := t.actors[id]; ok {
		return alias
	}
	alias := fmt.Sprintf("p%04d", len(t.actors)+1)
	t.actors[id] = alias
	return alias
}

func (t *economyObservationTracker) entity(id int64) *string {
	if id == 0 {
		return nil
	}
	if alias, ok := t.entities[id]; ok {
		return stringPointer(alias)
	}
	alias := fmt.Sprintf("w%05d", len(t.entities)+1)
	t.entities[id] = alias
	return stringPointer(alias)
}

func economySide(player *common.Player) string {
	if player.Team == common.TeamCounterTerrorists {
		return "CT"
	}
	if player.Team == common.TeamTerrorists {
		return "T"
	}
	return "unknown"
}

func (t *economyObservationTracker) observeEvent(ctx *models.DemoContext, kind string, player *common.Player, weapon *common.Equipment) {
	if player == nil || player.SteamID64 == 0 {
		return
	}
	tick := ctx.Parser.GameState().IngameTick()
	if t.initialFrontiers && tick >= 0 {
		t.sequence++
		t.rememberEarlyCallback(models.EconomyEarlyCallback{ActorAlias: t.actor(player.SteamID64), Kind: kind, Tick: tick, CallbackSequence: t.sequence})
	}
	if t.current < 0 || ctx.Parser.GameState().IsWarmupPeriod() {
		return
	}
	if tick < t.round().StartTick {
		return
	}
	alias := t.actor(player.SteamID64)
	if !t.initialFrontiers {
		t.sequence++
	}
	event := models.EconomyCallbackEvent{
		EventID: fmt.Sprintf("e%08d", t.sequence), Kind: kind, Tick: tick,
		CallbackSequence: t.sequence, ActorAlias: alias, Side: economySide(player),
		Attribution: "callback_only",
	}
	if weapon != nil {
		if name := strings.TrimSpace(weapon.String()); name != "" && weapon.Type != common.EqUnknown {
			event.Weapon = stringPointer(name)
		}
		event.EntityAlias = t.entity(playerstate.EquipmentID(weapon))
	}
	t.recordEvent(event)
}

func (t *economyObservationTracker) rememberEarlyCallback(event models.EconomyEarlyCallback) {
	if len(t.earlyCallbacks) > 0 && t.earlyCallbacks[0].Tick != event.Tick {
		t.earlyCallbacks = nil
	}
	t.earlyCallbacks = append(t.earlyCallbacks, event)
}

func (t *economyObservationTracker) earlyCallbacksAt(tick int) []models.EconomyEarlyCallback {
	result := []models.EconomyEarlyCallback{}
	for _, event := range t.earlyCallbacks {
		if event.Tick == tick {
			result = append(result, event)
		}
	}
	return result
}

func (t *economyObservationTracker) recordEvent(event models.EconomyCallbackEvent) {
	if event.Kind == "pickup" {
		event.Attribution = "purchase_or_pickup_unresolved"
		if event.EntityAlias != nil {
			if drop, ok := t.priorDrops[*event.EntityAlias]; ok && drop.ActorAlias != event.ActorAlias && drop.Side == event.Side && event.Side != "unknown" && drop.Tick <= event.Tick {
				event.PriorDropEventID = stringPointer(drop.EventID)
				event.Attribution = "prior_allied_drop_same_entity"
			}
			delete(t.priorDrops, *event.EntityAlias)
		}
	} else if event.Kind == "drop" && event.EntityAlias != nil {
		t.priorDrops[*event.EntityAlias] = event
	}
	t.round().Events = append(t.round().Events, event)
	// This flag is consumed only by the next initial snapshot, not applied to
	// snapshots already copied. Later activity cannot mutate the initial facts.
	t.activity[event.ActorAlias] = true
}

func missingEconomyFact(source string) models.EconomyFact {
	return models.EconomyFact{Status: "unavailable", Source: source}
}

func observedEconomyFact(value any, source string, tick int) models.EconomyFact {
	return models.EconomyFact{Value: value, Status: "observed", Source: source,
		ObservationTick: intPointer(tick), AvailabilityTick: intPointer(tick)}
}

func nativeEconomyFact(entity st.Entity, property string, tick int, boolean bool) models.EconomyFact {
	if entity == nil {
		return missingEconomyFact(property)
	}
	value, ok := entity.PropertyValue(property)
	if !ok || value.Any == nil {
		return missingEconomyFact(property)
	}
	if boolean {
		flag, valid := value.Any.(bool)
		if !valid {
			return missingEconomyFact(property)
		}
		return observedEconomyFact(flag, property, tick)
	}
	var amount int64
	switch native := value.Any.(type) {
	case int32:
		amount = int64(native)
	case uint32:
		amount = int64(native)
	case int64:
		amount = native
	case uint64:
		if native > 2147483647 {
			return missingEconomyFact(property)
		}
		amount = int64(native)
	default:
		return missingEconomyFact(property)
	}
	if amount < 0 || amount > 2147483647 {
		return missingEconomyFact(property)
	}
	return observedEconomyFact(int(amount), property, tick)
}

func (t *economyObservationTracker) inventory(ctx *models.DemoContext, player *common.Player, tick int) models.EconomyFact {
	pawn := player.PlayerPawnEntity()
	if pawn == nil {
		return missingEconomyFact(economyWeaponsProperty)
	}
	value, ok := pawn.PropertyValue(economyWeaponsProperty)
	if !ok || value.Any == nil {
		return missingEconomyFact(economyWeaponsProperty)
	}
	items := []models.EconomyInventoryItem{}
	seen := map[string]bool{}
	slots, valid := value.Any.([]any)
	if !valid || len(slots) > 64 {
		return missingEconomyFact(economyWeaponsProperty)
	}
	for i := range slots {
		handle, exists := pawn.PropertyValue(fmt.Sprintf("%s.%04d", economyWeaponsProperty, i))
		if !exists || handle.Any == nil {
			return missingEconomyFact(economyWeaponsProperty)
		}
		rawHandle, valid := handle.Any.(uint64)
		if !valid {
			return missingEconomyFact(economyWeaponsProperty)
		}
		index := rawHandle & constants.EntityHandleIndexMaskSource2
		if index == constants.EntityHandleIndexMaskSource2 {
			continue
		}
		weapon := ctx.Parser.GameState().Weapons()[int(index)]
		entity := ctx.Parser.GameState().EntityByHandle(rawHandle)
		if weapon == nil || entity == nil || weapon.Entity == nil || weapon.Entity.ID() != entity.ID() || weapon.Entity.SerialNum() != entity.SerialNum() || weapon.Type == common.EqUnknown {
			return missingEconomyFact(economyWeaponsProperty)
		}
		alias := t.entity(playerstate.EquipmentID(weapon))
		if alias == nil {
			return missingEconomyFact(economyWeaponsProperty)
		}
		if seen[*alias] {
			return missingEconomyFact(economyWeaponsProperty)
		}
		seen[*alias] = true
		items = append(items, models.EconomyInventoryItem{EntityAlias: *alias, Weapon: weapon.String()})
	}
	sort.Slice(items, func(i, j int) bool { return items[i].EntityAlias < items[j].EntityAlias })
	return observedEconomyFact(items, economyWeaponsProperty, tick)
}

var economyActorProperties = map[string]string{
	"money": "m_pInGameMoneyServices.m_iAccount", "spent_native": "m_pInGameMoneyServices.m_iCashSpentThisRound",
	"armor": "m_ArmorValue", "helmet": "m_pItemServices.m_bHasHelmet", "kit": "m_pItemServices.m_bHasDefuser", "buy_zone": "m_bInBuyZone",
	"equipment_value_native": "m_unCurrentEquipmentValue",
}

func unavailableEconomyActor(alias, side string) models.EconomyActorSnapshot {
	facts := map[string]models.EconomyFact{}
	for key, property := range economyActorProperties {
		facts[key] = missingEconomyFact(property)
	}
	facts["inventory"] = missingEconomyFact(economyWeaponsProperty)
	return models.EconomyActorSnapshot{ActorAlias: alias, Side: side, Facts: facts, PrepurchaseStatus: "unavailable", PrepurchaseReason: "participant_unavailable"}
}

func setEconomyPrepurchaseStatus(actor *models.EconomyActorSnapshot, phase string, startTick, tick int, activity bool) {
	actor.PrepurchaseStatus, actor.PrepurchaseReason = "unavailable", "not_initial_snapshot"
	if !actor.Connected {
		actor.PrepurchaseReason = "participant_unavailable"
		return
	}
	if phase != "initial_available" {
		return
	}
	if tick != startTick {
		actor.PrepurchaseReason = "initial_sample_after_round_start"
		return
	}
	if activity {
		actor.PrepurchaseReason = "activity_at_or_before_initial"
		return
	}
	for _, key := range []string{"money", "inventory", "spent_native"} {
		if actor.Facts[key].Status != "observed" {
			actor.PrepurchaseReason = "required_initial_fact_unavailable"
			return
		}
	}
	if actor.Facts["spent_native"].Value != 0 {
		actor.PrepurchaseReason = "spend_at_initial_observation"
		return
	}
	actor.PrepurchaseStatus, actor.PrepurchaseReason = "available", "observed_at_round_start_without_prior_activity"
}

func (t *economyObservationTracker) capture(ctx *models.DemoContext, phase string) {
	gs := ctx.Parser.GameState()
	tick := gs.IngameTick()
	if tick < t.round().StartTick {
		return
	}
	t.sequence++
	snapshot := models.EconomySnapshot{Phase: phase, Tick: tick, CallbackSequence: t.sequence,
		Context: map[string]models.EconomyFact{}, Actors: []models.EconomyActorSnapshot{}}
	rules := gs.Rules()
	for _, name := range []string{"mp_buytime", "mp_freezetime", "mp_maxmoney", "mp_startmoney", "mp_maxrounds", "mp_overtime_enable"} {
		fact := missingEconomyFact("convar:" + name)
		if value, ok := rules.ConVars()[name]; ok {
			fact = observedEconomyFact(value, "convar:"+name, tick)
		}
		snapshot.Context[name] = fact
	}
	for side, team := range map[string]*common.TeamState{"CT": gs.TeamCounterTerrorists(), "T": gs.TeamTerrorists()} {
		fact := missingEconomyFact("m_iScore")
		if team != nil && team.Entity != nil {
			fact = nativeEconomyFact(team.Entity, "m_iScore", tick, false)
		}
		snapshot.Context["score_"+side] = fact
	}
	seen := map[string]bool{}
	for _, player := range uniqueEconomyPlayers(gs.Participants().Playing()) {
		alias, side := t.actor(player.SteamID64), economySide(player)
		if side == "unknown" {
			continue
		}
		t.roster[alias], seen[alias] = side, true
		actor := unavailableEconomyActor(alias, side)
		actor.Connected = player.IsConnected
		if player.IsConnected && player.Entity != nil {
			for key, property := range economyActorProperties {
				entity := player.PlayerPawnEntity()
				if key == "money" || key == "spent_native" {
					entity = player.Entity
				}
				actor.Facts[key] = nativeEconomyFact(entity, property, tick, key == "helmet" || key == "kit" || key == "buy_zone")
			}
			actor.Facts["inventory"] = t.inventory(ctx, player, tick)
		}
		setEconomyPrepurchaseStatus(&actor, phase, t.round().StartTick, tick, t.activity[alias])
		if actor.Connected && phase == "initial_available" && t.round().FreezeEndTick != nil && *t.round().FreezeEndTick <= tick {
			actor.PrepurchaseStatus, actor.PrepurchaseReason = "unavailable", "freeze_closed_before_initial"
		}
		snapshot.Actors = append(snapshot.Actors, actor)
	}
	for alias, side := range t.roster {
		if !seen[alias] {
			snapshot.Actors = append(snapshot.Actors, unavailableEconomyActor(alias, side))
		}
	}
	sort.Slice(snapshot.Actors, func(i, j int) bool { return snapshot.Actors[i].ActorAlias < snapshot.Actors[j].ActorAlias })
	t.round().Snapshots = append(t.round().Snapshots, snapshot)
}
