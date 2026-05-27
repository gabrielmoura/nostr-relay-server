package blossom

import "testing"

func TestValidateMirrorRequest(t *testing.T) {
	t.Parallel()

	valid := mirrorRequest{
		URL:    "https://origin.example.com/blob/abc",
		SHA256: "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef",
	}
	if err := validateMirrorRequest(valid); err != nil {
		t.Fatalf("validateMirrorRequest(valid) error = %v", err)
	}

	invalidCases := []mirrorRequest{
		{URL: "", SHA256: valid.SHA256},
		{URL: "/relative", SHA256: valid.SHA256},
		{URL: "ftp://origin.example.com/blob/abc", SHA256: valid.SHA256},
		{URL: valid.URL, SHA256: "ABCDEF"},
	}

	for _, tc := range invalidCases {
		if err := validateMirrorRequest(tc); err == nil {
			t.Fatalf("validateMirrorRequest(%+v) expected error", tc)
		}
	}
}
