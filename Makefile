bleve-bench: index.go
	go build -tags 'leveldb debug'

tmp:
	mkdir -p tmp

tmp/enwiki-20070527-pages-articles.xml.bz2: tmp
	curl -o tmp/enwiki-20070527-pages-articles.xml.bz2 http://snapshots.couchbase.com.s3.amazonaws.com/bleve-bench/enwiki-20070527-pages-articles.xml.bz2

linefile:
	go build linefile.go

wikilinefile: linefile tmp/enwiki-20070527-pages-articles.xml.bz2
	./linefile tmp/enwiki-20070527-pages-articles.xml.bz2 tmp/enwiki.txt tmp/categories-enwiki.txt

clean:
	rm -rf tmp
