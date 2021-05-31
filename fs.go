// +build go1.16

package render

import (
	"bytes"
	"fmt"
	"html/template"
	"io"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"time"
)

type FS fs.FS

func LocalFS(dir string) FS {
	return os.DirFS(dir)
}

func (r *Render) compileTemplatesFromFS() error {
	dir := "."
	tmpTemplates := template.New(dir)
	tmpTemplates.Delims(r.opt.Delims.Left, r.opt.Delims.Right)

	var errs []error

	err := fs.WalkDir(r.opt.FileSystem, dir, func(path string, info fs.DirEntry, err error) error {
		// Fix same-extension-dirs bug: some dir might be named to: "users.tmpl", "local.html".
		// These dirs should be excluded as they are not valid golang templates, but files under
		// them should be treat as normal.
		// If is a dir, return immediately (dir is not a valid golang template).
		if info == nil || info.IsDir() {
			return nil
		}

		ext := filepath.Ext(path)

		for _, extension := range r.opt.Extensions {
			if ext != extension {
				continue
			}

			tmplFile, err := r.opt.FileSystem.Open(path)
			if err != nil {
				errs = append(errs, err)
				break
			}

			tmplBytes, err := io.ReadAll(tmplFile)
			if err != nil {
				errs = append(errs, err)
				break
			}

			name := path[0 : len(path)-len(ext)]
			tmpl := tmpTemplates.New(filepath.ToSlash(name))

			// Add our funcmaps.
			for _, funcs := range r.opt.Funcs {
				tmpl = tmpl.Funcs(funcs)
			}

			// Add helperFuncs
			tmpl = tmpl.Funcs(helperFuncs)

			// Parse template
			tmpl, err = tmpl.Parse(string(tmplBytes))
			if err != nil {
				errs = append(errs, err)
				break
			}

			tmpTemplates = tmpl
		}

		return nil
	})
	if err != nil {
		return err
	} else if len(errs) > 0 {
		return fmt.Errorf("one or more errors occurred while loading or parsing templates: %+v", errs)
	}

	r.lock.Lock()
	defer r.lock.Unlock()
	r.templates = tmpTemplates

	if watcherFs, isWatcherFs := r.opt.FileSystem.(WatcherFS); isWatcherFs {
		var watchErr error

		go func() {
			watcherEventChan, err := watcherFs.Watch()
			if err != nil {
				watchErr = err
				return
			}

			<-watcherEventChan
			err = r.CompileTemplates()
			if err != nil {
				watchErr = err
				return
			}
		}()

		if watchErr != nil {
			return watchErr
		}
	}

	return nil
}

func (a *AssetFS) Open(name string) (fs.File, error) {
	name = path.Clean(name)
	names := a.AssetNames()

	if name == "." {
		// List assets
		var ents []*staticDirEntry
		for _, n := range names {
			ents = append(ents, &staticDirEntry{
				fileInfo: &staticFileInfo{
					basename: n,
					dir:      false,
				},
			})
		}

		return &staticFile{
			reader: nil,
			fileInfo: &staticFileInfo{
				basename: name,
				dir:      true,
				ents:     ents,
			},
		}, nil
	}

	// Find asset
	for _, n := range names {
		if n == name {
			assetContent, err := a.Asset(name)
			if err != nil {
				return nil, err
			}

			return &staticFile{
				reader: bytes.NewReader(assetContent),
				fileInfo: &staticFileInfo{
					basename: name,
					dir:      false,
				},
			}, nil
		}
	}

	return nil, fs.ErrNotExist
}

type staticFile struct {
	reader   io.ReadSeeker
	fileInfo *staticFileInfo
	entpos   int
}

var _ fs.File = &staticFile{}

func (s *staticFile) Close() error {
	if s.reader == nil {
		return fs.ErrInvalid
	}
	return nil
}

func (s *staticFile) Read(p []byte) (n int, err error) {
	if s.reader == nil {
		return 0, fs.ErrInvalid
	}
	return s.reader.Read(p)
}

func (s *staticFile) Seek(offset int64, whence int) (int64, error) {
	if s.reader == nil {
		return 0, fs.ErrInvalid
	}
	return s.reader.Seek(offset, whence)
}

var _ fs.ReadDirFile = &staticFile{}

func (s *staticFile) ReadDir(count int) ([]fs.DirEntry, error) {
	if !s.fileInfo.dir {
		return nil, fs.ErrInvalid
	}
	var fis []fs.DirEntry

	limit := s.entpos + count
	if count <= 0 || limit > len(s.fileInfo.ents) {
		limit = len(s.fileInfo.ents)
	}
	for ; s.entpos < limit; s.entpos++ {
		fis = append(fis, s.fileInfo.ents[s.entpos])
	}

	if len(fis) == 0 && count > 0 {
		return fis, io.EOF
	} else {
		return fis, nil
	}
}

func (s *staticFile) Stat() (fs.FileInfo, error) { return s.fileInfo, nil }

type staticFileInfo struct {
	basename string
	size     int64
	dir      bool
	modtime  time.Time
	ents     []*staticDirEntry
}

var _ fs.FileInfo = &staticFileInfo{}

func (s *staticFileInfo) Name() string { return s.basename }
func (s *staticFileInfo) Size() int64  { return s.size }
func (s *staticFileInfo) Mode() fs.FileMode {
	if s.dir {
		return 0755 | fs.ModeDir
	}
	return 0644
}
func (s *staticFileInfo) ModTime() time.Time { return s.modtime }
func (s *staticFileInfo) IsDir() bool        { return s.dir }
func (s *staticFileInfo) Sys() interface{}   { return nil }

type staticDirEntry struct {
	fileInfo *staticFileInfo
}

var _ fs.DirEntry = &staticDirEntry{}

func (s *staticDirEntry) Name() string              { return s.fileInfo.Name() }
func (s *staticDirEntry) IsDir() bool               { return s.fileInfo.IsDir() }
func (s staticDirEntry) Type() fs.FileMode          { return s.fileInfo.Mode() }
func (s staticDirEntry) Info() (fs.FileInfo, error) { return s.fileInfo, nil }
