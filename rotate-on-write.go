package rotate

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/pkg/errors"
)

const (
	_backupTimeFormat = "2006-01-02T15-04-05.000"
	_defaultMaxSize   = 5
)

var (
	// _currentTime exists so it can be mocked out by tests.
	_currentTime = time.Now
	// _os_Stat exists so it can be mocked out by tests.
	_os_Stat = os.Stat
	// _megabyte is the conversion factor between MaxSize and bytes.  It is a
	// variable so tests can mock it out and not need to write megabytes of data
	// to disk.
	_megabyte = 1024 * 1024
	// ensure we always implement io.Writer
	_ io.Writer = (*RotateOnWrite)(nil)
)

// RotateOnWrite is an io.Writer that writes to the specified filename.Noted that this is designed for single thread because more efficient, if running in multi threads env, data race will happen.
//
// RotateOnWrite opens or creates the file on first Write.
// If the file exists, the file is renamed by putting the current time in a timestamp in the name immediately
// before the file's extension (or the end of the filename if there's no extension).
// A new file is then created using original filename.
// if the file not exists, the file will be create for writing.
//
// Whenever a write would write length of byte exceed MaxSize megabytes,
// it will return error.
//
// Backups use the file name given to RotateOnWrite, in the form
// `name-timestamp.ext` where name is the filename without the extension,
// timestamp is the time at which the file was rotated formatted with the
// time.Time format of `2006-01-02T15-04-05.000` and the extension is the
// original extension.  For example, if your RotateOnWrite.Filename is
// `/var/log/foo/server.log`, a backup created at 6:30pm on Nov 11 2016 would
// use the filename `/var/log/foo/server-2016-11-04T18-30-00.000.log`
//
// # Cleaning Up Old Files
//
// Whenever a new file gets created, old files may be deleted.  The most
// recent files according to the encoded timestamp will be retained, up to a
// number equal to MaxBackups (or all of them if MaxBackups is 0).  Any files
// with an encoded timestamp older than MaxAge days are deleted, regardless of
// MaxBackups.  Note that the time encoded in the timestamp is the rotation
// time, which may differ from the last time that file was written to.
//
// If MaxBackups and MaxAge are both 0, no old files will be deleted.
type RotateOnWrite struct {
	// Filename is the file to write bytes to.  Backup files will be retained
	// in the same directory if BackupDir not set.  It uses
	// <processname>-rotate-on-write.log in os.TempDir() if empty.
	Filename string `json:"filename,omitempty" yaml:"filename,omitempty"`
	// BackupDir is the backup file dir, if not set, backup files will
	// be retained in the same of Filename directory, if set,
	// must be ensure BackupDir and Filename directory in same volume,
	// or it will happen err("invalid cross-device link") when os.Rename.
	BackupDir string `json:"backup_dir,omitempty" yaml:"backup_dir,omitempty"`
	// MaxSize is the maximum size in megabytes when writing every time. It defaults to 5 megabytes.
	MaxSize int `json:"max_size,omitempty" yaml:"max_size,omitempty"`
	// MaxAge is the maximum time to retain old files based on the
	// timestamp encoded in their filename.  Note that it base on time.Nanosecond (the minimal unit of time). The default is not to remove old files based on age.
	MaxAge time.Duration `json:"max_age,omitempty" yaml:"max_age,omitempty"`
	// MaxBackups is the maximum number of old files to retain.  The default
	// is to retain all old files (though MaxAge may still cause them to get
	// deleted.)
	MaxBackups int `json:"max_backups,omitempty" yaml:"max_backups,omitempty"`
	// LocalTime determines if the time used for formatting the timestamps in
	// backup files is the computer's local time.  The default is to use UTC
	// time.
	LocalTime bool `json:"local_time,omitempty" yaml:"local_time,omitempty"`
	// NotWriteIfEmpty determines if writing when
	// len(p) == 0 in "Write(p []byte)", The default is writing when empty.
	NotWriteIfEmpty bool `json:"not_write_if_empty,omitempty" yaml:"not_write_if_empty,omitempty"`
	// maxSize calculate when MaxSize setting, so we need not calc every time
	maxSize      int
	filenameBase string
	filenameExt  string
	filenameDir  string
	backupDir    string
	// isBackupNotInSameDir true if backupDir is not same as filenameDir.
	isBackupNotInSameDir bool
}

