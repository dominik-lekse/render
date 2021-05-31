// +build go1.16

package render

import (
	"fmt"
	"html/template"
	"io"
	"io/fs"
	"os"
	"path/filepath"
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

	r.templatesLk.Lock()
	r.templates = tmpTemplates
	r.templatesLk.Unlock()

	return nil
}

func (a *AssetFS) Open(name string) (fs.File, error) {

	// a.AssetNames
	// TODO implement assetFS
	panic("implement me")
}
