package main

import (
	"strings"

	"github.com/gdamore/tcell/v2"
	"github.com/myxo/glink/pkg"
	"github.com/rivo/tview"
)

type Tui struct {
	app   *tview.Application
	model chatModel
}

type chatModel struct {
	Msgs []string
}

func NewTui(gservice *glink.GlinkService) *Tui {
	app := tview.NewApplication()

	chat := tview.NewTextView().
		SetTextAlign(tview.AlignCenter).
		SetText("Chat")
	inputField := tview.NewInputField().
		SetLabel("Enter a number: ")
	//SetFieldWidth(10).
	//SetAcceptanceFunc(tview.InputFieldInteger).
	inputField.
		SetDoneFunc(func(key tcell.Key) {
			text := inputField.GetText()
			msg := glink.ChatMessage{Payload: text}
			gservice.SendMessage(msg)
		})

	grid := tview.NewGrid().
		SetColumns(30, 30).
		SetBorders(true).
		AddItem(chat, 0, 0, 1, 3, 0, 0, false).
		AddItem(inputField, 2, 0, 1, 3, 0, 0, false)

	tui := Tui{app: app}

	go func() {
		for {
			select {
			case msg := <-gservice.NewMsg:
				tui.model.Msgs = append(tui.model.Msgs, msg.Payload)
				app.QueueUpdateDraw(func() {
					chat.SetText(strings.Join(tui.model.Msgs, "\n"))
				})
			}
		}
	}()

	app.SetRoot(grid, true).EnableMouse(true)

	return &tui
}

func (tui *Tui) Run() {
	if err := tui.app.Run(); err != nil {
		panic(err)
	}
}
