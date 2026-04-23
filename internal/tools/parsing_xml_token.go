package tools

import (
	"regexp"
	"strings"
)

// xmlOpenTagPattern は <tag_name> 形式の開始タグを検出する正規表現
var xmlOpenTagPattern = regexp.MustCompile(`<([a-zA-Z_][\w-]*)>`)

type xmlOpenTagToken struct {
	openStart    int
	contentStart int
	tagName      string
}

func findNextXMLOpenTag(text string, searchFrom int) (xmlOpenTagToken, bool) {
	loc := xmlOpenTagPattern.FindStringSubmatchIndex(text[searchFrom:])
	if loc == nil {
		return xmlOpenTagToken{}, false
	}
	return xmlOpenTagToken{
		openStart:    searchFrom + loc[0],
		contentStart: searchFrom + loc[1],
		tagName:      text[searchFrom+loc[2] : searchFrom+loc[3]],
	}, true
}

func xmlCloseTag(tagName string) string {
	return "</" + tagName + ">"
}

func findXMLCloseTagIndex(text string, contentStart int, tagName string) int {
	return strings.Index(text[contentStart:], xmlCloseTag(tagName))
}
