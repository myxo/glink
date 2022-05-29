package main

import (
	"bufio"
	"fmt"
	"os"
	"sync"

	"github.com/myxo/glink/pkg"
)

var connMap = &sync.Map{}

func main() {

	gservice := glink.NewGlinkService()
	gservice.Launch()

	for {
		reader := bufio.NewReader(os.Stdin)
		fmt.Println(">> ")
		var msg glink.ChatMessage
		payload, err := reader.ReadString('\n')
		if err != nil {
			return
		}
		msg.Payload = payload
		gservice.SendMessage(msg)
	}
}
