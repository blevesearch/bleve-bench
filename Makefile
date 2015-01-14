tmp:
	mkdir -p tmp

tmp/enwiki-20070527-pages-articles.xml.bz2: tmp
	curl -o tmp/enwiki-20070527-pages-articles.xml.bz2 http://people.apache.org/~gsingers/wikipedia/enwiki-20070527-pages-articles.xml.bz2

linefile:
	go build linefile.go

wikilinefile: linefile
	./linefile tmp/enwiki-20070527-pages-articles.xml.bz2 tmp/enwiki.txt tmp/categories-enwiki.txt

clean:
	rm -rf tmp