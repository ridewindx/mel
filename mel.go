package mel

type Handler func(*Context)

type Mel struct {
	Router
	handlers []Handler
}
