package fbhttp

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"log"
	"mime"
	"net/http"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	fberrors "github.com/filebrowser/filebrowser/v2/errors"
	"github.com/filebrowser/filebrowser/v2/files"
	"github.com/filebrowser/filebrowser/v2/fileutils"
	"github.com/maruel/natural"
	"github.com/shirou/gopsutil/v4/disk"
	"github.com/spf13/afero"
)

var resourceGetHandler = withUser(func(w http.ResponseWriter, r *http.Request, d *data) (int, error) {
	file, err := files.NewFileInfo(&files.FileOptions{
		Fs:         d.user.Fs,
		Path:       r.URL.Path,
		Modify:     d.user.Perm.Modify,
		Expand:     true,
		ReadHeader: d.server.TypeDetectionByHeader,
		Checker:    d,
		Content:    d.user.Perm.Download,
	})
	if err != nil {
		return errToStatus(err), err
	}

	encoding := r.Header.Get("X-Encoding")
	if file.IsDir {
		file.Sorting = d.user.Sorting
		file.ApplySort()
		return renderJSON(w, r, file)
	} else if encoding == "true" {
		if !d.user.Perm.Download {
			return http.StatusAccepted, nil
		}
		if file.Type != "text" {
			return renderJSON(w, r, file)
		}

		f, err := d.user.Fs.Open(r.URL.Path)
		if err != nil {
			return errToStatus(err), err
		}
		defer f.Close()

		data, err := io.ReadAll(f)
		if err != nil {
			return http.StatusInternalServerError, err
		}

		w.Header().Set("Content-Type", "application/octet-stream")
		w.WriteHeader(http.StatusOK)
		_, err = w.Write(data)
		return 0, err
	}

	if checksum := r.URL.Query().Get("checksum"); checksum != "" {
		if !d.user.Perm.Download {
			return http.StatusAccepted, nil
		}

		err := file.Checksum(checksum)
		if errors.Is(err, fberrors.ErrInvalidOption) {
			return http.StatusBadRequest, nil
		} else if err != nil {
			return http.StatusInternalServerError, err
		}

		// do not waste bandwidth if we just want the checksum
		file.Content = ""
	}

	return renderJSON(w, r, file)
})

