package bot

import (
	"context"
	"fmt"
	"time"

	"go.mau.fi/whatsmeow"
	"go.mau.fi/whatsmeow/proto/waE2E"
	"go.mau.fi/whatsmeow/types"
	"google.golang.org/protobuf/proto"
)

// BuildQuotedReplyMessage menyusun pesan ExtendedTextMessage dengan metadata ContextInfo untuk Quoted Reply
func BuildQuotedReplyMessage(
	replyText string,
	msgID types.MessageID,
	sender types.JID,
	quotedMsg *waE2E.Message,
	isGroup bool,
) *waE2E.Message {
	if msgID == "" {
		return &waE2E.Message{
			Conversation: proto.String(replyText),
		}
	}

	ctxInfo := &waE2E.ContextInfo{
		StanzaID:      proto.String(string(msgID)),
		QuotedMessage: quotedMsg,
	}
	if isGroup {
		ctxInfo.Participant = proto.String(sender.ToNonAD().String())
	}

	return &waE2E.Message{
		ExtendedTextMessage: &waE2E.ExtendedTextMessage{
			Text:        proto.String(replyText),
			ContextInfo: ctxInfo,
		},
	}
}

// ReplyWithTyping mengirimkan reaksi emoji, simulasi status mengetik, dan pesan balasan (Quoted Reply) ke pengguna
func ReplyWithTyping(
	ctx context.Context,
	client *whatsmeow.Client,
	chat types.JID,
	sender types.JID,
	msgID types.MessageID,
	quotedMsg *waE2E.Message,
	isGroup bool,
	replyText string,
	emoji string,
	typingDuration time.Duration,
	actionName string,
) {
	if replyText == "" || client == nil {
		return
	}

	// 1. Berikan reaksi emoji pada pesan yang dikirim pengguna
	if emoji != "" && msgID != "" {
		reactionMsg := client.BuildReaction(chat, sender, msgID, emoji)
		_, _ = client.SendMessage(ctx, chat, reactionMsg)
	}

	// 2. Simulasi status "sedang mengetik..."
	if typingDuration <= 0 {
		typingDuration = 600 * time.Millisecond
	}
	_ = client.SendChatPresence(ctx, chat, types.ChatPresenceComposing, types.ChatPresenceMediaText)
	time.Sleep(typingDuration)
	_ = client.SendChatPresence(ctx, chat, types.ChatPresencePaused, types.ChatPresenceMediaText)

	// 3. Susun dan kirim pesan balasan dengan Quoted Reply
	msgToSend := BuildQuotedReplyMessage(replyText, msgID, sender, quotedMsg, isGroup)
	_, err := client.SendMessage(ctx, chat, msgToSend)
	if err != nil {
		fmt.Printf("Gagal mengirim balasan %s ke %s: %v\n", actionName, chat.User, err)
	} else {
		fmt.Printf("Sukses membalas %s ke %s (Quoted Reply)\n", actionName, chat.User)
	}
}
