package utils

import (
	"html"

	"github.com/microcosm-cc/bluemonday"
)

var htmlPolicy = func() *bluemonday.Policy {
	p := bluemonday.UGCPolicy()
	p.RequireParseableURLs(true)
	p.AllowURLSchemes("http", "https")
	p.RequireNoFollowOnLinks(true)
	return p
}()

func SafeHTML(s string) string {
	if s == "" {
		return s
	}
	return htmlPolicy.Sanitize(s)
}

func SafeText(s string) string {
	if s == "" {
		return s
	}
	return html.EscapeString(s)
}
