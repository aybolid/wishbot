package main

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/aybolid/wishbot/internal/env"
	"github.com/aybolid/wishbot/internal/logger"
	"github.com/aybolid/wishbot/internal/tgbot"
)

func main() {
	env.Init()
	logger.Init()
	tgbot.Init()

	cancel := tgbot.ListenToUpdates()
	defer cancel()

	reader := bufio.NewReader(os.Stdin)
	for {
		cmd, err := reader.ReadString('\n')
		if err != nil {
			panic(err)
		}
		cmd = strings.TrimSpace(cmd)

		switch cmd {
		case "exit":
			os.Exit(0)
		default:
			fmt.Printf("%s: unknown command\n", cmd)
		}
	}
}
