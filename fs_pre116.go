// +build !go1.16

package render

import (
	"bytes"
	"fmt"
	"html/template"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"time"
)

// FS is an alias for http.FileSystem for Go version < 1.16
type FS http.FileSystem

func LocalFS(dir string) FS {
	return http.Dir(dir)
}

func (r *Render) compileTemplatesFromFS() error {
	dir := "."
	tmpTemplates := template.New(dir)
	tmpTemplates.Delims(r.opt.Delims.Left, r.opt.Delims.Right)

	var errs []error

	err := walk(r.opt.FileSystem, dir, func(path string, info os.FileInfo, err error) error {
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

			tmplBytes, err := ioutil.ReadAll(tmplFile)
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

func walk(fsys FS, root string, fn filepath.WalkFunc) error {
	info, err := stat(fsys, root)

	if err != nil {
		err = fn(root, nil, err)
	} else {
		err = walkDir(fsys, root, info, fn)
	}
	if err == filepath.SkipDir {
		return nil
	}

	return err
}

func walkDir(fsys FS, name string, info os.FileInfo, walkDirFn filepath.WalkFunc) error {
	if err := walkDirFn(name, info, nil); err != nil || !info.IsDir() {
		if err == filepath.SkipDir && info.IsDir() {
			// Successfully skipped directory.
			err = nil
		}
		return err
	}

	dirs, err := readDir(fsys, name)
	if err != nil {
		// Second call, to report ReadDir error.
		err = walkDirFn(name, info, err)
		if err != nil {
			return err
		}
	}

	for _, d1 := range dirs {
		name1 := path.Join(name, d1.Name())
		if err := walkDir(fsys, name1, d1, walkDirFn); err != nil {
			if err == filepath.SkipDir {
				break
			}
			return err
		}
	}
	return nil
}

func stat(fs http.FileSystem, name string) (os.FileInfo, error) {
	f, err := fs.Open(name)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	return f.Stat()
}

func readDir(fsys FS, name string) ([]os.FileInfo, error) {
	file, err := fsys.Open(name)
	if err != nil {
		return nil, err
	}
	defer file.Close()
	return file.Readdir(-1)
}

func (a *AssetFS) Open(name string) (http.File, error) {
	name = path.Clean(name)
	names := a.AssetNames()

	if name == "." {
		// List assets
		var ents []*staticFileInfo
		for _, n := range names {
			ents = append(ents, &staticFileInfo{
				basename: n,
				dir:      false,
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

	return nil, os.ErrNotExist
}

type staticFile struct {
	reader   io.ReadSeeker
	fileInfo *staticFileInfo
	entpos   int
}

var _ http.File = &staticFile{}

func (s *staticFile) Close() error {
	if s.reader == nil {
		return os.ErrInvalid
	}
	return nil
}

func (s *staticFile) Read(p []byte) (n int, err error) {
	if s.reader == nil {
		return 0, os.ErrInvalid
	}
	return s.reader.Read(p)
}

func (s *staticFile) Seek(offset int64, whence int) (int64, error) {
	if s.reader == nil {
		return 0, os.ErrInvalid
	}
	return s.reader.Seek(offset, whence)
}

func (s *staticFile) Readdir(count int) ([]os.FileInfo, error) {
	if !s.fileInfo.dir {
		return nil, os.ErrInvalid
	}
	var fis []os.FileInfo

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

func (s *staticFile) Stat() (os.FileInfo, error) { return s.fileInfo, nil }

type staticFileInfo struct {
	basename string
	size     int64
	dir      bool
	modtime  time.Time
	ents     []*staticFileInfo
}

var _ os.FileInfo = &staticFileInfo{}

func (s *staticFileInfo) Name() string { return s.basename }
func (s *staticFileInfo) Size() int64  { return s.size }
func (s *staticFileInfo) Mode() os.FileMode {
	if s.dir {
		return 0755 | os.ModeDir
	}
	return 0644
}
func (s *staticFileInfo) ModTime() time.Time { return s.modtime }
func (s *staticFileInfo) IsDir() bool        { return s.dir }
func (s *staticFileInfo) Sys() interface{}   { return nil }
