package main

import (
	"bytes"
	"html/template"
	"net/url"
	"strings"
)

func isValidURL(text string) bool {
	_, err := url.ParseRequestURI(text)
	if err != nil {
		return false
	}

	u, err := url.Parse(text)
	if err != nil || u.Scheme == "" || u.Host == "" {
		return false
	}

	return true
}

func HTMLLink(title string, url string) string {
	tmpl, err := template.New("link").Parse(`<a href="{{.URL}}" title="{{.Title}}">{{.Title}}</a>`)
	if err != nil {
		return ""
	}

	repalcer := strings.NewReplacer("&amp;", "&", "&lt;", "<", "&gt;", ">", "&quot;", "\"", "&apos;", "'")

	data := struct {
		Title template.HTML
		URL   template.URL
	}{
		Title: template.HTML(repalcer.Replace(title)),
		URL:   template.URL(url),
	}

	var buffer bytes.Buffer
	err = tmpl.Execute(&buffer, data)
	if err != nil {
		return ""
	}

	return buffer.String()
}
