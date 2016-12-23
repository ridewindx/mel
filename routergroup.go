package mel

type RouterGroup struct {
	basePath string
	handlers []Handler
	mel *Mel
}

func (group *RouterGroup) Use(middleware ...Handler) {
	group.handlers = append(group.handlers, middleware...)
}

// Group creates a new router group.
// You should add all the routes that have common middlwares or the same path prefix.
func (group *RouterGroup) Group(relativePath string, handlers ...Handler) *RouterGroup {
	return &RouterGroup{
		basePath: joinPaths(group.basePath, relativePath),
		handlers: group.combineHandlers(handlers),
		mel: group.mel,
	}
}

func (group *RouterGroup) combineHandlers(handlers []Handler) []Handler {
	finalSize := len(group.handlers) + len(handlers)
	if finalSize >= int(abortIndex) {
		panic("too many handlers")
	}
	mergedHandlers := make([]Handler, finalSize)
	copy(mergedHandlers, group.handlers)
	copy(mergedHandlers[len(group.handlers):], handlers)
	return mergedHandlers
}

func (group *RouterGroup) calculateAbsolutePath(relativePath string) string {
	return joinPaths(group.basePath, relativePath)
}