// Write implements io.Writer.
// a write would cause the file renamed to include a timestamp of the
// current time, and a new file is created using the original file name.
// If the length of the write is greater than MaxSize, an error is returned.
func (row *RotateOnWrite) Write(p []byte) (n int, err error) {

	var lenP = len(p)
	var maxSize = row.max()
	if lenP > maxSize {
		err = errors.Errorf("write length %d exceeds maximum file size %d", lenP, maxSize)
		return
	}

	n, err = row.rotateOnWrite(p, lenP)
	return
}

func (row *RotateOnWrite) rotateOnWrite(p []byte, lenP int) (n int, err error) {

	// if file is exist, rename and open a new file, then write
	// if is not exist, open a new file and write
	// finally close file
	// note: need to clear old file because row.MaxAge, row.MaxBackups

	var filename = row.filename()
	var dir = row.getFilenameDir()
	if err = os.MkdirAll(dir, 0744); err != nil {
		err = errors.Wrapf(err, "can't make directories: %s for new file: %s", dir, filename)
		return
	}
	var backupDir = row.getBackupDir()
	if row.isBackupNotInSameDir {
		if err = os.MkdirAll(backupDir, 0744); err != nil {
			err = errors.Wrapf(err, "can't make backup directories: %s for new file: %s", backupDir, filename)
			return
		}
	}
	var info os.FileInfo
	var mode = os.FileMode(0644)
	var isExist bool
	switch info, err = _os_Stat(filename); {
	case os.IsNotExist(err):
		err = nil // we need to set err to nil here
	case err == nil:
		// Copy the mode off the old logfile.
		mode = info.Mode()
		// move the existing file
		var base = row.getFilenameBase()
		var ext = row.getFilenameExt()
		var t = _currentTime()
		if !row.LocalTime {
			t = t.UTC()
		}
		var bkName = filepath.Join(backupDir, fmt.Sprintf("%s-%s%s", base[:len(base)-len(ext)], t.Format(_backupTimeFormat), ext))
		if err = os.Rename(filename, bkName); err != nil {
			err = errors.Wrapf(err, "can't rename file, oldName: %s, newName: %s", filename, bkName)
			return
		}
		isExist = true
	default:
		err = errors.Wrapf(err, "get file: %s info fail", filename)
		return
	}
	// judge whether writing, if not write, just check backup file
	if row.NotWriteIfEmpty && lenP == 0 {
		go row.millRunOnce() // todo what am I going to do for err from row.millRunOnce(), log this?
		return
	}
	// open a new file and write, finally close file
	// we use truncate here because this should only get called when we've moved
	// the file ourselves. if someone else creates the file in the meantime,
	// just wipe out the contents.
	var file *os.File
	if file, err = os.OpenFile(filename, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, mode); err != nil {
		err = errors.Wrapf(err, "can't open new file: %s", filename)
		return
	}
	defer file.Close()
	if isExist {
		// this is a no-op anywhere but linux
		if err = chown(filename, info); err != nil {
			return
		}
	}
	// write
	if n, err = file.Write(p); err != nil {
		err = errors.Wrapf(err, "write len: %d fail, %s: %s", lenP, filename, string(p))
		return
	}
	go row.millRunOnce() // todo what am I going to do for err from row.millRunOnce(), log this?
	return
}

// max returns the maximum size in bytes of files before rolling.
func (row *RotateOnWrite) max() (max int) {

	switch {
	case row.maxSize != 0:
	case row.MaxSize != 0:
		row.maxSize = row.MaxSize * _megabyte
	default:
		row.maxSize = _defaultMaxSize * _megabyte
	}
	max = row.maxSize
	return
}

func (row *RotateOnWrite) getFilenameBase() (base string) {

	if row.filenameBase == "" {
		row.filenameBase = filepath.Base(row.filename())
	}
	base = row.filenameBase
	return
}

func (row *RotateOnWrite) getFilenameExt() (ext string) {

	if row.filenameExt == "" {
		row.filenameExt = filepath.Ext(row.getFilenameBase())
	}
	ext = row.filenameExt
	return
}

