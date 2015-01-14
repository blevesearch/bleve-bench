package main

import (
	"log"
	"os"
	"os/exec"
	"testing"

	"github.com/blevesearch/bleve"
)

var wikiReader *WikiReader
var articleMapping *bleve.IndexMapping

func TestMain(m *testing.M) {
	var err error
	wikiReader, err = NewWikiReader("tmp/enwiki.txt")
	if err != nil {
		log.Fatal(err)
	}
	articleMapping = buildArticleMapping()

	err = createWikiIndex("empty.bleve", "", 0)
	if err != nil {
		log.Fatal(err)
	}

	err = createWikiIndex("1k.bleve", "empty.bleve", 1000)
	if err != nil {
		log.Fatal(err)
	}

	err = createWikiIndex("10k.bleve", "1k.bleve", 9000)
	if err != nil {
		log.Fatal(err)
	}

	err = createWikiIndex("100k.bleve", "10k.bleve", 90000)
	if err != nil {
		log.Fatal(err)
	}

	rv := m.Run()
	os.RemoveAll("empty.bleve")
	os.RemoveAll("1k.bleve")
	os.RemoveAll("10k.bleve")
	os.RemoveAll("100k.bleve")
	os.Exit(rv)
}

func createWikiIndex(name, source string, count int) error {
	log.Printf("Creating Index: %s", name)
	var index bleve.Index
	if source != "" {
		log.Printf("\tcopying: %s", source)
		err := copyBleve(source, name)
		if err != nil {
			log.Fatal(err)
		}
		index, err = bleve.Open(name)
		if err != nil {
			log.Fatal(err)
		}
	} else {
		log.Printf("\tnew")
		var err error
		index, err = bleve.New(name, articleMapping)
		if err != nil {
			log.Fatal(err)
		}
	}
	log.Printf("\tadding %d docs", count)
	benchmarkIndex(wikiReader, index, count, 1000)
	log.Printf("\tdone.")
	index.Close()
	return nil
}

func copyBleve(src, dst string) error {
	cmd := exec.Command("cp", "-r", src, dst)
	out, err := cmd.Output()
	if err != nil {
		log.Printf("copy error: %s", out)
		return err
	}
	return nil
}

func BenchmarkWiki(b *testing.B) {
	benchmarkIndexWiki(b, "empty.bleve", 1, 0)
}

func BenchmarkWikiAfter1k(b *testing.B) {
	benchmarkIndexWiki(b, "1k.bleve", 1, 0)
}

func BenchmarkWikiAfter10k(b *testing.B) {
	benchmarkIndexWiki(b, "10k.bleve", 1, 0)
}

func BenchmarkWikiAfter100k(b *testing.B) {
	benchmarkIndexWiki(b, "100k.bleve", 1, 0)
}

func benchmarkIndexWiki(b *testing.B, source string, count, batch int) {

	err := copyBleve(source, "bench.bleve")
	if err != nil {
		b.Fatal(err)
	}
	index, err := bleve.Open("bench.bleve")
	if err != nil {
		b.Fatal(err)
	}
	defer os.RemoveAll("bench.bleve")
	defer index.Close()

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		err = benchmarkIndex(wikiReader, index, count, batch)
		if err != nil {
			b.Fatalf("error indexing: %v", err)
		}
	}
}

func benchmarkIndex(wikiReader *WikiReader, index bleve.Index, count, batch int) error {
	if batch == 0 {
		for i := 0; i < count; i++ {
			a, err := wikiReader.Next()
			if err != nil {
				return err
			}
			err = index.Index(a.Title, a)
			if err != nil {
				return err
			}
		}
	} else {
		b := bleve.NewBatch()
		for i := 0; i < count; i++ {
			a, err := wikiReader.Next()
			if err != nil {
				return err
			}
			b.Index(a.Title, a)
			if b.Size() == batch {
				err = index.Batch(b)
				if err != nil {
					return err
				}
				b = bleve.NewBatch()
			}
		}
		// dont forget last batch
		if b.Size() > 0 {
			err := index.Batch(b)
			if err != nil {
				return err
			}
		}
	}
	return nil
}
