package handlers

import (
	"cs2-demo-service/models"

	events "github.com/markus-wa/demoinfocs-golang/v4/pkg/demoinfocs/events"
)

// RegisterChatHandlers registra handlers para mensajes de chat
func RegisterChatHandlers(ctx *models.DemoContext) {
	ctx.Parser.RegisterEventHandler(func(e events.ChatMessage) {
		// Ignorar mensajes del sistema o vacíos si es necesario
		if e.Text == "" {
			return
		}

		senderName := "Console"
		var senderID uint64 = 0
		senderTeam := "Spectator"

		if e.Sender != nil {
			senderName = e.Sender.Name
			senderID = e.Sender.SteamID64

			switch e.Sender.Team {
			case 2:
				senderTeam = "T"
			case 3:
				senderTeam = "CT"
			}
		}

		chatEvent := models.ChatEvent{
			SenderName:    senderName,
			SenderSteamID: senderID,
			SenderTeam:    senderTeam,
			Text:          e.Text,
			IsTeamChat:    !e.IsChatAll, // Note: IsChatAll is true for global chat
		}

		event := models.TimelineEvent{
			Type:  "chat",
			Tick:  ctx.Parser.GameState().IngameTick(),
			Round: ctx.CurrentRound,
			Chat:  &chatEvent,
		}

		AddTimelineEvent(ctx, event)
	})
}
