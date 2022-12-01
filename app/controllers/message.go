package controllers

import (
	"context"
	"errors"
	"io"
	"log"
	"net/http"
	"net/url"
	"strings"

	"github.com/gabriel-vasile/mimetype"
	"github.com/gofiber/fiber/v2"
	"github.com/leadstolink/go-whatsapp-proxy/app/dto"
	"go.mau.fi/whatsmeow"
	waProto "go.mau.fi/whatsmeow/binary/proto"
	"go.mau.fi/whatsmeow/types"
	"google.golang.org/protobuf/proto"
)

type whatsappMessage struct {
	Receiver string `json:"receiver"`
	Message  string `json:"message"`
	Media    string `json:"media"`
}

func (k *Controller) SendMessage(c *fiber.Ctx) error {
	mess := whatsappMessage{}
	if err := c.BodyParser(&mess); err != nil {
		return err
	}

	jid, ok := parseJID(mess.Receiver)

	if !ok {
		return c.JSON(dto.Response{Status: false})
	}

	message, err := k.makeMessage(&mess)

	if err != nil {
		k.client.Log.Errorf("Message sending error: %s", err.Error())
		return c.JSON(dto.Response{Status: false})
	}

	_, err = k.client.SendMessage(jid, "", message)

	if err != nil {
		k.client.Log.Errorf("Message sending error: %s", err.Error())
		return c.JSON(dto.Response{Status: false})
	}

	return c.JSON(dto.Response{Status: true})
}

func (k *Controller) LastMessage(c *fiber.Ctx) error {
	l := len(messageList)

	if l == 0 {
		return c.SendStatus(404)
	}

	return c.JSON(messageList[l-1])
}

func (k *Controller) makeMessage(input *whatsappMessage) (*waProto.Message, error) {
	message := waProto.Message{}

	if len(input.Media) > 0 {
		resp, err := http.Get(input.Media)
		if err != nil {
			return nil, errors.New("Error getting media file by url")
		}
		defer resp.Body.Close()

		file, err := io.ReadAll(resp.Body)
		if err != nil {
			return nil, errors.New("Error reading file body")
		}

		mtype := mimetype.Detect(file)
		mimeType := mtype.String()
		mess := ""
		if len(input.Message) > 0 {
			mess = input.Message
		}
		switch mimeType {
		case "image/jpeg":
			fallthrough
		case "image/png":
			resp, err := k.client.Upload(context.Background(), file, whatsmeow.MediaImage)
			if err != nil {
				return nil, errors.New("Error uploading file")
			}

			message.ImageMessage = &waProto.ImageMessage{
				Caption:       proto.String(mess),
				Mimetype:      proto.String(mimeType),
				Url:           &resp.URL,
				DirectPath:    &resp.DirectPath,
				MediaKey:      resp.MediaKey,
				FileEncSha256: resp.FileEncSHA256,
				FileSha256:    resp.FileSHA256,
				FileLength:    &resp.FileLength,
			}
		case "audio/ogg":
			mimeType = "audio/ogg; codecs=opus"
			fallthrough
		case "audio/mp3":
			fallthrough
		case "audio/mp4":
			fallthrough
		case "audio/mpeg":
			fallthrough
		case "audio/amr":
			resp, err := k.client.Upload(context.Background(), file, whatsmeow.MediaAudio)
			if err != nil {
				return nil, errors.New("Error uploading file")
			}

			message.AudioMessage = &waProto.AudioMessage{
				//	Caption:       proto.String(""),
				Mimetype:      proto.String(mimeType),
				Url:           &resp.URL,
				DirectPath:    &resp.DirectPath,
				MediaKey:      resp.MediaKey,
				FileEncSha256: resp.FileEncSHA256,
				FileSha256:    resp.FileSHA256,
				FileLength:    &resp.FileLength,
			}
		case "video/mp4":
			resp, err := k.client.Upload(context.Background(), file, whatsmeow.MediaVideo)
			if err != nil {
				return nil, errors.New("Error uploading file")
			}

			message.VideoMessage = &waProto.VideoMessage{
				Caption:       proto.String(mess),
				Mimetype:      proto.String(mimeType),
				Url:           &resp.URL,
				DirectPath:    &resp.DirectPath,
				MediaKey:      resp.MediaKey,
				FileEncSha256: resp.FileEncSHA256,
				FileSha256:    resp.FileSHA256,
				FileLength:    &resp.FileLength,
			}
		default:
			resp, err := k.client.Upload(context.Background(), file, whatsmeow.MediaDocument)
			if err != nil {
				return nil, errors.New("Error uploading file")
			}

			u, _ := url.ParseRequestURI(input.Media)

			message.DocumentMessage = &waProto.DocumentMessage{
				//Caption:       proto.String(""),
				Title:         proto.String(u.Path),
				Mimetype:      proto.String(mimeType),
				Url:           &resp.URL,
				DirectPath:    &resp.DirectPath,
				MediaKey:      resp.MediaKey,
				FileEncSha256: resp.FileEncSHA256,
				FileSha256:    resp.FileSHA256,
				FileLength:    &resp.FileLength,
			}
		}
	} else {
		message.Conversation = proto.String(input.Message)
	}

	return &message, nil
}

func parseJID(rec string) (types.JID, bool) {
	if !strings.ContainsRune(rec, '@') {
		return types.NewJID(rec, types.DefaultUserServer), true
	} else {
		recipient, err := types.ParseJID(rec)
		if err != nil {
			log.Printf("Invalid JID %s: %v", rec, err)
			return recipient, false
		} else if recipient.User == "" {
			log.Printf("Invalid JID %s: no server specified", rec)
			return recipient, false
		}
		log.Printf("JID OK: %s", recipient.String())
		return recipient, true
	}
}
