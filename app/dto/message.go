package dto

type IncomingMessage struct {
	ID           string `json:"id"`
	Chat         string `json:"chat"`
	Caption      string `json:"caption"`
	Sender       string `json:"sender"`
	SenderName   string `json:"senderName"`
	IsFromMe     bool   `json:"isFromMe"`
	IsGroup      bool   `json:"isGroup"`
	IsEphemeral  bool   `json:"isEphemeral"`
	IsViewOnce   bool   `json:"isViewOnce"`
	Timestamp    string `json:"timestamp"`
	MediaType    string `json:"mediaType"`
	Multicast    bool   `json:"multicast"`
	Conversation string `json:"conversation"`
}

type MessageAttachment struct {
	File     []byte
	Filename string
}

func (ma *MessageAttachment) IsEmpty() bool {
	return len(ma.File) == 0
}
