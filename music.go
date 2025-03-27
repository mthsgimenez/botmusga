package main

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log"
	"os"
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

	r, w := io.Pipe()
	// "buffer" for ffmpeg output, so when ffmpeg finishes we can still write remaining packets
	br, bw := io.Pipe()

	ytdlpCmd := exec.Command("yt-dlp", "-f", "bestaudio", url, "-o", "-")
	ytdlpCmd.Stdout = w

	cmd := exec.Command("ffmpeg", "-nostdin", "-threads", "1", "-i", "-", "-c:a", "libopus", "-ac", "2", "-ar", "48000", "-f", "ogg", "-b:a", "96K", "-vbr", "off", "pipe:1")
	cmd.Stdin = r

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		log.Println("Error getting audio stream from ffmpeg: ", err)
		return
	}

	err = cmd.Start()
	if err != nil {
		log.Println("Error starting ffmpeg: ", err)
                conn.Close(context.TODO())
		return
	}

	err = ytdlpCmd.Start()
	if err != nil {
		log.Println("Error starting yt-dlp: ", err)
                conn.Close(context.TODO())
                return
	}

	go func() {
		defer bw.Close()
		if _, err := io.Copy(bw, stdout); err != nil && !errors.Is(err, os.ErrClosed) {
			log.Println("Error copying ffmpeg output: ", err)
		}
	}()

	go func() {
		ytdlpCmd.Wait()
		log.Println("yt-dlp finished")
		// If we don't manually close the writer, ffmpeg never finishes
		w.Close()
	}()

	go func() {
		cmd.Wait()
		log.Println("ffmpeg finished")
	}()

	ticker := time.NewTicker(time.Millisecond * 20)
	defer ticker.Stop()

	decoder := NewDecoder(br)

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
