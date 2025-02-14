package main

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/aybolid/wishbot/internal/db"
	"github.com/aybolid/wishbot/internal/env"
	"github.com/aybolid/wishbot/internal/logger"
	"github.com/aybolid/wishbot/internal/tgbot"
)

func init() {
	env.Init()
	logger.Init()

	db.Init()
	tgbot.Init()
}

func main() {
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
		case "state":
			fmt.Println("Pending group creation")
			for userID, pending := range tgbot.STATE.PendingGroupCreation {
				fmt.Printf("\t%d: %t\n", userID, pending)
			}
			fmt.Println("Pending invite creation")
			for userID, pending := range tgbot.STATE.PendingInviteCreation {
				fmt.Printf("\t%d: %d\n", userID, pending)
			}
		default:
			fmt.Printf("%s: unknown command\n", cmd)
		}
	}
}
