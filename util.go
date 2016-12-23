package mel

import "path"

func lastChar(str string) uint8 {
	size := len(str)
	if size == 0 {
		panic("The length of the string can't be 0")
	}
	return str[size-1]
}

func joinPaths(absolutePath, relativePath string) string {
	if len(relativePath) == 0 {
		return absolutePath
	}

	finalPath := path.Join(absolutePath, relativePath)
	appendSlash := lastChar(relativePath) == '/' && lastChar(finalPath) != '/'
	if appendSlash {
		// do not leave out the suffix slash
		return finalPath + "/"
	}
	return finalPath
}

func assert(guard bool, text string) {
	if !guard {
		panic(text)
	}
}
