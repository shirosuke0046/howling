package main

import (
	"fmt"
	"os"

	"github.com/jessevdk/go-flags"
)

func main() {
	var opts Options

	_, err := flags.NewParser(&opts, flags.HelpFlag|flags.PassDoubleDash).Parse()
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
	}
}

type Options struct {
	DiscordToken    string `short:"t" description:"discord bot token" required:"true"`
	JTalkDictionary string `short:"x" description:"open-jtalk dictionary directory" required:"true"`
	JTalkHTSVoice   string `short:"m" description:"open-jtalk HTS voice files" required:"true"`
}
