package openai

import "testing"

// TestIsChatModel checks the chat-model filter keeps chat families and drops non-chat variants.
func TestIsChatModel(t *testing.T) {
	chat := []string{"gpt-4o", "gpt-4o-mini", "gpt-4.1", "gpt-5", "chatgpt-4o-latest", "o1", "o3-mini", "o4-mini"}
	notChat := []string{
		"text-embedding-3-small", "gpt-4o-audio-preview", "gpt-4o-realtime-preview",
		"gpt-image-1", "tts-1", "whisper-1", "dall-e-3", "gpt-3.5-turbo-instruct",
		"omni-moderation-latest", "davinci-002", "babbage-002",
	}
	for _, m := range chat {
		if !isChatModel(m) {
			t.Errorf("%q should be a chat model", m)
		}
	}
	for _, m := range notChat {
		if isChatModel(m) {
			t.Errorf("%q should NOT be a chat model", m)
		}
	}
}
