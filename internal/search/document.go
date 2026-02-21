package search

// Document is input for building the index: one page with ID and text to tokenize.
type Document struct {
	ID   int
	Text string
}
