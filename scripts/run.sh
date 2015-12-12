export GOPATH=/data/gopath
d=$(cd $(dirname $0) ; pwd -P)
cd $d/bleve-bench
git checkout master
git pull
/data/go1/bin/go get -u github.com/blevesearch/bleve 
cd cmd/bleve-blast
/data/go1/bin/go build -tags 'boltdb forestdb rocksdb'
cd ../bleve-query
/data/go1/bin/go build -tags 'boltdb forestdb rocksdb'
python ../../scripts/daily.py -f $d/test_config -d $d/data
scp -i $d/nimish2.pem $d/data/index/data/* ubuntu@46.137.228.120:~/perf/data/index/data
scp -i $d/nimish2.pem $d/data/query/data/* ubuntu@46.137.228.120:~/perf/data/query/data
#scp -i $d/nimish2.pem $d/bleve-bench/configs/* ubuntu@46.137.228.120:~/perf/configs
