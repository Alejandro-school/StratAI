package handlers

import (
	"fmt"
	"math"
	"sort"
	"strings"

	"cs2-demo-service/models"
	"github.com/markus-wa/demoinfocs-golang/v5/pkg/demoinfocs/events"
	"github.com/markus-wa/demoinfocs-golang/v5/pkg/demoinfocs/msg"
	st "github.com/markus-wa/demoinfocs-golang/v5/pkg/demoinfocs/sendtables"
)

const sellbackSource = "m_pBuyServices.m_vecSellbackPurchaseEntries"
const purchasesRoundSource = "m_pActionTrackingServices.m_weaponPurchasesThisRound.m_weaponPurchases"
const purchasesMatchSource = "m_pActionTrackingServices.m_weaponPurchasesThisMatch.m_weaponPurchases"

var purchaseMembers = []string{"m_nItemDefIndex", "m_nCount"}
var sellbackMembers = []string{"m_unDefIdx", "m_nCost", "m_nPrevArmor", "m_bPrevHelmet", "m_hItem"}

func sourceInteger(value any, max int64) (int64, bool) {
	var amount int64
	switch v := value.(type) {
	case int:
		amount = int64(v)
	case int32:
		amount = int64(v)
	case int64:
		amount = v
	case uint32:
		amount = int64(v)
	case uint64:
		if v > uint64(max) {
			return 0, false
		}
		amount = int64(v)
	default:
		return 0, false
	}
	return amount, amount >= 0 && amount <= max
}

func nativeSourceMember(entity st.Entity, source, member string, tick int) models.EconomyFact {
	if entity == nil {
		return missingEconomyFact(source)
	}
	property := entity.Property(source)
	if property == nil {
		return missingEconomyFact(source)
	}
	value := property.Value().Any
	if member == "m_nPrevArmor" {
		// Preserve native signed values and unsigned 32-bit encodings verbatim.
		// This is entry metadata, not a reconstructed armor amount.
		var n int64
		switch v := value.(type) {
		case int:
			n = int64(v)
		case int32:
			n = int64(v)
		case int64:
			n = v
		case uint32:
			n = int64(v)
		case uint64:
			if v > math.MaxUint32 {
				return missingEconomyFact(source)
			}
			n = int64(v)
		default:
			return missingEconomyFact(source)
		}
		if n >= math.MinInt32 && n <= math.MaxUint32 {
			return observedEconomyFact(n, source, tick)
		}
		return missingEconomyFact(source)
	}
	if member == "m_bPrevHelmet" {
		if b, ok := value.(bool); ok {
			return observedEconomyFact(b, source, tick)
		}
		if n, ok := sourceInteger(value, 1); ok {
			return observedEconomyFact(n == 1, source, tick)
		}
		return missingEconomyFact(source)
	}
	max := int64(math.MaxInt32)
	if member == "m_hItem" {
		max = math.MaxUint32
	}
	if member == "m_unDefIdx" || member == "m_nItemDefIndex" || member == "m_nCount" {
		max = math.MaxUint16
	}
	if n, ok := sourceInteger(value, max); ok {
		return observedEconomyFact(n, source, tick)
	}
	return missingEconomyFact(source)
}

func nativeSourceArray(entity st.Entity, root string, members []string, tick int) models.EconomyNativeArray {
	result := models.EconomyNativeArray{Status: "unavailable", Source: root, Reason: "native_array_unavailable", Entries: []map[string]models.EconomyFact{}}
	if entity == nil || entity.Property(root) == nil {
		return result
	}
	value := entity.Property(root).Value().Any
	if value == nil {
		return result
	}
	slots, ok := value.([]any)
	if !ok || len(slots) > 256 {
		result.Reason = "native_array_shape_unsupported"
		return result
	}
	result.Status, result.Reason, result.Length = "observed", "native_array_and_members_observed", intPointer(len(slots))
	for index := range slots {
		entry := map[string]models.EconomyFact{}
		for _, member := range members {
			path := fmt.Sprintf("%s.%04d.%s", root, index, member)
			entry[member] = nativeSourceMember(entity, path, member, tick)
			if entry[member].Status != "observed" {
				result.Status, result.Reason = "partial", "native_member_unavailable"
			}
		}
		result.Entries = append(result.Entries, entry)
	}
	return result
}

func sourceVersionFact(value *int32, source string, tick int) models.EconomyFact {
	if value == nil || *value <= 0 {
		return missingEconomyFact(source)
	}
	return observedEconomyFact(*value, source, tick)
}

