package main

import (
	// "bufio"
	// "fmt"
	// "os"
	"github.com/myxo/glink/pkg"
	"sync"
)

var connMap = &sync.Map{}

func main() {
	gservice := glink.NewGlinkService()
	gservice.Launch()

	tui := NewTui(gservice)
	tui.Run()

	// for {
	// 	reader := bufio.NewReader(os.Stdin)
	// 	fmt.Println(">> ")
	// 	var msg glink.ChatMessage
	// 	payload, err := reader.ReadString('\n')
	// 	if err != nil {
	// 		return
	// 	}
	// 	msg.Payload = payload
	// 	gservice.SendMessage(msg)
	// }
}
