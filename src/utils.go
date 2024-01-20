package main

import (
	"fmt"
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

func markdownLink(text string, link string) string {
	repalcer := strings.NewReplacer("[", "【", "]", "】", "(", "（", ")", "）")
	return fmt.Sprintf("[%s](%s)", repalcer.Replace(text), link)
}
