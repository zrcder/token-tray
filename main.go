package main

import (
	"flag"
	"os"

	"fyne.io/systray"
)

var (
	testMode bool
	genIcons bool
)

func main() {
	flag.BoolVar(&testMode, "test", false, "run with mock data (no real API calls)")
	flag.BoolVar(&genIcons, "gen-icons", false, "generate test icon PNGs and exit")
	flag.Parse()

	if genIcons {
		generateTestIcons()
		return
	}

	systray.Run(onReady, onExit)
}

func exit(msg string) {
	println(msg)
	os.Exit(1)
}
