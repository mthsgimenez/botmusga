package main

import (
	"log"
	"strings"

	"github.com/disgoorg/disgo/discord"
	"github.com/disgoorg/disgo/events"
)

func onReady(e *events.Ready) {
	log.Println("Bot is up and running!")
}

func onMessageCreate(e *events.MessageCreate) {
	if e.Message.Author.Bot {
		return
	}

	content := strings.Trim(e.Message.Content, " ")
	if !strings.HasPrefix(content, prefix) {
		return
	}

	command := strings.Fields(strings.TrimPrefix(content, prefix))

	switch command[0] {
	case "ping":
		e.Client().Rest().CreateMessage(e.ChannelID, discord.NewMessageCreateBuilder().SetContent("pong").Build())
	case "pong":
		e.Client().Rest().CreateMessage(e.ChannelID, discord.NewMessageCreateBuilder().SetContent("ping").Build())
	case "play":
		if len(command) < 2 {
			e.Client().Rest().CreateMessage(e.ChannelID, discord.NewMessageCreateBuilder().SetContent("Comando: `play <url>`").Build())
			return
		}
		go playMusic(e, command[1])
	}
}
