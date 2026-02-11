package templates

import "embed"

// StaticFS contains technology-agnostic files that are copied as-is.
//
//go:embed static/*
var StaticFS embed.FS

// SharedFS contains Go template files (.tmpl) rendered with project config.
//
//go:embed shared/*
var SharedFS embed.FS

// VariantsFS contains database and auth variant implementations.
//
//go:embed variants/*
var VariantsFS embed.FS
