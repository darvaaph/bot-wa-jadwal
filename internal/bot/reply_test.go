package bot

import (
	"testing"

	"go.mau.fi/whatsmeow/proto/waE2E"
	"go.mau.fi/whatsmeow/types"
	"google.golang.org/protobuf/proto"
)

func TestBuildQuotedReplyMessage(t *testing.T) {
	senderJID := types.NewJID("628123456789", types.DefaultUserServer)
	quotedMsg := &waE2E.Message{
		Conversation: proto.String("!jadwal"),
	}

	// 1. Uji Quoted Reply di Grup
	groupReply := BuildQuotedReplyMessage("Berikut jadwal kuliah:", "MSG-1001", senderJID, quotedMsg, true)
	if groupReply.GetExtendedTextMessage() == nil {
		t.Fatalf("Expected ExtendedTextMessage for group reply, got nil")
	}
	if groupReply.GetExtendedTextMessage().GetText() != "Berikut jadwal kuliah:" {
		t.Errorf("Unexpected text: %s", groupReply.GetExtendedTextMessage().GetText())
	}
	ctxInfo := groupReply.GetExtendedTextMessage().GetContextInfo()
	if ctxInfo == nil {
		t.Fatalf("Expected ContextInfo, got nil")
	}
	if ctxInfo.GetStanzaID() != "MSG-1001" {
		t.Errorf("Expected StanzaID 'MSG-1001', got '%s'", ctxInfo.GetStanzaID())
	}
	if ctxInfo.GetParticipant() != senderJID.ToNonAD().String() {
		t.Errorf("Expected Participant '%s', got '%s'", senderJID.ToNonAD().String(), ctxInfo.GetParticipant())
	}
	if ctxInfo.GetQuotedMessage() == nil || ctxInfo.GetQuotedMessage().GetConversation() != "!jadwal" {
		t.Errorf("QuotedMessage was not preserved correctly")
	}

	// 2. Uji Quoted Reply di DM Pribadi
	dmReply := BuildQuotedReplyMessage("Info pribadi:", "MSG-1002", senderJID, quotedMsg, false)
	if dmReply.GetExtendedTextMessage() == nil {
		t.Fatalf("Expected ExtendedTextMessage for DM reply, got nil")
	}
	dmCtx := dmReply.GetExtendedTextMessage().GetContextInfo()
	if dmCtx == nil {
		t.Fatalf("Expected ContextInfo in DM, got nil")
	}
	if dmCtx.GetStanzaID() != "MSG-1002" {
		t.Errorf("Expected StanzaID 'MSG-1002', got '%s'", dmCtx.GetStanzaID())
	}
	if dmCtx.GetParticipant() != "" {
		t.Errorf("Expected empty Participant in DM, got '%s'", dmCtx.GetParticipant())
	}

	// 3. Uji Broadcast tanpa MsgID
	broadcastMsg := BuildQuotedReplyMessage("Pengingat pagi:", "", senderJID, nil, true)
	if broadcastMsg.GetConversation() != "Pengingat pagi:" {
		t.Errorf("Expected regular conversation message for broadcast, got %v", broadcastMsg)
	}
	if broadcastMsg.GetExtendedTextMessage() != nil {
		t.Errorf("ExtendedTextMessage should be nil when MsgID is empty")
	}
}
