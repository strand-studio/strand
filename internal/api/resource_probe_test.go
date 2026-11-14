package api

import "testing"

func TestLooksLikeOpenAIModelsList_DataFirstTruncated(t *testing.T) {
	body := `{"data":[{"id":"gpt-6-luna","object":"model"},{"id":"omni-moderation`
	if !looksLikeOpenAIModelsList(body) {
		t.Fatal("expected truncated data-first models payload to pass")
	}
}
