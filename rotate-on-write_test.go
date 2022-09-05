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

func TestRotateOnWrite_Write(t *testing.T) {

	var testP = []byte(`{"test": "testP"}`)
	// testP = []byte{} // NotWriteIfEmpty test
	type fields struct {
		Filename        string
		BackupDir       string
		MaxSize         int
		MaxAge          time.Duration
		MaxBackups      int
		LocalTime       bool
		NotWriteIfEmpty bool
		// maxSize              int
		// filenameBase         string
		// filenameExt          string
		// filenameDir          string
		// backupDir            string
		// isBackupNotInSameDir bool
		// millCh               chan struct{}
		// startMill            sync.Once
	}
	type args struct {
		p []byte
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		wantN   int
		wantErr bool
	}{
		{
			name: "test",
			fields: fields{
				Filename:   "./test-dir/test/test-p.json",
				BackupDir:  "./test-dir/backup",
				MaxSize:    10,
				MaxBackups: 100,
				// NotWriteIfEmpty: true, // NotWriteIfEmpty test
			},
			args: args{
				p: testP,
			},
			wantN:   len(testP),
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			row := &RotateOnWrite{
				Filename:        tt.fields.Filename,
				BackupDir:       tt.fields.BackupDir,
				MaxSize:         tt.fields.MaxSize,
				MaxAge:          tt.fields.MaxAge,
				MaxBackups:      tt.fields.MaxBackups,
				LocalTime:       tt.fields.LocalTime,
				NotWriteIfEmpty: tt.fields.NotWriteIfEmpty,
			}
			gotN, err := row.Write(tt.args.p)
			if (err != nil) != tt.wantErr {
				t.Errorf("RotateOnWrite.Write() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if gotN != tt.wantN {
				t.Errorf("RotateOnWrite.Write() = %v, want %v", gotN, tt.wantN)
			}
		})
	}
}
