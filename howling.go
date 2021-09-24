package main

import (
	"io"
	"io/ioutil"
	"os"
	"os/exec"
	"strings"

	"github.com/bwmarrin/discordgo"
	"github.com/jonas747/dca"
	"github.com/sirupsen/logrus"
	"golang.org/x/xerrors"
)

const (
	MessagePrefix = "howl!"
)

type Howling struct {
	session   *discordgo.Session
	voiceConn *discordgo.VoiceConnection

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

	return &howling, nil
}

func (howling *Howling) Open() error {
	err := howling.session.Open()
	if err != nil {
		return xerrors.Errorf("howling: %v", err)
	}

	return nil
}

func (howling *Howling) Close() error {
	err := howling.session.Close()
	return xerrors.Errorf("howling: %v", err)
}

func (howling *Howling) MessageCreate(s *discordgo.Session, m *discordgo.MessageCreate) {
	if m.Author.ID == s.State.User.ID {
		return
	}

	if strings.HasPrefix(m.Content, MessagePrefix) {
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
				howling.Join(g.ID, vs.ChannelID)
			}
		}
	} else {
		howling.Speak(m.Content)
	}
}

func (howling *Howling) Join(guildID, channelID string) {
	if howling.voiceConn != nil {
		return
	}

	vc, err := howling.session.ChannelVoiceJoin(guildID, channelID, false, true)
	if err != nil {
		logrus.Error(err)
		return
	}

	howling.voiceConn = vc
}

func (howling *Howling) Leave() {
	howling.voiceConn.Close()
	howling.voiceConn = nil
}

func (howling *Howling) Speak(text string) {
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
		"-ow", fn,
	)

	stdin, _ := cmd.StdinPipe()
	io.WriteString(stdin, text)
	stdin.Close()

	err := cmd.Run()
	return fn, err
}