func (row *RotateOnWrite) getFilenameDir() (dir string) {

	if row.filenameDir == "" {
		row.filenameDir = filepath.Dir(row.filename())
	}
	dir = row.filenameDir
	return
}

// filename genFilename generates the name of the file from the current time.
func (row *RotateOnWrite) filename() (filename string) {

	if row.Filename == "" {
		row.Filename = filepath.Join(os.TempDir(), filepath.Base(os.Args[0])+"-rotate-on-write.log")
	}
	filename = row.Filename
	return
}

// filename genFilename generates the name of the file from the current time.
func (row *RotateOnWrite) getBackupDir() (dir string) {

	if row.backupDir == "" {
		var filenameDir = row.getFilenameDir()
		if row.BackupDir == "" {
			row.BackupDir, row.backupDir = filenameDir, filenameDir
		} else {
			row.backupDir = filepath.Clean(row.BackupDir) // if parse not correct, consider use filepath.Dir(filepath.Join(row.BackupDir, "x"))
			row.isBackupNotInSameDir = row.backupDir != filenameDir
		}
	}
	dir = row.backupDir
	return
}

// millRunOnce performs removal of stale files. old files are removed, keeping at most row.MaxBackups files, as long as none of them are older than MaxAge.
func (row *RotateOnWrite) millRunOnce() (err error) {

	if row.MaxBackups == 0 && row.MaxAge == 0 {
		return
	}

	var fnwts []fNameWithT
	if fnwts, err = row.oldFiles(); err != nil {
		return
	}
	var lenFnwts = len(fnwts)
	if lenFnwts == 0 {
		return
	}
	var rmIdx = lenFnwts
	var fnwtsForRm = fnwts
	if row.MaxBackups > 0 && row.MaxBackups < lenFnwts {
		rmIdx = row.MaxBackups
		fnwts = fnwts[:row.MaxBackups]
	}
	if row.MaxAge > 0 {
		var cutoff = _currentTime().Add(-1 * row.MaxAge)
		for i, f := range fnwts {
			if f.t.Before(cutoff) {
				rmIdx = i
				break
			}
		}
	}
	var errRm error
	for _, f := range fnwtsForRm[rmIdx:] {
		// todo should we record all errRm?
		if errRm = os.Remove(filepath.Join(row.getBackupDir(), f.fName)); err == nil && errRm != nil {
			err = errors.Wrapf(errRm, "rm file: %s fail", f.fName)
		}
	}
	return
}

// oldFiles returns the list of backup files stored in the same
// directory as the current file, sorted by time(from big to small) in name of file
func (row *RotateOnWrite) oldFiles() (fnwts []fNameWithT, err error) {

	var fileInfos []os.DirEntry
	var dir = row.getBackupDir()
	if fileInfos, err = os.ReadDir(dir); err != nil {
		err = errors.Wrapf(err, "can't read backup file directory: %s", dir)
		return
	}

	fnwts = make([]fNameWithT, 0, len(fileInfos))
	var base = row.getFilenameBase()
	var ext = row.getFilenameExt()
	var prefix = base[:len(base)-len(ext)] + "-"
	var fName, ts string
	var parseE error
	var t time.Time
	for _, f := range fileInfos {
		if f.IsDir() {
			continue
		}
		fName = f.Name()
		if strings.HasPrefix(fName, prefix) && strings.HasSuffix(fName, ext) {
			ts = fName[len(prefix) : len(fName)-len(ext)]
			if t, parseE = time.Parse(_backupTimeFormat, ts); parseE == nil {
				fnwts = append(fnwts, fNameWithT{fName: fName, t: t})
			}
			// error parsing means that the suffix at the end was not generated
			// by rotate-on-write, and therefore it's not a backup file.
		}
	}
	sort.Sort(byFormatTime(fnwts))
	return
}

// fNameWithT is a convenience struct to return the filename and its embedded time.Time.
type fNameWithT struct {
	fName string
	t     time.Time
}

// byFormatTime sorts by newest time formatted in the name.
type byFormatTime []fNameWithT

func (b byFormatTime) Less(i, j int) bool { return b[i].t.After(b[j].t) }

func (b byFormatTime) Swap(i, j int) { b[i], b[j] = b[j], b[i] }

func (b byFormatTime) Len() int { return len(b) }
