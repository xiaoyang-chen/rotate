//go:build linux

package rotate

import (
	"os"
	"syscall"

	"github.com/pkg/errors"
)

// _osChown is a var so we can mock it out during tests.
var _osChown = os.Chown

// chown name -> file must be exist
func chown(name string, info os.FileInfo) (err error) {

	var stat = info.Sys().(*syscall.Stat_t)
	if err = _osChown(name, int(stat.Uid), int(stat.Gid)); err != nil {
		err = errors.Wrapf(err, "chown file: %s to (uid: %d, gid: %d) fail", name, int(stat.Uid), int(stat.Gid))
	}
	return
}
