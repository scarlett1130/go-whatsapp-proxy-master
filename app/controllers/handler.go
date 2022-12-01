package controllers

import (
	"bytes"
	"fmt"
	"hash/fnv"
	"io"
	"io/ioutil"
	"mime/multipart"
	"net/http"
	"os"
	"reflect"
	"strconv"
	"strings"
	"time"

	"github.com/leadstolink/go-whatsapp-proxy/app/dto"
	waProto "go.mau.fi/whatsmeow/binary/proto"
	"go.mau.fi/whatsmeow/types/events"
)

var messageList []events.Message

func (k *Controller) eventHandler(evt interface{}) {
	switch v := evt.(type) {
	case *events.Message:
		messageList = append(messageList, *v)

		//todo remove this debug
		//fmt.Printf("eventHandler_message case *events.Message - %+v\n", v)
		//fmt.Printf("caption : %+v\n", *v.Message.ImageMessage.Caption)
		caption := ""
		//telf := ""
		if v.Message.ImageMessage != nil {

			if v.Message.ImageMessage.Caption != nil {
				caption = *v.Message.ImageMessage.Caption
			}

		}

		if v.Message.VideoMessage != nil {

			if v.Message.VideoMessage.Caption != nil {
				caption = *v.Message.VideoMessage.Caption
			}

		}
		mess := dto.IncomingMessage{
			ID:           v.Info.ID,
			Chat:         v.Info.Chat.String(),
			Caption:      caption,
			Sender:       v.Info.Sender.String(),
			SenderName:   v.Info.PushName,
			IsFromMe:     v.Info.IsFromMe,
			IsGroup:      v.Info.IsGroup,
			IsEphemeral:  v.IsEphemeral,
			IsViewOnce:   v.IsViewOnce,
			Timestamp:    v.Info.Timestamp.String(),
			MediaType:    v.Info.MediaType,
			Multicast:    v.Info.Multicast,
			Conversation: v.Message.GetConversation(),
		}

		if mess.Conversation == "" {
			if v.Message.ExtendedTextMessage != nil {
				mess.Conversation = v.Message.ExtendedTextMessage.GetText()
			}
		}

		var attachment dto.MessageAttachment
		if mess.MediaType != "" {
			attachment.File, _ = k.client.DownloadAny(v.Message)
			attachment.Filename = getFilename(v.Info.MediaType, v.Message)
		}
		if v.Message.ExtendedTextMessage != nil && v.Message.ExtendedTextMessage.ContextInfo != nil {
			if v.Message.ExtendedTextMessage.ContextInfo.QuotedMessage.Conversation != nil {
				s := *v.Message.ExtendedTextMessage.ContextInfo.QuotedMessage.Conversation
				p := *v.Message.ExtendedTextMessage.ContextInfo.Participant
				//	mess.Caption = "<em> &quot; " + s + " &quot; >\n &emsp; " + mess.Conversation
				mess.Conversation = "{+" + p + "}\n  [" + s + "]\n  \n" + mess.Conversation
			}
			if v.Message.ExtendedTextMessage.ContextInfo.QuotedMessage.ImageMessage != nil {
				attachment.File, _ = k.client.DownloadAny(v.Message.ExtendedTextMessage.ContextInfo.QuotedMessage)
				mediaType := strings.Split(*v.Message.ExtendedTextMessage.ContextInfo.QuotedMessage.ImageMessage.Mimetype, "/")
				attachment.Filename = getFilename(mediaType[0], v.Message.ExtendedTextMessage.ContextInfo.QuotedMessage)
			}
			if v.Message.ExtendedTextMessage.ContextInfo.QuotedMessage.VideoMessage != nil {
				attachment.File, _ = k.client.DownloadAny(v.Message.ExtendedTextMessage.ContextInfo.QuotedMessage)
				mediaType := strings.Split(*v.Message.ExtendedTextMessage.ContextInfo.QuotedMessage.VideoMessage.Mimetype, "/")
				attachment.Filename = getFilename(mediaType[0], v.Message.ExtendedTextMessage.ContextInfo.QuotedMessage)
			}
		}
		if v.Message.ContactMessage != nil {
			s := *v.Message.ContactMessage.Vcard
			email := ""
			telf := ""
			indx := strings.Index(s, "TEL")
			if indx >= 0 {
				telf = s[indx:]
				indx = strings.Index(telf, ":")
				indx2 := strings.Index(telf, "\n")
				telf = "Telf: " + telf[indx+1:indx2]
			}
			indx = strings.Index(s, "EMAIL")
			if indx >= 0 {
				email = s[indx:]
				indx = strings.Index(email, ":")
				indx2 := strings.Index(email, "\n")
				email = "Email: " + email[indx+1:indx2]
			}

			nobre := *v.Message.ContactMessage.DisplayName
			mess.Conversation = "Nombre: " + nobre + "\n" + email + "\n" + telf
			mess.Caption = nobre
		}
		if v.Message.LocationMessage != nil {

			mapsUrl := "https://maps.google.com"
			latitude := *v.Message.LocationMessage.DegreesLatitude
			longitude := *v.Message.LocationMessage.DegreesLongitude

			locationUrl := fmt.Sprintf("%s/?q=%f,%f", mapsUrl, latitude, longitude)

			mess.Conversation = locationUrl
			mess.Caption = locationUrl

			//
			//		attachment.File = v.Message.LocationMessage.JpegThumbnail
			//	attachment.Filename = getFilename("location", v.Message)

		}

		//k.proxyToChatApp(mess, attachment)
		if mess.Chat != "status@broadcast" {
			k.proxyToChatApp(mess, attachment)
		}
		//todo remove this debug
		//	fmt.Printf("eventHandler_message modified- %+v\n", mess)

		//fmt.Println(k.proxyToChatApp(mess, attachment))
		//fmt.Println("Received a message!", v.Message.GetConversation())
	}
}

