package main

import (
	"strings"

	"github.com/gdamore/tcell/v2"
	"github.com/juju/loggo"
	"github.com/myxo/glink/pkg"
	"github.com/rivo/tview"
)

type (
	Tui struct {
		app        *tview.Application
		gservice   *glink.GlinkService
		model      *chatModel
		view       *chatView
		log_writer *TuiLogger
	}

	chatModel struct {
		own_info    glink.UserLightInfo
		Msgs        []string
		Chats       []string
		active_chat string
	}

	chatView struct {
		chat *tview.TextView
	}
)

func NewTui(gservice *glink.GlinkService, log_writer *TuiLogger) *Tui {
	app := tview.NewApplication()
	app.EnableMouse(false)

	chat_model := chatModel{own_info: gservice.OwnChatInfo}
	chats, err := gservice.Db.GetLastChats()
	if err != nil {
		log_writer.Warnf("Cannot read last chats: %s", err)
	}
	chat_model.Chats = chats

	if len(chat_model.Chats) != 0 {
		chat_model.active_chat = chat_model.Chats[0]
	}

	log_writer.Infof("Active chat: %s", chat_model.active_chat)

	chat := tview.NewTextView().
		SetTextAlign(tview.AlignLeft).
		SetDynamicColors(true)

	inputField := tview.NewInputField().
		SetLabel(" " + chat_model.own_info.Name + ": ").
		SetFieldBackgroundColor(tcell.ColorBlack)
	
	inputField.
		SetDoneFunc(func(key tcell.Key) {
			if key == tcell.KeyEscape {
				inputField.SetText("")
				return
			}
			if key != tcell.KeyEnter {
				return
			}
			text := inputField.GetText()
			if text == "" {
				return
			}
			inputField.SetText("")
			msg := glink.ChatMessage{Text: text, ToCid: chat_model.active_chat}
			gservice.UserMessage(msg)
		})

	grid := tview.NewGrid().
		SetColumns(30).
		SetBorders(true).
		AddItem(chat, 0, 0, 7, 3, 0, 0, false).
		AddItem(inputField, 7, 0, 1, 3, 0, 0, true)

	tui := Tui{app: app, gservice: gservice, model: &chat_model, view: &chatView{chat: chat}, log_writer: log_writer}

	err = tui.initMessages()
	if err != nil {
		log_writer.Warnf("Cannot init messages in tui: %s", err)
	}

	go func() {
		for {
			select {
			case ev := <-gservice.UxEvents:
				tui.processEvent(ev)
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

func (t *Tui) processEvent(ev interface{}) {
	switch ev := ev.(type) {
	case glink.ChatMessage:
		text := "[blue]" + ev.FromName + "[white]: " + ev.Text
		t.model.Msgs = append(t.model.Msgs, text)
		t.app.QueueUpdateDraw(func() {
			t.view.chat.SetText(strings.Join(t.model.Msgs, "\n"))
		})
	case glink.ChatUpdate:
		t.model.active_chat = ev.Cid
	default:
		t.log_writer.Error("Unknown event type")
	}
}

func (t *Tui) initMessages() error {
	msgs, err := t.gservice.GetMessages(t.model.active_chat)
	if err != nil {
		return err
	}
	for _, msg := range msgs {
		text := "[blue]" + msg.FromName + "[white]: " + msg.Text
		t.model.Msgs = append(t.model.Msgs, text)

	}
	t.view.chat.SetText(strings.Join(t.model.Msgs, "\n"))
	return nil
}

func getLogText(entry *loggo.Entry) string {
	var color string
	switch entry.Level {
	case loggo.ERROR:
		color = "[red][ERROR[][white]"
	case loggo.WARNING:
		color = "[red][WARN[][white]"
	case loggo.INFO:
		color = "[blue][INFO[][white]"
	case loggo.DEBUG:
		color = "[white][DEBUG[][white]"
	case loggo.TRACE:
		color = "[grey][TRACE[][grey]"
	}

	return "[white][LOG[]" + color + " " + entry.Message

}

func (tui *Tui) Run() {
	if err := tui.app.Run(); err != nil {
		panic(err)
	}
}
