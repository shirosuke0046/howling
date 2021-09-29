package main

import (
	"io"
	"io/ioutil"
	"os"
	"os/exec"
	"strings"
	"sync"
	"time"

	"github.com/bwmarrin/discordgo"
	"github.com/jonas747/dca"
	"github.com/sirupsen/logrus"
	"golang.org/x/xerrors"
)

type Howling struct {
	session          *discordgo.Session
	voiceConn        *discordgo.VoiceConnection
	messageChannelID string

	mu      sync.Mutex
	voicech chan string
	done    chan struct{}

	dictionary string
	htsvoice   string
}

func New(token, dictionary, htsvoice string) (*Howling, error) {
	var howling Howling

	dg, err := discordgo.New("Bot " + token)
	if err != nil {
		return nil, xerrors.Errorf("howling: %v", err)
	}
	dg.AddHandler(howling.MessageCreate)
	dg.Identify.Intents = discordgo.IntentsGuilds | discordgo.IntentsGuildMessages | discordgo.IntentsGuildVoiceStates

	howling.session = dg
	howling.dictionary = dictionary
	howling.htsvoice = htsvoice
	howling.voicech = make(chan string)
	howling.done = make(chan struct{})

	return &howling, nil
}

func (howling *Howling) Open() error {
	howling.mu.Lock()
	defer howling.mu.Unlock()

	err := howling.session.Open()
	if err != nil {
		return xerrors.Errorf("howling: %v", err)
	}

	go func() {
		t := time.NewTicker(1 * time.Minute)
		defer t.Stop()

		for {
			select {
			case voice := <-howling.voicech:
				howling.Speak(voice)
			case <-howling.done:
				return
			}
		}
	}()

	return nil
}

func (howling *Howling) Close() error {
	howling.mu.Lock()
	defer howling.mu.Unlock()

	howling.Leave()
	close(howling.done)
	err := howling.session.Close()
	return xerrors.Errorf("howling: %v", err)
}

func (howling *Howling) MessageCreate(s *discordgo.Session, m *discordgo.MessageCreate) {
	if m.Author.ID == s.State.User.ID {
		return
	}

	switch {
	case strings.HasPrefix(m.Content, "hws!"):
		c, err := s.State.Channel(m.ChannelID)
		if err != nil {
			return
		}

		g, err := s.State.Guild(c.GuildID)
		if err != nil {
			return
		}

		for _, vs := range g.VoiceStates {
			if vs.UserID == m.Author.ID {
				howling.Join(g.ID, vs.ChannelID, m.ChannelID)
			}
		}
	case strings.HasPrefix(m.Content, "hwl!"):
		howling.Leave()
	case m.ChannelID == howling.messageChannelID:
		howling.voicech <- m.Content
	}
}

func (howling *Howling) Join(guildID, channelID, messageChannelID string) {
	howling.mu.Lock()
	defer howling.mu.Unlock()

	if howling.voiceConn != nil {
		return
	}

	vc, err := howling.session.ChannelVoiceJoin(guildID, channelID, false, false)
	if err != nil {
		logrus.Error(err)
		return
	}

	howling.voiceConn = vc
	howling.messageChannelID = messageChannelID
}

func (howling *Howling) leave() {
}

func (howling *Howling) Leave() {
	howling.mu.Lock()
	defer howling.mu.Unlock()

	if howling.voiceConn == nil {
		return
	}

	howling.voiceConn.Disconnect()
	howling.voiceConn = nil
	howling.messageChannelID = ""
}

func (howling *Howling) Speak(text string) {
	howling.mu.Lock()
	defer howling.mu.Unlock()

	if howling.voiceConn == nil {
		return
	}

	f, err := GenerateJtalkWav(text, howling.dictionary, howling.htsvoice)
	if err != nil {
		logrus.Error(err)
		return
	}
	defer os.Remove(f)

	encoder, err := dca.EncodeFile(f, dca.StdEncodeOptions)
	if err != nil {
		logrus.Error(err)
		return
	}
	defer encoder.Cleanup()

	errch := make(chan error)
	dca.NewStream(encoder, howling.voiceConn, errch)
	if err := <-errch; err != nil && err != io.EOF {
		logrus.Error(err)
		howling.Leave()
	}
}

func GenerateJtalkWav(text, dictionary, htsvoice string) (string, error) {
	tmpfile, _ := ioutil.TempFile("", "howling")
	fn := tmpfile.Name()
	tmpfile.Close()

	cmd := exec.Command(
		"open_jtalk",
		"-x", dictionary,
		"-m", htsvoice,
		"-r", "1.2",
		"-ow", fn,
	)

	stdin, _ := cmd.StdinPipe()
	io.WriteString(stdin, text)
	stdin.Close()

	err := cmd.Run()
	return fn, err
}
