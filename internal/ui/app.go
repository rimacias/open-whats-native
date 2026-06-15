package ui

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/widget"

	"open-whats/internal/domain"
)

// AppUI coordinates the entire Fyne application state and layout
type AppUI struct {
	client     domain.WhatsAppClient
	msgStore   domain.MessageStore
	fyneApp    fyne.App
	mainWindow fyne.Window

	activeJID   string
	chatList    *widget.List
	allChats    []domain.ChatPreview
	filteredCh  []domain.ChatPreview
	
	// Caching contacts for name resolution
	contactMap  map[string]string
	avatarMap   map[string][]byte
	allContacts []domain.Contact

	msgScroll   *container.Scroll
	msgVBox     *fyne.Container
	allMsgs     []domain.Message
	filteredMsg []domain.Message

	msgEntry  *widget.Entry
	sendBtn   *widget.Button
	chatTitle *widget.Label

	searchChat *widget.Entry
	searchMsg  *widget.Entry

	qrDialog      dialog.Dialog
	syncIndicator *widget.ProgressBarInfinite
	syncLabel     *widget.Label
	syncContainer *fyne.Container
}

// NewAppUI initializes the primary Fyne application components and registers listeners
func NewAppUI(client domain.WhatsAppClient, msgStore domain.MessageStore) *AppUI {
	fyneApp := app.New()
	mainWindow := fyneApp.NewWindow("Open Whats - Native")
	mainWindow.Resize(fyne.NewSize(1100, 750))

	ui := &AppUI{
		client:     client,
		msgStore:   msgStore,
		fyneApp:    fyneApp,
		mainWindow: mainWindow,
		contactMap: make(map[string]string),
		avatarMap:  make(map[string][]byte),
	}

	ui.syncIndicator = widget.NewProgressBarInfinite()
	ui.syncLabel = widget.NewLabel("Syncing from WhatsApp...")
	ui.syncLabel.Alignment = fyne.TextAlignCenter
	ui.syncContainer = container.NewVBox(ui.syncLabel, ui.syncIndicator)
	ui.syncContainer.Hide()

	ui.client.RegisterMessageCallback(ui.onMessageReceived)
	ui.client.RegisterLoginCallbacks(ui.showQRCode, ui.onLoginSuccess)
	ui.client.RegisterSyncCallback(ui.onSync)
	return ui
}

