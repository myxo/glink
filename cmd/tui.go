package main

import (
	"strings"

	"github.com/gdamore/tcell/v2"
	"github.com/juju/loggo"
	"github.com/myxo/glink/pkg"
	"github.com/rivo/tview"
)

type Tui struct {
	app        *tview.Application
	model      chatModel
	log_writer *TuiLogger
}

type chatModel struct {
	Msgs []string
}

func NewTui(gservice *glink.GlinkService, log_writer *TuiLogger) *Tui {
	app := tview.NewApplication()

	chat := tview.NewTextView().
		SetTextAlign(tview.AlignLeft).
		SetDynamicColors(true)

	inputField := tview.NewInputField().
		SetLabel("Enter a number: ")
	//SetFieldWidth(10).
	//SetAcceptanceFunc(tview.InputFieldInteger).
	inputField.
		SetDoneFunc(func(key tcell.Key) {
			text := inputField.GetText()
			inputField.SetText("")
			msg := glink.ChatMessage{Payload: text}
			gservice.SendMessage(msg)
		})

	grid := tview.NewGrid().
		SetColumns(30, 30).
		SetBorders(true).
		AddItem(chat, 0, 0, 1, 3, 0, 0, false).
		AddItem(inputField, 2, 0, 1, 3, 0, 0, false)

	tui := Tui{app: app, log_writer: log_writer}

	go func() {
		for {
			select {
			case msg := <-gservice.NewMsg:
				log := msg.FromName + ": " + msg.Payload
				tui.model.Msgs = append(tui.model.Msgs, log)
				app.QueueUpdateDraw(func() {
					chat.SetText(strings.Join(tui.model.Msgs, "\n"))
				})
			case log_entry := <-log_writer.Messages:
				tui.model.Msgs = append(tui.model.Msgs, getLogText(&log_entry))
				app.QueueUpdateDraw(func() {
					chat.SetText(strings.Join(tui.model.Msgs, "\n"))
				})

			}
		}
	}()

	app.SetRoot(grid, true).SetFocus(inputField).EnableMouse(true)

	return &tui
}

func getLogText(entry *loggo.Entry) string {
	var color string
	switch entry.Level {
	case loggo.ERROR:
		color = "[red]ERROR[white]"
	case loggo.WARNING:
		color = "[red]WARN[white]"
	case loggo.INFO:
		color = "[blue]INFO[white]"
	case loggo.DEBUG:
		color = "[white]DEBUG[white]"
	}

	return "LOG: " + color + " " + entry.Message

}

func (tui *Tui) Run() {
	if err := tui.app.Run(); err != nil {
		panic(err)
	}
}
