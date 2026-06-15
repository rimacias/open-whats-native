package ui

import (
	"encoding/base64"
	"fmt"
	"image/color"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/widget"
	"golang.org/x/image/webp"

	"open-whats/internal/domain"
)

// MessageRenderer defines the strategy for rendering different message types
type MessageRenderer interface {
	Render(msg domain.Message, senderName string, avatarData []byte) fyne.CanvasObject
}

// TextMessageRenderer renders standard text messages
type TextMessageRenderer struct{}

func (r *TextMessageRenderer) Render(msg domain.Message, senderName string, avatarData []byte) fyne.CanvasObject {
	wrappedText := wrapText(msg.Text, 60)
	textLbl := widget.NewLabel(wrappedText)
	// We deliberately do NOT set Wrapping = fyne.TextWrapWord here.
	// By using explicit newlines, Fyne's HBox will respect the exact dimensions of our text block!
	
	bgColor := color.NRGBA{R: 38, G: 45, B: 49, A: 255} // Dark gray for others
	if msg.IsFromMe {
		bgColor = color.NRGBA{R: 5, G: 97, B: 98, A: 255} // WhatsApp dark green for me
	}
	
	bg := canvas.NewRectangle(bgColor)
	bg.CornerRadius = 8

	// Add padding to bubble content
	paddedText := container.NewPadded(textLbl)
	bubble := container.NewStack(bg, paddedText)

	return buildMessageRow(msg, senderName, avatarData, bubble)
}

// StickerMessageRenderer renders webp stickers
type StickerMessageRenderer struct{}

func (r *StickerMessageRenderer) Render(msg domain.Message, senderName string, avatarData []byte) fyne.CanvasObject {
	var content fyne.CanvasObject

	if msg.MediaURL == "" {
		content = widget.NewLabel("[Sticker Error]")
	} else {
		b64data := strings.TrimPrefix(msg.MediaURL, "data:image/webp;base64,")
		data, err := base64.StdEncoding.DecodeString(b64data)
		if err != nil {
			content = widget.NewLabel("[Sticker Error]")
		} else {
			imgReader := strings.NewReader(string(data))
			img, err := webp.Decode(imgReader)
			if err != nil {
				content = widget.NewLabel("[Animated Sticker (Unsupported by Decoder)]")
			} else {
				cImg := canvas.NewImageFromImage(img)
				cImg.FillMode = canvas.ImageFillContain
				cImg.SetMinSize(fyne.NewSize(150, 150))
				content = cImg
			}
		}
	}

	return buildMessageRow(msg, senderName, avatarData, content)
}

func buildMessageRow(msg domain.Message, senderName string, avatarData []byte, content fyne.CanvasObject) fyne.CanvasObject {
	var vboxObjects []fyne.CanvasObject
	var senderLbl *widget.Label

	var headerRow *fyne.Container

	if senderName != "" {
		senderLbl = widget.NewLabelWithStyle(senderName, fyne.TextAlignLeading, fyne.TextStyle{Bold: true})
		if msg.IsFromMe {
			senderLbl.Alignment = fyne.TextAlignTrailing
		}
		
		if len(avatarData) > 0 && !msg.IsFromMe {
			imgReader := strings.NewReader(string(avatarData))
			cImg := canvas.NewImageFromReader(imgReader, "avatar.png")
			cImg.FillMode = canvas.ImageFillContain
			cImg.SetMinSize(fyne.NewSize(30, 30))
			headerRow = container.NewHBox(cImg, senderLbl)
			vboxObjects = append(vboxObjects, headerRow)
		} else {
			vboxObjects = append(vboxObjects, senderLbl)
		}
	}
	
	vboxObjects = append(vboxObjects, content)

	if len(msg.Reactions) > 0 {
		counts := make(map[string]int)
		for _, r := range msg.Reactions {
			counts[r.Emoji]++
		}
		var emojis []fyne.CanvasObject
		for emoji, count := range counts {
			text := emoji
			if count > 1 {
				text = fmt.Sprintf("%s %d", emoji, count)
			}
			lbl := widget.NewLabel(text)
			emojis = append(emojis, lbl)
		}
		
		reactionBox := container.NewHBox(emojis...)
		if msg.IsFromMe {
			reactionBox = container.NewHBox(layout.NewSpacer(), reactionBox)
		}
		vboxObjects = append(vboxObjects, reactionBox)
	}
	
	vbox := container.NewVBox(vboxObjects...)
	
	if msg.IsFromMe {
		return container.NewHBox(layout.NewSpacer(), vbox)
	}
	
	return container.NewHBox(vbox, layout.NewSpacer())
}

// ImageMessageRenderer renders photo/image messages
type ImageMessageRenderer struct{}

func (r *ImageMessageRenderer) Render(msg domain.Message, senderName string, avatarData []byte) fyne.CanvasObject {
	var content fyne.CanvasObject

	if msg.MediaURL == "" {
		content = widget.NewLabel("[Image Error]")
	} else {
		// Extract base64 part
		parts := strings.Split(msg.MediaURL, ",")
		if len(parts) != 2 {
			content = widget.NewLabel("[Image Error: Invalid Data]")
		} else {
			data, err := base64.StdEncoding.DecodeString(parts[1])
			if err != nil {
				content = widget.NewLabel("[Image Error: Decode Failed]")
			} else {
				imgReader := strings.NewReader(string(data))
				cImg := canvas.NewImageFromReader(imgReader, "image")
				cImg.FillMode = canvas.ImageFillContain
				cImg.SetMinSize(fyne.NewSize(200, 200))
				content = cImg
			}
		}
	}

	return buildMessageRow(msg, senderName, avatarData, content)
}

// GetMessageRenderer acts as a factory returning the correct rendering strategy
func GetMessageRenderer(msg domain.Message) MessageRenderer {
	if msg.IsSticker {
		return &StickerMessageRenderer{}
	} else if msg.IsImage {
		return &ImageMessageRenderer{}
	}
	return &TextMessageRenderer{}
}

// wrapText manually inserts newlines so Fyne's layout engine can properly size the bubble
// without collapsing it to the width of the longest single word.
func wrapText(text string, lineLen int) string {
	var finalLines []string
	paragraphs := strings.Split(text, "\n")
	
	for _, p := range paragraphs {
		words := strings.Fields(p)
		if len(words) == 0 {
			finalLines = append(finalLines, "")
			continue
		}
		
		var currentLine []rune
		for _, word := range words {
			wordRunes := []rune(word)
			
			for len(wordRunes) > 0 {
				spaceLeft := lineLen - len(currentLine)
				if len(currentLine) > 0 {
					spaceLeft-- // account for the space character
				}
				
				if len(wordRunes) <= spaceLeft {
					if len(currentLine) > 0 {
						currentLine = append(currentLine, ' ')
					}
					currentLine = append(currentLine, wordRunes...)
					break
				}
				
				if len(currentLine) == 0 {
					// Force break the long word
					finalLines = append(finalLines, string(wordRunes[:lineLen]))
					wordRunes = wordRunes[lineLen:]
				} else {
					// Push current line and try again
					finalLines = append(finalLines, string(currentLine))
					currentLine = nil
				}
			}
		}
		if len(currentLine) > 0 {
			finalLines = append(finalLines, string(currentLine))
		}
	}
	return strings.Join(finalLines, "\n")
}
