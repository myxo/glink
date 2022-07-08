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
		Msgs        map[glink.Cid][]glink.ChatMessage
		Logs        []loggo.Entry
		Chats       []glink.ChatInfo
		active_chat glink.Cid
		uidToName   map[glink.Uid]string
	}

	chatView struct {
		chat     *tview.TextView
		logs     *tview.TextView
		chatList *tview.List
	}
)

func NewTui(gservice *glink.GlinkService, log_writer *TuiLogger) *Tui {
	app := tview.NewApplication()
	app.EnableMouse(false)

	chat_model := chatModel{
		own_info: gservice.OwnInfo,
		Msgs:     map[glink.Cid][]glink.ChatMessage{},
	}
	chats, err := gservice.Db.GetChats(true)
	if err != nil {
		log_writer.Warnf("Cannot read last chats: %s", err)
	}
	chat_model.Chats = chats

	if len(chat_model.Chats) != 0 {
		chat_model.active_chat = chat_model.Chats[0].Cid
	}
	uids, err := gservice.Db.GetUsersInfo()
	if err != nil {
		log_writer.Errorf("Cannot get users info: %s", err)
	}
	chat_model.uidToName = make(map[glink.Uid]string)
	for _, info := range uids {
		chat_model.uidToName[info.Uid] = info.Name
	}

	log_writer.Infof("Active chat: %s", chat_model.active_chat)

	chat := tview.NewTextView().
		SetTextAlign(tview.AlignLeft).
		SetDynamicColors(true)

	log_view := tview.NewTextView().
		SetTextAlign(tview.AlignLeft).
		SetDynamicColors(true)

	chat_list := tview.NewList()
	tui := Tui{
		app:        app,
		gservice:   gservice,
		model:      &chat_model,
		view:       &chatView{chat: chat, logs: log_view, chatList: chat_list},
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
			msg := glink.ChatMessage{Text: text, Cid: chat_model.active_chat}
			gservice.UserMessage(msg)
		})

	grid := tview.NewGrid().
		SetColumns(30).
		SetBorders(true).
		AddItem(log_view, 0, 0, 3, 3, 0, 0, false).
		AddItem(chat, 3, 0, 3, 3, 0, 0, false).
		AddItem(chat_list, 6, 0, 1, 3, 0, 0, false).
		AddItem(inputField, 7, 0, 1, 3, 0, 0, true)

	err = tui.initMessages()
	if err != nil {
		log_writer.Warnf("Cannot init messages in tui: %s", err)
	}
	tui.refreshChatList()
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
		t.model.Msgs[ev.Cid] = append(t.model.Msgs[ev.Cid], ev)
		t.app.QueueUpdateDraw(func() { t.refreshMessages() })

	case glink.ChatMessagePack:
		for _, msg := range ev.Messages {
			t.model.Msgs[msg.Cid] = append(t.model.Msgs[msg.Cid], msg)
		}
		t.app.QueueUpdateDraw(func() { t.refreshMessages() })

	case glink.ChatUpdate:
		t.model.active_chat = ev.Info.Cid
		for _, ci := range t.model.Chats {
			if ci.Cid == ev.Info.Cid {
				return
			}
		}
		t.model.Chats = append(t.model.Chats, *ev.Info)
		t.app.QueueUpdateDraw(func() { t.refreshChatList() })

	default:
		t.log_writer.Error("Unknown event type")
	}
}

func (t *Tui) initMessages() error {
	for _, chat := range t.model.Chats {
		msgs, err := t.gservice.GetMessages(chat.Cid)
		if err != nil {
			return err
		}
		t.model.Msgs[chat.Cid] = msgs

	}
	return nil
}

func (t *Tui) refreshChatList() {
	t.view.chatList.Clear()
	for i, chat := range t.model.Chats {
		iCopy := i
		name := chat.Name
		if name == "" && !chat.Group {
			// other guy name
		}
		t.view.chatList.AddItem(name, "", 'a'+rune(i), func() {
			new_active_chat := t.model.Chats[iCopy].Cid
			if new_active_chat != t.model.active_chat {
				t.model.active_chat = new_active_chat
				t.refreshMessages()
			}
		})
	}
}

func (t *Tui) refreshMessages() {
	msgs := make([]string, 0, 10)
	for _, msg := range t.model.Msgs[t.model.active_chat] {
		name := t.GetNameByUid(msg.Uid)
		text := "[blue]" + name + "[white]: " + msg.Text
		msgs = append(msgs, text)

	}
	t.view.chat.SetText(strings.Join(msgs, "\n"))

	logs := make([]string, 0, 10)
	for _, entry := range t.model.Logs {
		logs = append(logs, getLogText(&entry))
	}
	t.view.logs.SetText(strings.Join(logs, "\n"))
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

func (t *Tui) GetNameByUid(uid glink.Uid) string {
	name, ok := t.model.uidToName[uid]
	if ok {
		return name
	}
	name, err := t.gservice.GetNameByCid(uid)
	if err != nil {
		//t.log_writer.Errorf("Cannot get name by cid: %s", err)
		return "name?"
	}
	t.model.uidToName[uid] = name
	return name
}
