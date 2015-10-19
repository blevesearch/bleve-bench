d=$(cd $(dirname $0) ; pwd -P)
cd $d/bleve-bench
git checkout master
git pull
cd cmd/bleve-blast
go build -tags 'boltdb'
cd ../bleve-query
go build -tags 'boltdb'
python ../../scripts/daily.py -f $d/test_config -d $d/data
scp -i $d/nimish2.pem $d/data/index/data/* ubuntu@46.137.228.120:~/perf/data/index/data
scp -i $d/nimish2.pem $d/data/query/data/* ubuntu@46.137.228.120:~/perf/data/query/data
