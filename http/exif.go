package fbhttp

import (
	"net/http"

	"github.com/filebrowser/filebrowser/v2/exifmeta"
	"github.com/filebrowser/filebrowser/v2/files"
)

var exifHandler = withUser(func(w http.ResponseWriter, r *http.Request, d *data) (int, error) {
	if !d.user.Perm.Download {
		return http.StatusAccepted, nil
	}

	file, err := files.NewFileInfo(&files.FileOptions{
		Fs:         d.user.Fs,
		Path:       r.URL.Path,
		Modify:     d.user.Perm.Modify,
		Expand:     false,
		ReadHeader: d.server.TypeDetectionByHeader,
		Checker:    d,
	})
	if err != nil {
		return errToStatus(err), err
	}
	if file.IsDir {
		return http.StatusBadRequest, nil
	}

	fd, err := file.Fs.Open(file.Path)
	if err != nil {
		return errToStatus(err), err
	}
	defer fd.Close()

	meta, err := exifmeta.Read(fd)
	if err != nil {
		return http.StatusInternalServerError, err
	}

	return renderJSON(w, r, meta)
})
