# Open-Whats Native: Status & Development Roadmap

This document serves as the master plan for the Open-Whats project. It tracks our current feature completion status and outlines the technical roadmap for bringing the app to parity with the official WhatsApp Web client.

## 🟢 Current Feature Status (Completed)

The foundational architecture of the application has been stabilized using Go, Fyne (for the native UI), and `whatsmeow` (for the WhatsApp Multi-Device protocol).

### Authentication & Core Data
- [x] Multi-Device Pairing (Native QR Code UI popup)
- [x] Full Offline Database (SQLite integration for instant boot-ups)
- [x] Background History Sync (Syncs chats and messages from WhatsApp servers)
- [x] Real-time message streaming & database storage
- [x] Session persistence (auto-login on boot)
- [x] Logout functionality

### Messaging & UI 
- [x] Modern Dual-Panel Layout (Sidebar for chats, main area for messaging)
- [x] Chat filtering/searching (Title & Last Message)
- [x] Message searching within individual chats
- [x] Native Chat Bubbles (Dynamic height, color-coded per sender/receiver)
- [x] WhatsApp Push Name Resolution (Resolves names for unknown contacts in groups)
- [x] Automatic Group Name mapping & resolution
- [x] Basic Sticker Rendering (Static WebP parsing and rendering)
- [x] Asynchronous UI Indicators ("Syncing from WhatsApp..." progress bar)

---

## 🚀 Development Roadmap (Planned Features)

The following plan breaks down the features required to match official WhatsApp Web capabilities.

### Phase 1: Rich Media & Attachments
**1. Rendering Images & Photos** [x]
- *Description*: Handle standard photo messages (`*waE2E.ImageMessage`) sent in chats.
- *Implementation*: Extend `whatsmeow`'s `Download()` method. Use Fyne's `canvas.NewImageFromReader` to natively display them within the Strategy pattern `MessageRenderer`.
- *Storage*: Cache media bytes locally in a dedicated media directory to avoid re-downloading.

**2. Sending Attachments (Photos/Videos/Docs)** [x] (Photos)
- *Description*: Add an "Attach" button (+) next to the text input.
- *Implementation*: Open a Fyne `dialog.NewFileOpen`. Upload the file to WhatsApp servers using `client.Upload(context, data, type)` and wrap the resulting URL in a `*waE2E.Message`.

**3. Video Rendering** [ ]
- *Description*: Play received `.mp4` video messages.
- *Implementation*: Fyne does not natively play videos. We will need to render a video thumbnail and provide a "Click to open externally" button, or integrate a third-party Go binding for video playback.

### Phase 2: Profiles & Avatars
**1. Self Profile Management**
- *Description*: Allow the user to view and edit their own "About" text and Profile Picture.
- *Implementation*: Add a settings dialog. Use `client.SetStatusMessage()` and `client.SetProfilePicture()` to update the WhatsApp servers.

**2. Other Users' Profiles & Avatars**
- *Description*: Render profile pictures next to messages and in the left sidebar chat list.
- *Implementation*: 
  - Call `client.GetProfilePictureInfo(jid, &whatsmeow.GetProfilePictureParams{})` to retrieve avatar URLs.
  - Download and cache avatars locally mapped by JID.
  - Implement a `canvas.Image` with `canvas.NewCircle()` mask to render circular avatars next to contact names.

### Phase 3: Statuses (Stories)
**1. Viewing Statuses**
- *Description*: A new "Statuses" tab or button to view temporary stories from contacts.
- *Implementation*: 
  - Statuses arrive via the `status@broadcast` JID. We need to intercept messages sent to this JID in our `eventHandler`.
  - Group them by Sender JID, check their 24h expiration timestamp, and render them in an image carousel dialog.

**2. Posting Statuses**
- *Description*: Send a photo or text as a status update.
- *Implementation*: Send standard `waE2E.Message` payloads targeting the `status@broadcast` JID instead of a specific contact.

### Phase 4: Advanced Native Features
**1. Reply & Quoted Messages**
- *Description*: Allow users to reply to specific messages, showing a quoted preview above the new message.
- *Implementation*: Add a right-click context menu to message bubbles to trigger "Reply". Populate `ContextInfo` -> `StanzaId` and `Participant` in the outgoing `waE2E.Message`.

**2. Read Receipts & Ticks**
- *Description*: Display one tick (sent), two ticks (delivered), blue ticks (read).
- *Implementation*: Listen to `*events.Receipt` in the event handler and update the SQLite database. Add small icons to the bottom right of chat bubbles.

**3. Animated Stickers**
- *Description*: Fully render WhatsApp's `.webp` animated stickers.
- *Implementation*: Replace `golang.org/x/image/webp` with a CGO binding or a specialized WebP parser that supports animation frames. Iterate through the frames using a `time.Ticker` in Fyne.
