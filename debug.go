package mel

import (
	"bytes"
	"html/template"
	"log"
)

func init() {
	log.SetFlags(0)
}

// IsDebugging returns true if the framework is running in debug mode.
// Use SetMode(gin.Release) to switch to disable the debug mode.
func IsDebugging() bool {
	return true
}

func debugPrintRoute(httpMethod, absolutePath string, handlers []Handler) {
	if IsDebugging() {
		nuHandlers := len(handlers)
		handlerName := nameOfFunction(handlers[nuHandlers-1])
		debugPrint("%-6s %-25s --> %s (%d handlers)\n", httpMethod, absolutePath, handlerName, nuHandlers)
	}
}

func debugPrintLoadTemplate(tmpl *template.Template) {
	if IsDebugging() {
		var buf bytes.Buffer
		for _, tmpl := range tmpl.Templates() {
			buf.WriteString("\t- ")
			buf.WriteString(tmpl.Name())
			buf.WriteString("\n")
		}
		debugPrint("Loaded HTML Templates (%d): \n%s\n", len(tmpl.Templates()), buf.String())
	}
}

func debugPrint(format string, v ...interface{}) {
	if IsDebugging() {
		log.Printf("[GIN-debug] " + format, v...)
	}
}

func debugPrintWARNINGNew() {
	debugPrint(`[WARNING] Running in "debug" mode. Switch to "release" mode in production.
 - using env:	export GIN_MODE=release
 - using code:	gin.SetMode(gin.ReleaseMode)

`)
}

func debugPrintWARNINGSetHTMLTemplate() {
	debugPrint(`[WARNING] Since SetHTMLTemplate() is NOT thread-safe. It should only be called
at initialization. ie. before any route is registered or the router is listening in a socket:

	router := gin.Default()
	router.SetHTMLTemplate(template) // << good place

`)
}

func debugPrintError(err error) {
	if err != nil {
		debugPrint("[ERROR] %v\n", err)
	}
}
