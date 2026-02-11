package generator

import "strings"

// sourceModuleName is the module name used in the template source files.
const sourceModuleName = "go-api-template"

// rewriteImports replaces the source module name with the target module name
// in Go source file content.
func rewriteImports(content string, targetModule string) string {
	return strings.ReplaceAll(content, sourceModuleName, targetModule)
}
