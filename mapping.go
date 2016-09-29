package blevebench

import (
	"github.com/blevesearch/bleve"
	"github.com/blevesearch/bleve/mapping"
)

// BuildArticleMapping returns a mapping for indexing wikipedia articles
// in a manner similar to that done by lucene nightly benchmarks
func BuildArticleMapping() mapping.IndexMapping {

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

	// _all (disabled)
	disabledSection := bleve.NewDocumentDisabledMapping()
	articleMapping.AddSubDocumentMapping("_all", disabledSection)

	indexMapping := bleve.NewIndexMapping()
	indexMapping.DefaultMapping = articleMapping
	indexMapping.DefaultAnalyzer = "standard"

	return indexMapping
}
