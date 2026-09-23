package router

import (
	"bytes"
	"html"
	"strings"

	"github.com/QuantumNous/new-api/common"
	htmlnode "golang.org/x/net/html"
	"golang.org/x/net/html/atom"
)

const (
	siteNameMarker      = "__NEWAPI_MAO_SITE_NAME__"
	siteDescriptionSlot = "__NEWAPI_MAO_SITE_DESCRIPTION_SLOT__"
)

func prepareSiteIndexPage(page []byte) []byte {
	document, err := htmlnode.Parse(bytes.NewReader(page))
	if err != nil {
		return page
	}

	head := findHTMLHead(document)
	if head == nil {
		return page
	}

	for child := head.FirstChild; child != nil; {
		next := child.NextSibling
		if isDynamicSiteMetadataNode(child) {
			head.RemoveChild(child)
		}
		child = next
	}

	metadata := []*htmlnode.Node{
		{
			Type:     htmlnode.ElementNode,
			DataAtom: atom.Title,
			Data:     "title",
			FirstChild: &htmlnode.Node{
				Type: htmlnode.TextNode,
				Data: siteNameMarker,
			},
		},
		newSiteMeta("name", "title", siteNameMarker),
		newSiteMeta("property", "og:title", siteNameMarker),
		newSiteMeta("property", "og:site_name", siteNameMarker),
		{Type: htmlnode.CommentNode, Data: siteDescriptionSlot},
	}
	for i := len(metadata) - 1; i >= 0; i-- {
		head.InsertBefore(metadata[i], head.FirstChild)
	}

	var output bytes.Buffer
	if err := htmlnode.Render(&output, document); err != nil {
		return page
	}
	return output.Bytes()
}

func findHTMLHead(node *htmlnode.Node) *htmlnode.Node {
	if node.Type == htmlnode.ElementNode && node.Data == "head" {
		return node
	}
	for child := node.FirstChild; child != nil; child = child.NextSibling {
		if head := findHTMLHead(child); head != nil {
			return head
		}
	}
	return nil
}

func isDynamicSiteMetadataNode(node *htmlnode.Node) bool {
	if node.Type != htmlnode.ElementNode {
		return false
	}
	if node.Data == "title" {
		return true
	}
	if node.Data != "meta" {
		return false
	}
	for _, attribute := range node.Attr {
		if strings.EqualFold(attribute.Key, "name") &&
			(strings.EqualFold(attribute.Val, "title") || strings.EqualFold(attribute.Val, "description")) {
			return true
		}
		if strings.EqualFold(attribute.Key, "property") &&
			(strings.EqualFold(attribute.Val, "og:title") ||
				strings.EqualFold(attribute.Val, "og:site_name") ||
				strings.EqualFold(attribute.Val, "og:description")) {
			return true
		}
	}
	return false
}

func newSiteMeta(attribute, value, content string) *htmlnode.Node {
	return &htmlnode.Node{
		Type:     htmlnode.ElementNode,
		DataAtom: atom.Meta,
		Data:     "meta",
		Attr: []htmlnode.Attribute{
			{Key: attribute, Val: value},
			{Key: "content", Val: content},
		},
	}
}

func renderSiteIndexPage(page []byte) []byte {
	common.OptionMapRWMutex.RLock()
	systemName := common.OptionMap["SystemName"]
	description := common.OptionMap["SystemDescription"]
	common.OptionMapRWMutex.RUnlock()
	if systemName == "" {
		systemName = "NewAPI-Mao"
	}

	descriptionMetadata := ""
	if description != "" {
		escapedDescription := html.EscapeString(description)
		descriptionMetadata = `<meta name="description" content="` + escapedDescription + `"><meta property="og:description" content="` + escapedDescription + `">`
	}

	replacer := strings.NewReplacer(
		siteNameMarker, html.EscapeString(systemName),
		"<!--"+siteDescriptionSlot+"-->", descriptionMetadata,
	)
	return []byte(replacer.Replace(string(page)))
}
