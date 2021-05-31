package render

// AssetFS
type AssetFS struct {
	// Asset function to use in place of directory. Defaults to nil.
	Asset func(name string) ([]byte, error)
	// AssetNames function to use in place of directory. Defaults to nil.
	AssetNames func() []string
}

var _ FS = &AssetFS{}
