// +build go1.16

package embed

import "embed"

//go:embed fixtures/*/*.html fixtures/*/*.tmpl fixtures/*/*/*.tmpl fixtures/*/*.amber fixtures/*/*/*.amber
var Fixtures embed.FS
