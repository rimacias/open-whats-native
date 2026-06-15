package ui

import (
	"encoding/base64"
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
	Render(msg domain.Message, senderName string) fyne.CanvasObject
}

// TextMessageRenderer renders standard text messages
type TextMessageRenderer struct{}

func (r *TextMessageRenderer) Render(msg domain.Message, senderName string) fyne.CanvasObject {
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

	return buildMessageRow(msg.IsFromMe, senderName, bubble)
}

// StickerMessageRenderer renders webp stickers
type StickerMessageRenderer struct{}

func (r *StickerMessageRenderer) Render(msg domain.Message, senderName string) fyne.CanvasObject {
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

	return buildMessageRow(msg.IsFromMe, senderName, content)
}

func buildMessageRow(isFromMe bool, senderName string, content fyne.CanvasObject) fyne.CanvasObject {
	senderLbl := widget.NewLabelWithStyle(senderName, fyne.TextAlignLeading, fyne.TextStyle{Bold: true})
	
	vbox := container.NewVBox(senderLbl, content)
	
	if isFromMe {
		senderLbl.Alignment = fyne.TextAlignTrailing
		return container.NewHBox(layout.NewSpacer(), vbox)
	}
	
	return container.NewHBox(vbox, layout.NewSpacer())
}

// GetMessageRenderer acts as a factory returning the correct rendering strategy
func GetMessageRenderer(msg domain.Message) MessageRenderer {
	if msg.IsSticker {
		return &StickerMessageRenderer{}
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
		var currentLine string
		for _, word := range words {
			if len([]rune(currentLine))+len([]rune(word))+1 > lineLen {
				if currentLine != "" {
					finalLines = append(finalLines, currentLine)
					currentLine = word
				} else {
					// Word itself is longer than lineLen
					finalLines = append(finalLines, word)
					currentLine = ""
				}
			} else {
				if currentLine == "" {
					currentLine = word
				} else {
					currentLine += " " + word
				}
			}
		}
		if currentLine != "" {
			finalLines = append(finalLines, currentLine)
		}
	}
	return strings.Join(finalLines, "\n")
}