func (k *Controller) proxyToChatApp(message dto.IncomingMessage, attachment ...dto.MessageAttachment) string {
	client := &http.Client{
		Timeout: time.Second * 10,
	}

	// New multipart writer.
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)

	v := reflect.ValueOf(message)
	typeOfS := v.Type()

	for i := 0; i < v.NumField(); i++ {
		var fw io.Writer
		var err error
		if fw, err = writer.CreateFormField(typeOfS.Field(i).Name); err != nil {
			continue
		}
		if _, err = io.Copy(fw, strings.NewReader(fmt.Sprintf("%v", v.Field(i).Interface()))); err != nil {
			continue
		}
	}

	if !attachment[0].IsEmpty() {
		fw, err := writer.CreateFormFile("attachment", attachment[0].Filename)
		if err != nil {
			k.client.Log.Errorf("POST2PROXY make attachment err: %s", err)
		}

		_, err = io.Copy(fw, bytes.NewReader(attachment[0].File))
		if err != nil {
			k.client.Log.Errorf("POST2PROXY make attachment err: %s", err)
		}
	}

	writer.Close()
	req, err := http.NewRequest("POST", os.Getenv("PROXY_URL"), bytes.NewReader(body.Bytes()))
	if err != nil {
		k.client.Log.Errorf("POST2PROXY err: %s", err)
	}

	req.Header.Set("Content-Type", writer.FormDataContentType())
	resp, _ := client.Do(req)
	if resp.StatusCode != http.StatusOK {
		k.client.Log.Errorf("POST2PROXY request failed with response code: %d", resp.StatusCode)
	}

	content, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		k.client.Log.Errorf("POST2PROXY reading body err: %s", err)
	}

	return string(content)
}

func getFilename(mediaType string, message *waProto.Message) string {
	switch mediaType {
	case "sticker":
		return hash(message.StickerMessage.String()) + ".webp"
	case "image":
		return hash(message.ImageMessage.String()) + message.ImageMessage.GetMimetype()[6:]
	case "video":
		return hash(message.VideoMessage.String()) + ".mp4"
	case "document":
		return message.DocumentMessage.GetFileName()
	case "vcard":
		return message.ContactMessage.GetDisplayName() + ""
	case "ptt":
		return hash(message.AudioMessage.String()) + ".ogg"
	}
	return ""
}

func hash(s string) string {
	h := fnv.New32a()
	h.Write([]byte(s))
	return strconv.FormatUint(uint64(h.Sum32()), 10)
}
