package api

import "testing"

func TestSupportedAssetContentType(t *testing.T) {
	tests := []struct {
		name     string
		declared string
		data     []byte
		want     string
	}{
		{
			name:     "uses allowed declared image type",
			declared: "image/heic",
			data:     []byte("heic bytes"),
			want:     "image/heic",
		},
		{
			name: "detects PDF when header is absent",
			data: []byte("%PDF-1.7\n"),
			want: "application/pdf",
		},
		{
			name: "detects plain text when header is absent",
			data: []byte("a captured note"),
			want: "text/plain",
		},
		{
			name:     "rejects HTML",
			declared: "text/html",
			data:     []byte("<html><script>alert(1)</script></html>"),
			want:     "",
		},
		{
			name:     "rejects SVG",
			declared: "image/svg+xml",
			data:     []byte("<svg></svg>"),
			want:     "",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := supportedAssetContentType(tc.declared, tc.data); got != tc.want {
				t.Fatalf("supportedAssetContentType() = %q, want %q", got, tc.want)
			}
		})
	}
}
