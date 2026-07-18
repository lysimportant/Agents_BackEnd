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
	if err := store.MigrateAndSeed(); err != nil {
		t.Fatalf("rerun migration for socket title backfill: %v", err)
	}
	backfilled, ok := store.FindSocketConversation(conversation.ID)
	if !ok || backfilled.Title != "你好" {
		t.Fatalf("existing conversation title was not backfilled: %+v", backfilled)
	}
	if _, err := store.db.Exec(`UPDATE socket_conversations SET title='' WHERE id=?`, conversation.ID); err != nil {
		t.Fatalf("prepare empty title: %v", err)
	}
	if !store.SetSocketConversationOnline(conversation.ID, false) {
		t.Fatal("socket conversation presence was not updated")
	}
	updated, ok := store.FindSocketConversation(conversation.ID)
	if !ok || updated.Online {
		t.Fatalf("conversation should be offline: %+v", updated)
	}
	titled, ok := store.SetSocketConversationTitle(conversation.ID, "第一句咨询标题", true)
	if !ok || titled.Title != "第一句咨询标题" {
		t.Fatalf("conversation title was not initialized: %+v", titled)
	}
	unchanged, ok := store.SetSocketConversationTitle(conversation.ID, "不应覆盖", true)
	if !ok || unchanged.Title != titled.Title {
		t.Fatalf("first-message title should not be overwritten: %+v", unchanged)
	}
	renamed, ok := store.SetSocketConversationTitle(conversation.ID, "客户自定义标题", false)
	if !ok || renamed.Title != "客户自定义标题" {
		t.Fatalf("conversation title was not renamed: %+v", renamed)
	}
	closed, ok := store.CloseSocketConversation(conversation.ID)
	if !ok || closed.Status != "closed" || closed.Online || store.ValidateSocketConversationToken(conversation.ID, "hashed-token") {
		t.Fatalf("conversation should be closed and reject reconnects: %+v", closed)
	}
	if !store.SoftDeleteSocketConversation(conversation.ID) {
		t.Fatal("soft delete conversation failed")
	}
	if store.ValidateSocketConversationToken(conversation.ID, "hashed-token") {
		t.Fatal("deleted conversation token should be rejected")
	}
	if len(store.ListSocketConversations()) != 0 || len(store.ListSocketMessages(conversation.ID)) != 1 {
		t.Fatal("soft delete should hide conversation while preserving messages")
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
