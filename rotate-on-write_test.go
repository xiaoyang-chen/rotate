package rotate

import (
	"reflect"
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

func TestRotateOnWrite_oldFiles(t *testing.T) {

	var wantFnwts = make([]fNameWithT, 3)
	var strTimes = []string{"2023.06.08-09.29.10.177", "2023.06.08-08.47.18.506", "2023.06.08-08.46.18.493"}
	var fNames = []string{"a-2023-06-08T09-29-10.177.json", "a-2023-06-08T08-47-18.506.json", "a-2023-06-08T08-46-18.493.json"}
	for i := range wantFnwts {
		wantFnwts[i].fName = fNames[i]
		if t, err := time.Parse("2006.01.02-15.04.05.000", strTimes[i]); err != nil {
			panic(err)
		} else {
			wantFnwts[i].t = t
		}
	}

	type fields struct {
		Filename             string
		BackupDir            string
		MaxSize              int
		MaxAge               time.Duration
		MaxBackups           int
		LocalTime            bool
		NotWriteIfEmpty      bool
		maxSize              int
		filenameBase         string
		filenameExt          string
		filenameDir          string
		backupDir            string
		isBackupNotInSameDir bool
	}
	tests := []struct {
		name      string
		fields    fields
		wantFnwts []fNameWithT
		wantErr   bool
	}{
		{
			name: "test",
			fields: fields{
				Filename:             "./test/test/a.json",
				BackupDir:            "./test/backup/test",
				MaxSize:              10,
				MaxAge:               86400 * time.Second,
				MaxBackups:           10,
				LocalTime:            false,
				NotWriteIfEmpty:      true,
				maxSize:              0,
				filenameBase:         "",
				filenameExt:          "",
				filenameDir:          "",
				backupDir:            "",
				isBackupNotInSameDir: false,
			},
			wantFnwts: wantFnwts,
			wantErr:   false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			row := &RotateOnWrite{
				Filename:             tt.fields.Filename,
				BackupDir:            tt.fields.BackupDir,
				MaxSize:              tt.fields.MaxSize,
				MaxAge:               tt.fields.MaxAge,
				MaxBackups:           tt.fields.MaxBackups,
				LocalTime:            tt.fields.LocalTime,
				NotWriteIfEmpty:      tt.fields.NotWriteIfEmpty,
				maxSize:              tt.fields.maxSize,
				filenameBase:         tt.fields.filenameBase,
				filenameExt:          tt.fields.filenameExt,
				filenameDir:          tt.fields.filenameDir,
				backupDir:            tt.fields.backupDir,
				isBackupNotInSameDir: tt.fields.isBackupNotInSameDir,
			}
			gotFnwts, err := row.oldFiles()
			if (err != nil) != tt.wantErr {
				t.Errorf("RotateOnWrite.oldFiles() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(gotFnwts, tt.wantFnwts) {
				t.Errorf("RotateOnWrite.oldFiles() = %v, want %v", gotFnwts, tt.wantFnwts)
			}
		})
	}
}

func TestRotateOnWrite_getFilenameDir(t *testing.T) {
	type fields struct {
		Filename             string
		BackupDir            string
		MaxSize              int
		MaxAge               time.Duration
		MaxBackups           int
		LocalTime            bool
		NotWriteIfEmpty      bool
		maxSize              int
		filenameBase         string
		filenameExt          string
		filenameDir          string
		backupDir            string
		isBackupNotInSameDir bool
	}
	tests := []struct {
		name    string
		fields  fields
		wantDir string
	}{
		{
			name: "test",
			fields: fields{
				Filename:             "/test/test-1/test-2.json",
				BackupDir:            "",
				MaxSize:              0,
				MaxAge:               0,
				MaxBackups:           0,
				LocalTime:            false,
				NotWriteIfEmpty:      false,
				maxSize:              0,
				filenameBase:         "",
				filenameExt:          "",
				filenameDir:          "",
				backupDir:            "",
				isBackupNotInSameDir: false,
			},
			wantDir: "/test/test-1",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			row := &RotateOnWrite{
				Filename:             tt.fields.Filename,
				BackupDir:            tt.fields.BackupDir,
				MaxSize:              tt.fields.MaxSize,
				MaxAge:               tt.fields.MaxAge,
				MaxBackups:           tt.fields.MaxBackups,
				LocalTime:            tt.fields.LocalTime,
				NotWriteIfEmpty:      tt.fields.NotWriteIfEmpty,
				maxSize:              tt.fields.maxSize,
				filenameBase:         tt.fields.filenameBase,
				filenameExt:          tt.fields.filenameExt,
				filenameDir:          tt.fields.filenameDir,
				backupDir:            tt.fields.backupDir,
				isBackupNotInSameDir: tt.fields.isBackupNotInSameDir,
			}
			if gotDir := row.getFilenameDir(); gotDir != tt.wantDir {
				t.Errorf("RotateOnWrite.getFilenameDir() = %v, want %v", gotDir, tt.wantDir)
			}
		})
	}
}