// RegisterEconomySourceHandler observes native purchase counters and sellback
// entries at frame end. Changes between frames are not individual transactions.
// It is opt-in and must run after the v2 observer registers its callbacks.
func RegisterEconomySourceHandler(ctx *models.DemoContext) {
	ledger := &models.EconomyNativeSources{VersionObservations: []models.EconomyVersionObservation{}, Snapshots: []models.EconomySourceSnapshot{}, GenericEventCounts: map[string]int{}}
	ctx.EconomyNativeSources = ledger
	ctx.Parser.RegisterNetMessageHandler(func(header *msg.CDemoFileHeader) {
		tick := ctx.Parser.GameState().IngameTick()
		ledger.VersionObservations = append(ledger.VersionObservations, models.EconomyVersionObservation{Source: "CDemoFileHeader", Tick: tick, Fields: map[string]models.EconomyFact{
			"build_num":     sourceVersionFact(header.BuildNum, "CDemoFileHeader.build_num", tick),
			"patch_version": sourceVersionFact(header.PatchVersion, "CDemoFileHeader.patch_version", tick),
		}})
	})
	ctx.Parser.RegisterNetMessageHandler(func(info *msg.CSVCMsg_ServerInfo) {
		tick := ctx.Parser.GameState().IngameTick()
		ledger.VersionObservations = append(ledger.VersionObservations, models.EconomyVersionObservation{Source: "CSVCMsg_ServerInfo", Tick: tick, Fields: map[string]models.EconomyFact{
			"network_protocol": sourceVersionFact(info.Protocol, "CSVCMsg_ServerInfo.protocol", tick),
		}})
	})
	// Count only allowlisted event names; never retain names, chat or raw user IDs.
	ctx.Parser.RegisterEventHandler(func(event events.GenericGameEvent) {
		switch event.Name {
		case "item_purchase", "item_pickup", "item_refund", "buytime_ended":
			ledger.GenericEventCounts[event.Name]++
		}
	})
	previous := map[string]string{}
	previousTick := map[string]int{}
	rosters := map[int]map[string]string{}
	ctx.Parser.RegisterEventHandler(func(events.FrameDone) {
		observations := ctx.EconomyObservations
		if observations == nil || len(observations.Rounds) == 0 {
			return
		}
		round := observations.Rounds[len(observations.Rounds)-1]
		gs := ctx.Parser.GameState()
		tick := gs.IngameTick()
		if !nativeSourceTickAllowed(round, tick, gs.IsWarmupPeriod(), gs.IsFreezetimePeriod()) {
			return
		}
		if rosters[round.Instance] == nil {
			rosters[round.Instance] = map[string]string{}
		}
		seen := map[string]bool{}
		retain := func(snapshot models.EconomySourceSnapshot) {
			key := fmt.Sprintf("%d:%s", round.Instance, snapshot.ActorAlias)
			fingerprint := nativeSourceFingerprint(snapshot)
			boundary := tick == round.StartTick || (round.FreezeEndTick != nil && tick == *round.FreezeEndTick)
			if previous[key] == fingerprint && (!boundary || previousTick[key] == tick) {
				return
			}
			previous[key], previousTick[key] = fingerprint, tick
			snapshot.ObservationID = fmt.Sprintf("n%08d", len(ledger.Snapshots)+1)
			ledger.Snapshots = append(ledger.Snapshots, snapshot)
		}
		for _, player := range uniqueEconomyPlayers(gs.Participants().Playing()) {
			if player == nil || player.SteamID64 == 0 {
				continue
			}
			alias, exists := observations.ActorAliases[player.SteamID64]
			if !exists {
				continue
			}
			side := economySide(player)
			if side == "unknown" {
				continue
			}
			rosters[round.Instance][alias], seen[alias] = side, true
			ledger.SnapshotChecks++
			// Observe the purchase vectors every freeze frame; retain a new row
			// only when facts change or at a boundary. This also catches a net-zero
			// money change when its purchase counter or sellback entries change.
			retain(nativeSourceActor(round.Instance, tick, alias, side, player.IsConnected, player.Entity, player.PlayerPawnEntity()))
		}
		missing := []string{}
		for alias := range rosters[round.Instance] {
			if !seen[alias] {
				missing = append(missing, alias)
			}
		}
		sort.Strings(missing)
		for _, alias := range missing {
			ledger.SnapshotChecks++
			retain(nativeSourceActor(round.Instance, tick, alias, rosters[round.Instance][alias], false, nil, nil))
		}
	})
}

func nativeSourceTickAllowed(round models.EconomyObservedRound, tick int, warmup, freeze bool) bool {
	if warmup || tick < round.StartTick || (round.FreezeEndTick != nil && tick > *round.FreezeEndTick) {
		return false
	}
	return freeze || (round.FreezeEndTick != nil && tick == *round.FreezeEndTick)
}

func nativeSourceActor(instance, tick int, alias, side string, connected bool, controller, pawn st.Entity) models.EconomySourceSnapshot {
	if !connected {
		controller, pawn = nil, nil
	}
	return models.EconomySourceSnapshot{Instance: instance, Tick: tick, ActorAlias: alias, Side: side, Connected: connected,
		Money:          nativeEconomyFact(controller, "m_pInGameMoneyServices.m_iAccount", tick, false),
		SpentNative:    nativeEconomyFact(controller, "m_pInGameMoneyServices.m_iCashSpentThisRound", tick, false),
		Sellback:       nativeSourceArray(pawn, sellbackSource, sellbackMembers, tick),
		PurchasesRound: nativeSourceArray(pawn, purchasesRoundSource, purchaseMembers, tick),
		PurchasesMatch: nativeSourceArray(pawn, purchasesMatchSource, purchaseMembers, tick)}
}

func nativeSourceFingerprint(snapshot models.EconomySourceSnapshot) string {
	// Timestamp-independent comparison preserves value, presence, roster and
	// native array shape without making the previous row a new observation.
	var b strings.Builder
	fmt.Fprintf(&b, "%s:%t|", snapshot.Side, snapshot.Connected)
	fact := func(value models.EconomyFact) { fmt.Fprintf(&b, "%s:%v|", value.Status, value.Value) }
	fact(snapshot.Money)
	fact(snapshot.SpentNative)
	for index, array := range []models.EconomyNativeArray{snapshot.Sellback, snapshot.PurchasesRound, snapshot.PurchasesMatch} {
		length := -1
		if array.Length != nil {
			length = *array.Length
		}
		fmt.Fprintf(&b, "%s:%s:%d|", array.Status, array.Reason, length)
		members := purchaseMembers
		if index == 0 {
			members = sellbackMembers
		}
		for _, entry := range array.Entries {
			for _, member := range members {
				fact(entry[member])
			}
		}
	}
	return b.String()
}
