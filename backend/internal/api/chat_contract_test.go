package api

import "testing"

func TestSelectTierContract(t *testing.T) {
  tests := []struct {
    name string
    req  ChatRequest
    want string
  }{
    {name: "explicit tier wins", req: ChatRequest{Message: "hello", Tier: "quick"}, want: "quick"},
    {name: "short message defaults quick", req: ChatRequest{Message: "hi"}, want: "quick"},
    {name: "long message defaults deep", req: ChatRequest{Message: string(make([]byte, 201))}, want: "deep"},
    {name: "reasoning defaults deep", req: ChatRequest{Message: "plan my quarter", RequiresReasoning: true}, want: "deep"},
    {name: "middle message defaults standard", req: ChatRequest{Message: "please summarize my notes and tasks for today"}, want: "standard"},
  }

  for _, tc := range tests {
    t.Run(tc.name, func(t *testing.T) {
      if got := selectTier(tc.req); got != tc.want {
        t.Fatalf("selectTier() = %q, want %q", got, tc.want)
      }
    })
  }
}
