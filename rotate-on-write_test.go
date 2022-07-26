package rotate

import (
	"sort"
	"testing"
	"time"
)

func Test_byFormatTime_Less(t *testing.T) {

	type args struct {
		i int
		j int
	}
	tests := []struct {
		name string
		b    byFormatTime
		args args
		want bool
	}{
		{
			name: "test",
			b: []fNameWithT{
				{
					fName: "1",
					t:     time.Now().Add(time.Hour),
				},
				{
					fName: "2",
					t:     time.Now().Add(-1 * time.Hour),
				},
				{
					fName: "0",
					t:     time.Now(),
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sort.Sort(tt.b)
			t.Log(tt.b)
		})
	}
}
