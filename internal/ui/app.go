package ui

import (
	"fmt"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
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
			return container.NewVBox(
				widget.NewLabelWithStyle("Contact Name Placeholder", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
				widget.NewLabel("Last message preview"),
			)
		},
		func(i widget.ListItemID, o fyne.CanvasObject) {
			c := ui.filteredCh[i]
			box := o.(*fyne.Container)
			box.Objects[0].(*widget.Label).SetText(c.Name)
			box.Objects[1].(*widget.Label).SetText(c.LastMsg)
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

	inputArea := container.NewBorder(nil, nil, nil, ui.sendBtn, ui.msgEntry)

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
	if isSyncing {
		ui.syncContainer.Show()
	} else {
		ui.syncContainer.Hide()
		// Refresh chats after a sync payload
		go ui.fetchChats()
	}
}
