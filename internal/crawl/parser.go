package crawl

import (
	"bytes"
	"net/url"
	"strings"

	"golang.org/x/net/html"
)

type ParsedPage struct {
	Title string
	Links []string
}

func ParsePage(baseURL string, body []byte) (*ParsedPage, error) {

	doc, err := html.Parse(bytes.NewReader(body))
	if err != nil {
		return nil, err
	}

	base, err := url.Parse(baseURL)
	if err != nil {
		return nil, err
	}

	var (
		title string
		links []string
	)

	var walker func(*html.Node)
	walker = func(n *html.Node) {

		// Title extraction
		if n.Type == html.ElementNode && n.Data == "title" {
			if n.FirstChild != nil {
				title = strings.TrimSpace(n.FirstChild.Data)
			}
		}

		// Link extraction
		if n.Type == html.ElementNode && n.Data == "a" {
			for _, attr := range n.Attr {
				if attr.Key == "href" {
					href := strings.TrimSpace(attr.Val)
					if href == "" {
						continue
					}

					ref, err := url.Parse(href)
					if err != nil {
						continue
					}

					absolute := base.ResolveReference(ref)

					if absolute.Scheme == "http" || absolute.Scheme == "https" {
						links = append(links, absolute.String())
					}
				}
			}
		}

		for c := n.FirstChild; c != nil; c = c.NextSibling {
			walker(c)
		}
	}

	walker(doc)

	return &ParsedPage{
		Title: title,
		Links: links,
	}, nil
}
