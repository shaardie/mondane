package checks

import (
	"testing"
	"time"

	"github.com/shaardie/mondane/database"
)

func TestTLSCheck_Check(t *testing.T) {
	tests := []struct {
		name  string
		t     time.Time
		check database.TLSCheck
		want  bool
	}{
		{
			name: "google",
			check: database.TLSCheck{
				Host: "google.de",
				Port: 443,
			},
			want: true,
		},
		{
			name: "expired",
			check: database.TLSCheck{
				Host: "expired.badssl.com",
				Port: 443,
			},
			want: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tc := NewTLSCheck(tt.check)
			got, _ := tc.Check(tt.t)
			if got.(*database.TLSResult).Success != tt.want {
				t.Errorf("TLSCheck.Check() = %v, want %v", got, tt.want)
			}
		})
	}
}