func resourceDeleteHandler(fileCache FileCache) handleFunc {
	return withUser(func(_ http.ResponseWriter, r *http.Request, d *data) (int, error) {
		if r.URL.Path == "/" || !d.user.Perm.Delete {
			return http.StatusForbidden, nil
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

		if err = checkDescendants(d, r.URL.Path, ""); err != nil {
			return errToStatus(err), err
		}

		err = d.store.Share.DeleteWithPathPrefix(file.Path, d.user.ID)
		if err != nil {
			log.Printf("WARNING: Error(s) occurred while deleting associated shares with file: %s", err)
		}

		// delete thumbnails
		err = delThumbs(r.Context(), fileCache, file)
		if err != nil {
			return errToStatus(err), err
		}

		err = d.RunHook(func() error {
			return d.user.Fs.RemoveAll(r.URL.Path)
		}, "delete", r.URL.Path, "", d.user)

		if err != nil {
			return errToStatus(err), err
		}

		return http.StatusNoContent, nil
	})
}

func resourcePostHandler(fileCache FileCache) handleFunc {
	return withUser(func(w http.ResponseWriter, r *http.Request, d *data) (int, error) {
		if !d.user.Perm.Create || !d.Check(r.URL.Path) {
			return http.StatusForbidden, nil
		}

		// Directories creation on POST.
		if strings.HasSuffix(r.URL.Path, "/") {
			err := d.RunHook(func() error {
				return d.user.Fs.MkdirAll(r.URL.Path, d.settings.DirMode)
			}, "upload", r.URL.Path, "", d.user)
			return errToStatus(err), err
		}

		file, err := files.NewFileInfo(&files.FileOptions{
			Fs:         d.user.Fs,
			Path:       r.URL.Path,
			Modify:     d.user.Perm.Modify,
			Expand:     false,
			ReadHeader: d.server.TypeDetectionByHeader,
			Checker:    d,
		})
		if err == nil {
			if r.URL.Query().Get("override") != "true" {
				return http.StatusConflict, nil
			}

			// Permission for overwriting the file
			if !d.user.Perm.Modify {
				return http.StatusForbidden, nil
			}

			err = delThumbs(r.Context(), fileCache, file)
			if err != nil {
				return errToStatus(err), err
			}
		}

		err = d.RunHook(func() error {
			info, writeErr := writeFile(d.user.Fs, r.URL.Path, r.Body, d.settings.FileMode, d.settings.DirMode)
			if writeErr != nil {
				return writeErr
			}

			etag := fmt.Sprintf(`"%x%x"`, info.ModTime().UnixNano(), info.Size())
			w.Header().Set("ETag", etag)
			return nil
		}, "upload", r.URL.Path, "", d.user)

		if err != nil {
			_ = d.user.Fs.RemoveAll(r.URL.Path)
		}

		return errToStatus(err), err
	})
}

var resourcePutHandler = withUser(func(w http.ResponseWriter, r *http.Request, d *data) (int, error) {
	if !d.user.Perm.Modify || !d.Check(r.URL.Path) {
		return http.StatusForbidden, nil
	}

	// Only allow PUT for files.
	if strings.HasSuffix(r.URL.Path, "/") {
		return http.StatusMethodNotAllowed, nil
	}

	exists, err := afero.Exists(d.user.Fs, r.URL.Path)
	if err != nil {
		return http.StatusInternalServerError, err
	}
	if !exists {
		return http.StatusNotFound, nil
	}

	err = d.RunHook(func() error {
		info, writeErr := writeFile(d.user.Fs, r.URL.Path, r.Body, d.settings.FileMode, d.settings.DirMode)
		if writeErr != nil {
			return writeErr
		}

		etag := fmt.Sprintf(`"%x%x"`, info.ModTime().UnixNano(), info.Size())
		w.Header().Set("ETag", etag)
		return nil
	}, "save", r.URL.Path, "", d.user)

	return errToStatus(err), err
})

func resourcePatchHandler(fileCache FileCache) handleFunc {
	return withUser(func(_ http.ResponseWriter, r *http.Request, d *data) (int, error) {
		src := r.URL.Path
		dst := r.URL.Query().Get("destination")
		action := r.URL.Query().Get("action")
		dst, err := url.QueryUnescape(dst)
		dst = slashClean(dst)
		src = slashClean(src)
		if !d.Check(src) || !d.Check(dst) {
			return http.StatusForbidden, nil
		}
		if err != nil {
			return errToStatus(err), err
		}
		if dst == "/" || src == "/" {
			return http.StatusForbidden, nil
		}

		err = checkParent(src, dst)
		if err != nil {
			return http.StatusBadRequest, err
		}

		srcInfo, _ := d.user.Fs.Stat(src)
		dstInfo, _ := d.user.Fs.Stat(dst)
		same := os.SameFile(srcInfo, dstInfo)

		if action != "rename" || !same {
			override := r.URL.Query().Get("override") == "true"
			rename := r.URL.Query().Get("rename") == "true"
			if !override && !rename {
				if _, err = d.user.Fs.Stat(dst); err == nil {
					return http.StatusConflict, nil
				}
			}
			if rename {
				dst = addVersionSuffix(dst, d.user.Fs)
			}

			if override && !d.user.Perm.Modify {
				return http.StatusForbidden, nil
			}
		}

		if err = checkDescendants(d, src, dst); err != nil {
			return errToStatus(err), err
		}

		err = d.RunHook(func() error {
			return patchAction(r.Context(), action, src, dst, d, fileCache)
		}, action, src, dst, d.user)

		return errToStatus(err), err
	})
}

// checkDescendants reports an error if the rules deny any path inside the src
// tree or, when dst is not empty, the location that tree would land on. The
// handlers only authorize the root of the operation, but copy, rename and
// delete then recurse through the whole subtree, so a rule denying a descendant
// of an allowed directory would be bypassed by operating on the parent instead.
func checkDescendants(d *data, src, dst string) error {
	if len(d.settings.Rules) == 0 && len(d.user.Rules) == 0 {
		return nil
	}

	return afero.Walk(d.user.Fs, src, func(fPath string, _ os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if !d.CheckRules(fPath) {
			return fberrors.ErrPermissionDenied
		}

		if dst == "" {
			return nil
		}

		rel, err := filepath.Rel(src, fPath)
		if err != nil {
			return err
		}

		if !d.CheckRules(filepath.Join(dst, rel)) {
			return fberrors.ErrPermissionDenied
		}

		return nil
	})
}

func checkParent(src, dst string) error {
	rel, err := filepath.Rel(src, dst)
	if err != nil {
		return err
	}

	rel = filepath.ToSlash(rel)
	if !strings.HasPrefix(rel, "../") && rel != ".." && rel != "." {
		return fberrors.ErrSourceIsParent
	}

	return nil
}

func addVersionSuffix(source string, afs afero.Fs) string {
	counter := 1
	dir, name := path.Split(source)
	ext := filepath.Ext(name)
	base := strings.TrimSuffix(name, ext)

	for {
		if _, err := afs.Stat(source); err != nil {
			break
		}
		renamed := fmt.Sprintf("%s(%d)%s", base, counter, ext)
		source = path.Join(dir, renamed)
		counter++
	}

	return source
}

func writeFile(afs afero.Fs, dst string, in io.Reader, fileMode, dirMode fs.FileMode) (os.FileInfo, error) {
	dir, _ := path.Split(dst)
	err := afs.MkdirAll(dir, dirMode)
	if err != nil {
		return nil, err
	}

	file, err := afs.OpenFile(dst, os.O_RDWR|os.O_CREATE|os.O_TRUNC, fileMode)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	_, err = io.Copy(file, in)
	if err != nil {
		return nil, err
	}

	// Sync the file to ensure all data is written to storage.
	// to prevent file corruption.
	if err := file.Sync(); err != nil {
		return nil, err
	}

	// Gets the info about the file.
	info, err := file.Stat()
	if err != nil {
		return nil, err
	}

	return info, nil
}

func delThumbs(ctx context.Context, fileCache FileCache, file *files.FileInfo) error {
	for _, previewSizeName := range PreviewSizeNames() {
		size, _ := ParsePreviewSize(previewSizeName)
		if err := fileCache.Delete(ctx, previewCacheKey(file, size)); err != nil {
			return err
		}
	}

	return nil
}

func patchAction(ctx context.Context, action, src, dst string, d *data, fileCache FileCache) error {
	switch action {
	case "copy":
		if !d.user.Perm.Create {
			return fberrors.ErrPermissionDenied
		}

		return fileutils.Copy(d.user.Fs, src, dst, d.settings.FileMode, d.settings.DirMode)
	case "rename":
		if !d.user.Perm.Rename {
			return fberrors.ErrPermissionDenied
		}
		src = path.Clean("/" + src)
		dst = path.Clean("/" + dst)

		file, err := files.NewFileInfo(&files.FileOptions{
			Fs:         d.user.Fs,
			Path:       src,
			Modify:     d.user.Perm.Modify,
			Expand:     false,
			ReadHeader: false,
			Checker:    d,
		})
		if err != nil {
			return err
		}

		// delete thumbnails
		err = delThumbs(ctx, fileCache, file)
		if err != nil {
			return err
		}

		return fileutils.MoveFile(d.user.Fs, src, dst, d.settings.FileMode, d.settings.DirMode)
	default:
		return fmt.Errorf("unsupported action %s: %w", action, fberrors.ErrInvalidRequestParams)
	}
}

// RecursiveEntry is a single file/directory entry returned by the recursive listing endpoint.
type RecursiveEntry struct {
	Path      string    `json:"path"`
	Name      string    `json:"name"`
	Size      int64     `json:"size"`
	ModTime   time.Time `json:"modified"`
	IsDir     bool      `json:"isDir"`
	Extension string    `json:"extension"`
}

// recursiveFilter holds the optional filtering and ordering of a recursive listing.
type recursiveFilter struct {
	query   string
	typ     string
	minSize int64
	maxSize int64
	after   time.Time
	before  time.Time
	sortBy  string
	asc     bool
}

func parseRecursiveFilter(r *http.Request) (*recursiveFilter, error) {
	q := r.URL.Query()
	f := &recursiveFilter{
		query:   strings.ToLower(q.Get("query")),
		typ:     strings.ToLower(q.Get("type")),
		sortBy:  q.Get("sortBy"),
		asc:     q.Get("asc") == "true",
		minSize: -1,
		maxSize: -1,
	}

	if raw := q.Get("minSize"); raw != "" {
		value, err := strconv.ParseInt(raw, 10, 64)
		if err != nil || value < 0 {
			return nil, fmt.Errorf("invalid minSize %q: %w", raw, fberrors.ErrInvalidRequestParams)
		}
		f.minSize = value
	}
	if raw := q.Get("maxSize"); raw != "" {
		value, err := strconv.ParseInt(raw, 10, 64)
		if err != nil || value < 0 {
			return nil, fmt.Errorf("invalid maxSize %q: %w", raw, fberrors.ErrInvalidRequestParams)
		}
		f.maxSize = value
	}
	for _, date := range []struct {
		name    string
		value   *time.Time
		nextDay bool
	}{
		{"after", &f.after, false},
		{"before", &f.before, true},
	} {
		raw := q.Get(date.name)
		if raw == "" {
			continue
		}
		value, err := time.ParseInLocation("2006-01-02", raw, time.Local)
		if err != nil {
			return nil, fmt.Errorf("invalid %s %q: %w", date.name, raw, fberrors.ErrInvalidRequestParams)
		}
		if date.nextDay {
			value = value.AddDate(0, 0, 1)
		}
		*date.value = value
	}

	return f, nil
}

func (f *recursiveFilter) matchType(entry RecursiveEntry) bool {
	switch f.typ {
	case "":
		return true
	case "dir":
		return entry.IsDir
	case "file":
		return !entry.IsDir
	}
	if entry.IsDir {
		return false
	}

	switch f.typ {
	case "image", "audio", "music", "video":
		prefix := f.typ
		if prefix == "music" {
			prefix = "audio"
		}
		return strings.HasPrefix(mime.TypeByExtension(entry.Extension), prefix)
	case "pdf":
		return strings.EqualFold(entry.Extension, ".pdf")
	default:
		return strings.EqualFold(entry.Extension, "."+strings.TrimPrefix(f.typ, "."))
	}
}

func (f *recursiveFilter) match(entry RecursiveEntry, rootPath string) bool {
	if f.query != "" {
		relative := strings.TrimPrefix(entry.Path, rootPath)
		if !strings.Contains(strings.ToLower(relative), f.query) {
			return false
		}
	}
	if f.minSize >= 0 && entry.Size < f.minSize {
		return false
	}
	if f.maxSize >= 0 && entry.Size > f.maxSize {
		return false
	}
	if !f.after.IsZero() && entry.ModTime.Before(f.after) {
		return false
	}
	if !f.before.IsZero() && !entry.ModTime.Before(f.before) {
		return false
	}
	return f.matchType(entry)
}

func (f *recursiveFilter) sort(entries []RecursiveEntry) {
	by := f.sortBy
	switch by {
	case "name", "path", "size", "modified":
	default:
		by = "name"
	}

	lessNatural := func(a, b string) bool {
		if f.asc {
			return natural.Less(strings.ToLower(a), strings.ToLower(b))
		}
		return natural.Less(strings.ToLower(b), strings.ToLower(a))
	}

	sort.SliceStable(entries, func(i, j int) bool {
		a, b := entries[i], entries[j]
		if a.IsDir != b.IsDir {
			return a.IsDir
		}

		switch by {
		case "size":
			if a.Size != b.Size {
				return (a.Size < b.Size) == f.asc
			}
		case "modified":
			if !a.ModTime.Equal(b.ModTime) {
				return a.ModTime.Before(b.ModTime) == f.asc
			}
		case "path":
			if a.Path != b.Path {
				return lessNatural(a.Path, b.Path)
			}
		default:
			if a.Name != b.Name {
				return lessNatural(a.Name, b.Name)
			}
		}
		return a.Path < b.Path
	})
}

const (
	recursiveFlushBytes = 16 * 1024
	recursiveFlushEvery = 50 * time.Millisecond
)

type recursiveStream struct {
	writer  *bufio.Writer
	flusher http.Flusher
	pending int
	last    time.Time
	first   bool
}

func newRecursiveStream(w http.ResponseWriter) *recursiveStream {
	flusher, _ := w.(http.Flusher)
	return &recursiveStream{
		writer:  bufio.NewWriterSize(w, 32*1024),
		flusher: flusher,
		last:    time.Now(),
		first:   true,
	}
}

func (s *recursiveStream) flush() error {
	if err := s.writer.Flush(); err != nil {
		return err
	}
	if s.flusher != nil {
		s.flusher.Flush()
	}
	s.pending = 0
	s.last = time.Now()
	s.first = false
	return nil
}

func (s *recursiveStream) write(entry RecursiveEntry) error {
	data, err := json.Marshal(entry)
	if err != nil {
		return err
	}
	data = append(data, '\n')
	if _, err := s.writer.Write(data); err != nil {
		return err
	}
	s.pending += len(data)
	// Send the first entry immediately, then batch the rest.
	if s.first || s.pending >= recursiveFlushBytes || time.Since(s.last) >= recursiveFlushEvery {
		return s.flush()
	}
	return nil
}

// resourceGetRecursiveHandler returns a flat list of every file and directory
// under the requested path, walking the tree recursively on the server side
// so the client only needs a single HTTP call. Optional query parameters
// filter and sort the entries before they are returned. With stream=true,
// entries are written as NDJSON while the tree is walked instead of being
// buffered into one response.
var resourceGetRecursiveHandler = withUser(func(w http.ResponseWriter, r *http.Request, d *data) (int, error) {
	rootPath := r.URL.Path
	if rootPath == "" {
		rootPath = "/"
	}

	filter, err := parseRecursiveFilter(r)
	if err != nil {
		return errToStatus(err), err
	}
	stream := r.URL.Query().Get("stream") == "true"
	var streamWriter *recursiveStream
	if stream {
		w.Header().Set("Content-Type", "application/x-ndjson; charset=utf-8")
		w.Header().Set("Cache-Control", "no-cache")
		streamWriter = newRecursiveStream(w)
	}

	// Make sure the root itself exists and is a directory.
	info, err := d.user.Fs.Stat(rootPath)
	if err != nil {
		return errToStatus(err), err
	}
	if !info.IsDir() {
		return http.StatusBadRequest, fmt.Errorf("path is not a directory")
	}

	entries := make([]RecursiveEntry, 0)
	root := filepath.ToSlash(rootPath)

	err = afero.Walk(d.user.Fs, rootPath, func(fPath string, info os.FileInfo, err error) error {
		// Walking a large tree is expensive, and every entry is stat'ed through
		// the scoped filesystem. Stop as soon as nobody is waiting for the
		// answer instead of finishing the walk for a client that has gone.
		if ctxErr := r.Context().Err(); ctxErr != nil {
			return ctxErr
		}

		if err != nil {
			return nil // skip entries we cannot read
		}

		// Skip the root directory itself.
		if fPath == rootPath {
			return nil
		}

		// Respect user rules.
		if !d.Check(fPath) {
			if info.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}

		entry := RecursiveEntry{
			// afero.Walk joins paths with the OS separator, so on Windows fPath
			// uses backslashes. The web API contract is forward slashes, so
			// normalize it here (mirrors search/search.go).
			Path:      filepath.ToSlash(fPath),
			Name:      info.Name(),
			Size:      info.Size(),
			ModTime:   info.ModTime(),
			IsDir:     info.IsDir(),
			Extension: filepath.Ext(info.Name()),
		}
		if !filter.match(entry, root) {
			return nil
		}
		if stream {
			return streamWriter.write(entry)
		}
		entries = append(entries, entry)
		return nil
	})
	if err != nil {
		// Once the request is done there is nobody left to answer, whatever the
		// walk reported. Asking for a status here only produces a second header
		// write on a connection that is already gone.
		if r.Context().Err() != nil || stream {
			return 0, err
		}
		return http.StatusInternalServerError, err
	}

	if stream {
		return 0, streamWriter.flush()
	}

	filter.sort(entries)

	return renderJSON(w, r, entries)
})

type DiskUsageResponse struct {
	Total uint64 `json:"total"`
	Used  uint64 `json:"used"`
}

var diskUsage = withUser(func(w http.ResponseWriter, r *http.Request, d *data) (int, error) {
	file, err := files.NewFileInfo(&files.FileOptions{
		Fs:         d.user.Fs,
		Path:       r.URL.Path,
		Modify:     d.user.Perm.Modify,
		Expand:     false,
		ReadHeader: false,
		Checker:    d,
		Content:    false,
	})
	if err != nil {
		return errToStatus(err), err
	}
	fPath := file.RealPath()
	if !file.IsDir {
		return renderJSON(w, r, &DiskUsageResponse{
			Total: 0,
			Used:  0,
		})
	}

	usage, err := disk.UsageWithContext(r.Context(), fPath)
	if err != nil {
		return errToStatus(err), err
	}
	return renderJSON(w, r, &DiskUsageResponse{
		Total: usage.Total,
		Used:  usage.Used,
	})
})
