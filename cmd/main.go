package main

import (
	// "bufio"
	// "fmt"
	// "os"
	"github.com/juju/loggo"
	"github.com/myxo/glink/pkg"
	"sync"
)

var connMap = &sync.Map{}

func main() {
	tui_logger := NewTuiLogger()
	loggo.ReplaceDefaultWriter(tui_logger)
	logger := loggo.GetLogger("default")
	logger.SetLogLevel(loggo.DEBUG)

	gservice := glink.NewGlinkService(&logger)
	gservice.Launch()

	tui := NewTui(gservice, tui_logger)
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
