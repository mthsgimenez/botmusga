package main

import (
	"context"
	"fmt"
	"log"
        "io"
	"os/exec"
	"time"

	"github.com/disgoorg/disgo/discord"
	"github.com/disgoorg/disgo/events"
	"github.com/disgoorg/disgo/voice"
)

type UserNotInVoiceChannel struct {
	Message string
}

func (e *UserNotInVoiceChannel) Error() string {
	return fmt.Sprintln("Couldn't get channelID:", e.Message)
}

func playMusic(e *events.MessageCreate, url string) {
	state, connected := e.Client().Caches().VoiceState(*e.GuildID, e.Message.Author.ID)
	if !connected {
		e.Client().Rest().CreateMessage(e.ChannelID, discord.NewMessageCreateBuilder().SetContent("Você precisa estar em um canal de voz").Build())
		return
	}

	conn := e.Client().VoiceManager().CreateConn(state.GuildID)

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*10)
	defer cancel()

	err := conn.Open(ctx, *state.ChannelID, false, true)
	if err != nil {
		e.Client().Rest().CreateMessage(e.ChannelID, discord.NewMessageCreateBuilder().SetContent("Não foi possível se conectar ao canal").Build())
		log.Println("Error joining voice channel: ", err)
		return
	}
	defer conn.Close(context.TODO())

	err = conn.SetSpeaking(context.TODO(), voice.SpeakingFlagMicrophone)
	if err != nil {
		log.Println("Error setting voice flag: ", err)
	}

	cmd := exec.Command("yt-dlp", "-f", "bestaudio",
                "--quiet", "--no-progress", "--no-warnings",
                url,
                "--exec", "ffmpeg -i {} -threads 1 -c:a libopus -ac 2 -ar 48000 -b:a 96K -vbr off -f ogg - && rm {}",
	)

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		log.Println("Error getting audio stream from ffmpeg: ", err)
		return
	}

	err = cmd.Start()
	if err != nil {
		log.Println("Error starting yt-dlp: ", err)
		conn.Close(context.TODO())
		return
	}

	ticker := time.NewTicker(time.Millisecond * 20)
	defer ticker.Stop()

	decoder := NewDecoder(stdout)

	for range ticker.C {
		packet, err := decoder.GetPacket()
		if err != nil {
			if err == io.EOF || err == io.ErrUnexpectedEOF {
				break
			}
			log.Println(err)
		}

		conn.UDP().Write(packet)
	}
}
