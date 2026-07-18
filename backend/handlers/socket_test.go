package handlers

import (
	"testing"
)

func TestNewConversationRateLimit(t *testing.T) {
	handler := NewSocketHandler(nil, t.TempDir())
	for attempt := 1; attempt <= 3; attempt++ {
		if allowed, _ := handler.allowNewConversation("127.0.0.1"); !allowed {
			t.Fatalf("attempt %d should be allowed", attempt)
		}
	}
	if allowed, retryAfter := handler.allowNewConversation("127.0.0.1"); allowed || retryAfter <= 0 {
		t.Fatalf("fourth attempt should be rate limited, allowed=%v retryAfter=%v", allowed, retryAfter)
	}
}