// Start constructs the UI layouts and begins the Fyne event loop
func (ui *AppUI) Start() {
	// 1. Build Left Side (Chats List & Search)
	ui.searchChat = widget.NewEntry()
	ui.searchChat.SetPlaceHolder("Search chats...")
	ui.searchChat.OnChanged = func(s string) {
		ui.filterChats(s)
	}

	ui.chatList = widget.NewList(
		func() int { return len(ui.filteredCh) },
		func() fyne.CanvasObject {
			// Placeholder image
			img := canvas.NewImageFromResource(fyne.NewStaticResource("default", []byte{}))
			img.SetMinSize(fyne.NewSize(40, 40))
			img.FillMode = canvas.ImageFillContain
			
			vbox := container.NewVBox(
				widget.NewLabelWithStyle("Contact Name Placeholder", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
				widget.NewLabel("Last message preview"),
			)
			return container.NewHBox(img, vbox)
		},
		func(i widget.ListItemID, o fyne.CanvasObject) {
			c := ui.filteredCh[i]
			hbox := o.(*fyne.Container)
			img := hbox.Objects[0].(*canvas.Image)
			vbox := hbox.Objects[1].(*fyne.Container)
			
			vbox.Objects[0].(*widget.Label).SetText(c.Name)
			vbox.Objects[1].(*widget.Label).SetText(c.LastMsg)

			lookupJID := c.JID
			if !strings.Contains(lookupJID, "@") {
				lookupJID += "@s.whatsapp.net"
			}

			if av, ok := ui.avatarMap[lookupJID]; ok {
				if len(av) > 0 {
					img.Resource = fyne.NewStaticResource("avatar", av)
				} else {
					img.Resource = fyne.NewStaticResource("default", []byte{})
				}
				img.Refresh()
			} else {
				img.Resource = fyne.NewStaticResource("default", []byte{})
				img.Refresh()
				ui.avatarMap[lookupJID] = nil // Avoid repeated requests
				go func(jid string) {
					url, err := ui.client.GetProfilePicture(context.Background(), jid)
					if err == nil && url != "" {
						resp, err := http.Get(url)
						if err == nil {
							data, err := io.ReadAll(resp.Body)
							resp.Body.Close()
							if err == nil && len(data) > 0 {
								ui.avatarMap[jid] = applyCircularMask(data)
								fyne.Do(func() {
									ui.chatList.RefreshItem(i)
								})
							}
						}
					}
				}(lookupJID)
			}
		},
	)

	ui.chatList.OnSelected = func(id widget.ListItemID) {
		c := ui.filteredCh[id]
		ui.activeJID = c.JID
		ui.chatTitle.SetText(fmt.Sprintf("Chat: %s", c.Name))
		ui.loadChatHistory(c.JID)
	}

	newChatBtn := widget.NewButton("New Chat", func() {
		ui.showNewChatDialog()
	})
	logoutBtn := widget.NewButton("Logout", func() {
		ui.logout()
	})
	topBtns := container.NewHBox(newChatBtn, logoutBtn)

	leftPanel := container.NewBorder(
		container.NewVBox(
			container.NewBorder(nil, nil, nil, topBtns, widget.NewLabelWithStyle("Chats", fyne.TextAlignCenter, fyne.TextStyle{Bold: true})),
			ui.searchChat,
			ui.syncContainer,
		),
		nil, nil, nil,
		ui.chatList,
	)

	// 2. Build Right Side (Chat Area & Search)
	ui.chatTitle = widget.NewLabelWithStyle("Select a chat to start messaging", fyne.TextAlignCenter, fyne.TextStyle{Bold: true})
	
	ui.searchMsg = widget.NewEntry()
	ui.searchMsg.SetPlaceHolder("Search in chat...")
	ui.searchMsg.OnChanged = func(s string) {
		ui.filterMessages(s)
	}

	headerContainer := container.NewVBox(ui.chatTitle, ui.searchMsg)

	ui.msgVBox = container.NewVBox()
	ui.msgScroll = container.NewVScroll(ui.msgVBox)

	ui.msgEntry = widget.NewEntry()
	ui.msgEntry.SetPlaceHolder("Type a message...")
	ui.msgEntry.OnSubmitted = func(s string) {
		ui.sendMessage()
	}

	ui.sendBtn = widget.NewButton("Send", func() {
		ui.sendMessage()
	})

	attachBtn := widget.NewButton("+", func() {
		ui.showAttachDialog()
	})

	inputArea := container.NewBorder(nil, nil, attachBtn, ui.sendBtn, ui.msgEntry)

	rightPanel := container.NewBorder(
		headerContainer,
		inputArea,
		nil, nil,
		ui.msgScroll,
	)

	// 3. Main Split
	split := container.NewHSplit(leftPanel, rightPanel)
	split.Offset = 0.3 // 30% left, 70% right

	ui.mainWindow.SetContent(split)

	// Load initial data securely (aborts if not logged in)
	go ui.fetchChats()

	ui.mainWindow.ShowAndRun()
}

func (ui *AppUI) onLoginSuccess() {
	if ui.qrDialog != nil {
		ui.qrDialog.Hide()
	}
	// Fetch chats and contacts now that we are successfully logged in
	go ui.fetchChats()
}

func (ui *AppUI) onSync(isSyncing bool) {
	fyne.Do(func() {
		if isSyncing {
			ui.syncContainer.Show()
		} else {
			ui.syncContainer.Hide()
			// Refresh chats after a sync payload
			go ui.fetchChats()
		}
	})
}
