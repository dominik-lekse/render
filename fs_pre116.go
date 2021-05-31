// +build !go1.16

package render

import (
	"fmt"
	"html/template"
	"io/ioutil"
	"net/http"
	"os"
	"path"
	"path/filepath"
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

	r.templatesLk.Lock()
	r.templates = tmpTemplates
	r.templatesLk.Unlock()

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

// TODO implement AssetFS.Open with http.File
func (a *AssetFS) Open(name string) (http.File, error) {

	// a.AssetNames
	// TODO implement assetFS
	panic("implement me")
}
