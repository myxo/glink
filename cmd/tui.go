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
		Msgs        map[string][]glink.ChatMessage
		Logs        []loggo.Entry
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

	chat_model := chatModel{
		own_info: gservice.OwnChatInfo,
		Msgs:     map[string][]glink.ChatMessage{},
	}
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

	tui := Tui{
		app:        app,
		gservice:   gservice,
		model:      &chat_model,
		view:       &chatView{chat: chat},
		log_writer: log_writer,
	}

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

	chat_list := tview.NewList()
	for i, chat := range chat_model.Chats {
		// TODO: propper handle of chat change
		iCopy := i
		chat_list.AddItem(chat, "", 'a'+rune(i), func() {
			new_active_chat := chat_model.Chats[iCopy]
			if new_active_chat != chat_model.active_chat {
				chat_model.active_chat = new_active_chat
				tui.refreshMessages()
			}
		})
	}

	grid := tview.NewGrid().
		SetColumns(30).
		SetBorders(true).
		AddItem(chat, 0, 0, 6, 3, 0, 0, false).
		AddItem(chat_list, 6, 0, 1, 3, 0, 0, false).
		AddItem(inputField, 7, 0, 1, 3, 0, 0, true)

	err = tui.initMessages()
	if err != nil {
		log_writer.Warnf("Cannot init messages in tui: %s", err)
	}
	tui.refreshMessages()

	go func() {
		for {
			select {
			case ev := <-gservice.UxEvents:
				tui.processEvent(ev)
			case log_entry := <-log_writer.Messages:
				tui.model.Logs = append(tui.model.Logs, log_entry)
				app.QueueUpdateDraw(func() { tui.refreshMessages() })
			}
		}
	}()

	app.SetRoot(grid, true).SetFocus(inputField).EnableMouse(true)

	return &tui
}

func (t *Tui) processEvent(ev interface{}) {
	switch ev := ev.(type) {
	case glink.ChatMessage:
		t.model.Msgs[ev.ToCid] = append(t.model.Msgs[ev.ToCid], ev)
		t.app.QueueUpdateDraw(func() { t.refreshMessages() })
	case glink.ChatUpdate:
		t.model.active_chat = ev.Cid
	default:
		t.log_writer.Error("Unknown event type")
	}
}

func (t *Tui) initMessages() error {
	for _, chat := range t.model.Chats {
		msgs, err := t.gservice.GetMessages(chat)
		if err != nil {
			return err
		}
		t.model.Msgs[chat] = msgs

	}
	return nil
}

func (t *Tui) refreshMessages() {
	msgs := make([]string, 0, 10)
	for _, msg := range t.model.Msgs[t.model.active_chat] {
		text := "[blue]" + msg.FromName + "[white]: " + msg.Text
		msgs = append(msgs, text)

	}
	t.view.chat.SetText(strings.Join(msgs, "\n"))
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
