package blevebench

import (
	"github.com/blevesearch/bleve"
)

func BuildArticleMapping() *bleve.IndexMapping {

	// a generic reusable mapping for english text
	standardJustIndexed := bleve.NewTextFieldMapping()
	standardJustIndexed.Store = false
	standardJustIndexed.IncludeInAll = false
	standardJustIndexed.IncludeTermVectors = false
	standardJustIndexed.Analyzer = "standard"

	keywordJustIndexed := bleve.NewTextFieldMapping()
	keywordJustIndexed.Store = false
	keywordJustIndexed.IncludeInAll = false
	keywordJustIndexed.IncludeTermVectors = false
	keywordJustIndexed.Analyzer = "keyword"

	articleMapping := bleve.NewDocumentMapping()

	// title
	articleMapping.AddFieldMappingsAt("title",
		keywordJustIndexed)

	// text
	articleMapping.AddFieldMappingsAt("text",
		standardJustIndexed)

	indexMapping := bleve.NewIndexMapping()
	indexMapping.DefaultMapping = articleMapping
	indexMapping.DefaultAnalyzer = "standard"

	return indexMapping
}
