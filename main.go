package main

import (
	"os"
	"os/signal"

	"github.com/jessevdk/go-flags"
	"github.com/sirupsen/logrus"
)

func main() {
	var opts Options

	_, err := flags.NewParser(&opts, flags.HelpFlag|flags.PassDoubleDash).Parse()
	if err != nil {
		logrus.Error(err)
		os.Exit(1)
	}

	bot, err := New(opts.DiscordToken, opts.JTalkDictionary, opts.JTalkHTSVoice)
	if err != nil {
		logrus.Error(err)
		os.Exit(1)
	}

	bot.Open()

	sig := make(chan os.Signal, 1)
	signal.Notify(sig, os.Interrupt)
	<-sig

	bot.Close()
}

type Options struct {
	DiscordToken    string `short:"t" description:"discord bot token" required:"true"`
	JTalkDictionary string `short:"x" description:"open-jtalk dictionary directory" required:"true"`
	JTalkHTSVoice   string `short:"m" description:"open-jtalk HTS voice files" required:"true"`
}
