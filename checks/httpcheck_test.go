package checks

import (
	"reflect"
	"testing"
	"time"

	"github.com/shaardie/mondane/database"
)

func TestHTTPCheck_Check(t *testing.T) {
	tests := []struct {
		name  string
		t     time.Time
		check database.HTTPCheck
		want  interface{}
	}{
		{
			name: "http",
			check: database.HTTPCheck{
				URL: "http://example.com",
			},
			want: &database.HTTPResult{
				Success: true,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			hc := NewHTTPCheck(tt.check, nil)
			got, _ := hc.Check(tt.t)
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("HTTPCheck.Check() = %v, want %v", got, tt.want)
			}
		})
	}
}
