package service

import (
	"context"
	"go-crawler/internal/model"
	"go-crawler/internal/search"
)

type IndexingWriterImpl struct {
	PageRepositoryWriter PageRepositoryWriter
	Index                *search.Index
}

func NewIndexingWriter(pageRepositoryWriter PageRepositoryWriter, index *search.Index) *IndexingWriterImpl {
	return &IndexingWriterImpl{
		PageRepositoryWriter: pageRepositoryWriter,
		Index:                index,
	}
}

func (i *IndexingWriterImpl) CreatePage(ctx context.Context, page *model.Page) error {
	saved, err := i.PageRepositoryWriter.UpsertPage(ctx, page)
	if err != nil {
		return err
	}
	i.Index.AddDocument(search.Document{
		ID:   saved.ID,
		Text: saved.Title + " " + saved.TextContent,
	})
	return nil
}
