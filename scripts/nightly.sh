#!/bin/sh

# set these for your environment
#export LEVELDB=""
#export ROCKSDB=""

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
go get github.com/jmhodges/levigo
echo "Installed levigo"

# set up rocksdb
export CGO_CFLAGS="-I${ROCKSDB}/include/"
export CGO_LDFLAGS="-L${ROCKSDB} -lrocksdb -lstdc++ -lm -lz -lbz2 -lsnappy"
go get github.com/tecbot/gorocksdb
echo "Installed gorocksdb"

# installing bleve-bench
export CGO_CFLAGS=""
export CGO_LDFLAGS=""
go get -tags 'leveldb rocksdb' github.com/blevesearch/bleve-bench/...

# cleanup
echo "Cleaning up ${TMPDIR}"
rm -r ${TMPDIR}
