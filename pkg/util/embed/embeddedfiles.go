package embed

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"embed"
	"path"
)

func ReadDirRecursive(fs embed.FS, dir string) [][]byte {
	result := make([][]byte, 0)
	entries, err := fs.ReadDir(dir)
	if err != nil {
		return nil
	}
	for _, v := range entries {
		if v.IsDir() {
			result = append(result, ReadDirRecursive(fs, path.Join(dir, v.Name()))...)
		} else {
			bytes, err := fs.ReadFile(path.Join(dir, v.Name()))
			if err != nil {
				result = append(result, nil)
			}
			result = append(result, bytes)
		}
	}

	return result
}
