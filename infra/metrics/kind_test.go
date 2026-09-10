package metrics

import "testing"

func TestGetKindNameUsesBoundedLabels(t *testing.T) {
	tests := []struct {
		name string
		kind int
		want string
	}{
		{name: "known kind", kind: 1, want: "KindTextNote"},
		{name: "unknown positive kind", kind: 123456789, want: "other"},
		{name: "unknown negative kind", kind: -1, want: "other"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := GetKindName(tt.kind); got != tt.want {
				t.Fatalf("GetKindName(%d) = %q, want %q", tt.kind, got, tt.want)
			}
		})
	}
}
