package search

import (
	"math"
	"sort"
	"strings"
	"sync"
	"unicode"
)

type Index struct {
	mu        sync.RWMutex
	entries   map[string]map[int]int // term -> document ID -> count
	docLen    map[int]int            // document ID -> total token count
	totalDocs int                    // number of documents indexed
}

func NewIndex() *Index {
	return &Index{
		entries: make(map[string]map[int]int),
		docLen:  make(map[int]int),
	}
}

func (i *Index) Tokenize(text string) []string {
	var tokens []string
	f := func(c rune) bool {
		return unicode.IsSpace(c) || unicode.IsPunct(c)
	}
	for _, token := range strings.FieldsFunc(text, f) {
		t := strings.ToLower(strings.TrimSpace(token))
		if t != "" && len(t) >= 2 {
			tokens = append(tokens, t)
		}
	}
	return tokens
}

func (i *Index) BuildFromDocuments(documents []Document) {
	i.mu.Lock()
	defer i.mu.Unlock()

	i.entries = make(map[string]map[int]int)
	i.docLen = make(map[int]int)
	i.totalDocs = len(documents)

	for _, doc := range documents {
		terms := i.Tokenize(doc.Text)
		i.docLen[doc.ID] = len(terms)
		for _, term := range terms {
			if _, ok := i.entries[term]; !ok {
				i.entries[term] = make(map[int]int)
			}
			i.entries[term][doc.ID]++
		}
	}
}

type SearchResult struct {
	DocumentID int
	Score      float64
}

func (i *Index) Search(query string) []SearchResult {
	i.mu.RLock()
	defer i.mu.RUnlock()

	terms := i.Tokenize(query)
	if len(terms) == 0 || i.totalDocs == 0 {
		return nil
	}

	scores := map[int]float64{}
	N := float64(i.totalDocs)

	for _, term := range terms {
		postings := i.entries[term]
		if len(postings) == 0 {
			continue
		}
		df := float64(len(postings))
		idf := math.Log((N+1)/(df+1)) + 1

		for docID, count := range postings {
			dl := i.docLen[docID]
			if dl == 0 {
				continue
			}
			tf := float64(count) / float64(dl)
			scores[docID] += tf * idf
		}
	}

	results := make([]SearchResult, 0, len(scores))
	for docID, score := range scores {
		results = append(results, SearchResult{
			DocumentID: docID,
			Score:      score,
		})
	}
	sort.Slice(results, func(a, b int) bool {
		if results[a].Score == results[b].Score {
			return results[a].DocumentID < results[b].DocumentID
		}
		return results[a].Score > results[b].Score
	})
	return results
}

func (i *Index) AddDocument(document Document) {
	i.mu.Lock()
	defer i.mu.Unlock()
	_, isReplace := i.docLen[document.ID]
	if isReplace {
		delete(i.docLen, document.ID)
		for _, postings := range i.entries {
			delete(postings, document.ID)
		}
	}
	if !isReplace {
		i.totalDocs++
	}
	terms := i.Tokenize(document.Text)
	i.docLen[document.ID] = len(terms)
	for _, term := range terms {
		if _, ok := i.entries[term]; !ok {
			i.entries[term] = make(map[int]int)
		}
		i.entries[term][document.ID]++
	}

}
