package main

import (
	// "bufio"
	// "os"
	"flag"

	"github.com/juju/loggo"
	"github.com/myxo/glink/pkg"
)

func main() {
	db_path := flag.String("db-path", "glink.db", "path to glink database")
	flag.Parse()

	tui_logger := NewTuiLogger()
	loggo.ReplaceDefaultWriter(tui_logger)
	logger := loggo.GetLogger("default")
	logger.SetLogLevel(loggo.DEBUG)

	gservice := glink.NewGlinkService(&logger, *db_path)
	gservice.Launch()

	tui := NewTui(gservice, tui_logger)
	tui.Run()
}
