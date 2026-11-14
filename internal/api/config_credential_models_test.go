package api

import "testing"

func TestLooksLikeOpenAIModelsList(t *testing.T) {
	cases := []struct {
		name string
		body string
		want bool
	}{
		{
			name: "complete list",
			body: `{"object":"list","data":[{"id":"gpt-6-luna","object":"model"}]}`,
			want: true,
		},
		{
			name: "complete list with spaces",
			body: `{ "object": "list", "data": [ { "id": "gpt-6-luna" } ] }`,
			want: true,
		},
		{
			name: "truncated openai models list",
			body: `{ "object": "list", "data": [ { "id": "omni-moderation-2024-09-26", "object": "model", "created": 1732734466, "owned_by": "system", "shutdown`,
			want: true,
		},
		{
			name: "truncated data-first list",
			body: `{"data":[{"id":"gpt-6-luna","object":"model"},{"id":"gpt-6-sol","object":"model"`,
			want: true,
		},
		{
			name: "html captive portal",
			body: `<!DOCTYPE html><html><body>ok</body></html>`,
			want: false,
		},
		{
			name: "error object",
			body: `{"error":{"message":"invalid_api_key","type":"invalid_request_error"}}`,
			want: false,
		},
		{
			name: "data array without object",
			body: `{"data":[{"id":"gpt-6-luna"}]}`,
			want: true,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := looksLikeOpenAIModelsList(tc.body); got != tc.want {
				t.Fatalf("got %v want %v", got, tc.want)
			}
		})
	}
}
