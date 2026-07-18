package repository

import (
	"testing"

	"collector-backend/models"
)

func TestSocketConversationAndMessagePersistence(t *testing.T) {
	store, _ := openTempStore(t)
	defer store.db.Close()

	conversation, ok := store.CreateSocketConversation("chat-test", "测试访客", "hashed-token")
	if !ok || conversation.ID != "chat-test" || !conversation.Online || conversation.VisitorName != "测试访客" {
		t.Fatalf("unexpected conversation: %+v", conversation)
	}
	if !store.ValidateSocketConversationToken(conversation.ID, "hashed-token") || store.ValidateSocketConversationToken(conversation.ID, "wrong-token") {
		t.Fatal("socket visitor token validation is incorrect")
	}

	created, ok := store.CreateSocketMessage(models.SocketMessage{
		ConversationID: conversation.ID,
		SenderType:     "visitor",
		SenderName:     conversation.VisitorName,
		MessageType:    "text",
		Content:        "你好",
	})
	if !ok || created.ID == 0 {
		t.Fatalf("socket message was not created: %+v", created)
	}
	messages := store.ListSocketMessages(conversation.ID)
	if len(messages) != 1 || messages[0].Content != "你好" || messages[0].SenderType != "visitor" {
		t.Fatalf("unexpected socket messages: %+v", messages)
	}
	conversations := store.ListSocketConversations()
	if len(conversations) != 1 || conversations[0].LastMessage != "你好" || conversations[0].MessageCount != 1 {
		t.Fatalf("unexpected socket conversation summary: %+v", conversations)
	}
	if !store.SetSocketConversationOnline(conversation.ID, false) {
		t.Fatal("socket conversation presence was not updated")
	}
	updated, ok := store.FindSocketConversation(conversation.ID)
	if !ok || updated.Online {
		t.Fatalf("conversation should be offline: %+v", updated)
	}
}

func TestWorkspaceSocketMenuHierarchy(t *testing.T) {
	store, _ := openTempStore(t)
	defer store.db.Close()

	menus := store.ListMenus()
	byCode := map[string]models.Menu{}
	for _, menu := range menus {
		byCode[menu.Code] = menu
	}
	workspace := byCode["workspace"]
	dashboard := byCode["dashboard"]
	socketSupport := byCode["socket-support"]
	if workspace.ID == 0 || dashboard.ParentID == nil || *dashboard.ParentID != workspace.ID || dashboard.Name != "预览台" {
		t.Fatalf("unexpected dashboard hierarchy: workspace=%+v dashboard=%+v", workspace, dashboard)
	}
	if socketSupport.ID == 0 || socketSupport.ParentID == nil || *socketSupport.ParentID != workspace.ID || socketSupport.Path != "socket-support" {
		t.Fatalf("unexpected socket menu hierarchy: %+v", socketSupport)
	}
}
