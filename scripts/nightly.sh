#!/bin/sh

# set these for your environment
#export LEVELDB=""
#export ROCKSDB=""
#export FORESTDB=""

export WORKDIR="/tmp"

# create temp workspace
export TMPDIR=`mktemp -d -q ${WORKDIR}/nightly-bleve-bench.XXXXXX`
if [ $? -ne 0 ]; then
	echo "$0: Can't create temp file, exiting..."
	exit 1
fi
echo "TMPDIR is ${TMPDIR}"

# set up gopath
export GOPATH="${TMPDIR}/go"
mkdir -p ${GOPATH}
echo "GOPATH is ${GOPATH}"

# set up leveldb
export CGO_CFLAGS="-I${LEVELDB}/include/"
export CGO_LDFLAGS="-L${LEVELDB}"
go get github.com/syndtr/goleveldb/leveldb
echo "Installed goleveldb"

# set up rocksdb
export CGO_CFLAGS="-I${ROCKSDB}/include/"
export CGO_LDFLAGS="-L${ROCKSDB} -lrocksdb -lstdc++ -lm -lz -lbz2 -lsnappy"
go get github.com/tecbot/gorocksdb
echo "Installed gorocksdb"

# setup forestdb
export CGO_CFLAGS="-I${FORESTDB}/include/"
export CGO_LDFLAGS="-L${FORESTDB}"
go get github.com/couchbase/goforestdb
echo "Installed goforestdb"

# installing bleve-bench
go get -tags 'leveldb rocksdb forestdb' github.com/blevesearch/bleve-bench/...

# cleanup
echo "Cleaning up ${TMPDIR}"
rm -r ${TMPDIR}