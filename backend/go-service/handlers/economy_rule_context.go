package handlers

import (
	"crypto/sha256"
	"encoding/hex"

	"cs2-demo-service/models"
	"github.com/markus-wa/demoinfocs-golang/v5/pkg/demoinfocs/events"
	"github.com/markus-wa/demoinfocs-golang/v5/pkg/demoinfocs/msg"
)

type economyRuleTracker struct {
	manifests *[]models.EconomyManifestSession
	payloads  *models.EconomySessionPayloads
	ledger    *models.EconomyRuleContext
	probe     models.EconomyRuleProbe
	sequence  int
	latest    map[string]models.EconomyEffectiveConvar
}

func newEconomyRuleTracker() *economyRuleTracker {
	probe, _ := models.LoadEconomyRuleProbe()
	return &economyRuleTracker{probe: probe, latest: map[string]models.EconomyEffectiveConvar{}, ledger: &models.EconomyRuleContext{
		Messages: []models.EconomyConvarMessage{}, Boundaries: []models.EconomyRuleBoundary{}, Sessions: []models.EconomySessionProbe{},
	}}
}

func (t *economyRuleTracker) convars(message *msg.CNETMsg_SetConVar, tick int) {
	t.sequence++
	row := models.EconomyConvarMessage{Sequence: t.sequence, Tick: tick, Entries: []models.EconomyConvarEntry{}}
	allowed := map[string]bool{}
	for _, name := range t.probe.Convars {
		allowed[name] = true
	}
	// Read pointer presence directly. SDK ConVarsUpdated loses null vs empty and
	// collapses repeated assignments within one message into a map.
	for _, entry := range message.GetConvars().GetCvars() {
		if entry == nil || entry.Name == nil || !allowed[*entry.Name] {
			continue
		}
		var value *string
		if entry.Value != nil {
			copied := *entry.Value
			value = &copied
		}
		row.Entries = append(row.Entries, models.EconomyConvarEntry{Name: *entry.Name, Value: value})
		t.latest[*entry.Name] = models.EconomyEffectiveConvar{Value: value, SourceSequence: intPointer(t.sequence), SourceTick: intPointer(tick)}
	}
	t.ledger.Messages = append(t.ledger.Messages, row)
}

func opaqueEconomyField(data []byte) models.EconomyOpaqueField {
	if data == nil {
		return models.EconomyOpaqueField{}
	}
	digest := sha256.Sum256(data)
	encoded := hex.EncodeToString(digest[:])
	return models.EconomyOpaqueField{Present: true, Bytes: intPointer(len(data)), SHA256: &encoded}
}

func (t *economyRuleTracker) session(info *msg.CSVCMsg_ServerInfo, tick int) {
	t.sequence++
	if t.manifests != nil {
		*t.manifests = append(*t.manifests, decodeEconomyManifest(info.GameSessionManifest, t.sequence, tick))
	}
	t.ledger.Sessions = append(t.ledger.Sessions, models.EconomySessionProbe{
		Sequence: t.sequence, Tick: tick, ConfigurationPresent: info.GameSessionConfig != nil,
		Manifest: opaqueEconomyField(info.GameSessionManifest), ConfigurationData: opaqueEconomyField(info.GetGameSessionConfig().GetData()),
	})
	if t.payloads != nil {
		t.payloads.Sessions = append(t.payloads.Sessions, models.EconomySessionPayload{Sequence: t.sequence, Tick: tick, Blocks: []models.EconomySessionBlock{
			decodeEconomySessionBlock(info.GameSessionManifest, "CSVCMsg_ServerInfo.game_session_manifest", false),
			decodeEconomySessionBlock(info.GetGameSessionConfig().GetData(), "CSVCMsg_ServerInfo.game_session_config.data", true),
		}})
	}
}

func (t *economyRuleTracker) boundary(instance, tick int, phase string, rules map[string]models.EconomyFact) {
	t.sequence++
	state := map[string]models.EconomyEffectiveConvar{}
	for _, name := range t.probe.Convars {
		state[name] = t.latest[name]
	}
	t.ledger.Boundaries = append(t.ledger.Boundaries, models.EconomyRuleBoundary{
		Sequence: t.sequence, Tick: tick, Instance: instance, Phase: phase, Convars: state, Rules: rules,
	})
}

// This observes parser callback order, not server subtick order or player knowledge.
func RegisterEconomyRuleContextHandler(ctx *models.DemoContext) {
	registerEconomyRuleContext(ctx, false)
}

func registerEconomyRuleContext(ctx *models.DemoContext, decodeSession bool) {
	t := newEconomyRuleTracker()
	t.manifests = ctx.EconomyManifestSessions
	if decodeSession {
		t.payloads = &models.EconomySessionPayloads{Sessions: []models.EconomySessionPayload{}}
		ctx.EconomySessionPayloads = t.payloads
	}
	ctx.EconomyRuleContext = t.ledger
	ctx.Parser.RegisterNetMessageHandler(func(message *msg.CNETMsg_SetConVar) { t.convars(message, ctx.Parser.GameState().IngameTick()) })
	ctx.Parser.RegisterNetMessageHandler(func(info *msg.CSVCMsg_ServerInfo) { t.session(info, ctx.Parser.GameState().IngameTick()) })
	capture := func(phase string) {
		gs := ctx.Parser.GameState()
		if gs.IsWarmupPeriod() || gs.IngameTick() < 0 || ctx.EconomyObservations == nil || len(ctx.EconomyObservations.Rounds) == 0 {
			return
		}
		round := ctx.EconomyObservations.Rounds[len(ctx.EconomyObservations.Rounds)-1]
		if phase == "round_start_probe" && round.StartTick != gs.IngameTick() {
			return
		}
		if phase == "freeze_end_probe" && (round.FreezeEndTick == nil || *round.FreezeEndTick != gs.IngameTick()) {
			return
		}
		rules := map[string]models.EconomyFact{}
		for name, boolean := range t.probe.Rules {
			rules[name] = nativeEconomyFact(gs.Rules().Entity(), name, gs.IngameTick(), boolean)
		}
		t.boundary(round.Instance, gs.IngameTick(), phase, rules)
	}
	ctx.Parser.RegisterEventHandler(func(events.RoundStart) { capture("round_start_probe") })
	ctx.Parser.RegisterEventHandler(func(events.RoundFreezetimeEnd) { capture("freeze_end_probe") })
}
