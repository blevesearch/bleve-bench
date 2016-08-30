#!/bin/sh

# set these in your ENV or here
#DATADIR="/Users/mschoch/Documents/research/lucene/data"

TMPDIR=$(mktemp -d) || { echo "Failed to create temp file"; exit 1; }
BASE=`dirname $0`

echo "TMPDIR $TMPDIR - BASE $BASE"

# create directory for output
OUTPUT=${TMPDIR}/output
mkdir -p ${OUTPUT}

# create directory for the commands we will build
CMDDIR=${TMPDIR}/cmd
mkdir -p ${CMDDIR}

# build commands we need (using your gopath)
go build -o ${CMDDIR}/bleve-blast ${BASE}/cmd/bleve-blast
go build -o ${CMDDIR}/bleve-query ${BASE}/cmd/bleve-query
go build -o ${CMDDIR}/bleve-analyzer ${BASE}/cmd/bleve-analyzer
go build -o ${CMDDIR}/bbaggregate ${BASE}/cmd/bbaggregate
go build -o ${CMDDIR}/bbrunner ${BASE}/cmd/bbrunner

# rewrite config file with the paths we want
sed -e "s:OUTPUT:${OUTPUT}:g" $BASE/scripts/compare-config.json > ${TMPDIR}/config.json
sed -i "" -e "s:CONFDIR:${BASE}/configs:g" ${TMPDIR}/config.json
sed -i "" -e "s:DATADIR:${DATADIR}:g" ${TMPDIR}/config.json
sed -i "" -e "s:CMDDIR:${CMDDIR}:g" ${TMPDIR}/config.json

# prepare output files for aggregated data
mkdir -p ${OUTPUT}/indexing/
echo 'date,moss-store' > ${OUTPUT}/indexing/avg_mb_per_second.json
mkdir -p ${OUTPUT}/querying/
echo 'date,moss-store' > ${OUTPUT}/querying/term-query-hi.json
echo 'date,moss-store' > ${OUTPUT}/querying/term-query-low.json
mkdir -p ${OUTPUT}/analyzing/
echo 'date,moss-store' > ${OUTPUT}/analyzing/analyzers.json


# run it
echo "running bbrunner against current GOPATH"
${CMDDIR}/bbrunner -config ${TMPDIR}/config.json

# checkout everything from master
echo "getting blevesearch master"
MASTER=${TMPDIR}/master
mkdir -p ${MASTER}
GOPATH=${MASTER}
go get github.com/blevesearch/bleve-bench/...

# create directory for output master
OUTPUTM=${TMPDIR}/output-master
mkdir -p ${OUTPUTM}

# create updated config file pointing to master binaries
sed -e "s:OUTPUT:${OUTPUTM}:g" $BASE/scripts/compare-config.json > ${TMPDIR}/config-master.json
sed -i "" -e "s:CONFDIR:${BASE}/configs:g" ${TMPDIR}/config-master.json
sed -i "" -e "s:DATADIR:${DATADIR}:g" ${TMPDIR}/config-master.json
sed -i "" -e "s:CMDDIR:${MASTER}/bin:g" ${TMPDIR}/config-master.json

# prepare output files for aggregated data
mkdir -p ${OUTPUTM}/indexing/
echo 'date,moss-store' > ${OUTPUTM}/indexing/avg_mb_per_second.json
mkdir -p ${OUTPUTM}/querying/
echo 'date,moss-store' > ${OUTPUTM}/querying/term-query-hi.json
echo 'date,moss-store' > ${OUTPUTM}/querying/term-query-low.json
mkdir -p ${OUTPUTM}/analyzing/
echo 'date,moss-store' > ${OUTPUTM}/analyzing/analyzers.json

# run it again
echo "running bbrunner against master"
${MASTER}/bin/bbrunner -config ${TMPDIR}/config-master.json

###
# summarize the results

echo "stat\t\t\t\tGOPATH\t\tmaster\t\tchange"

IDX_MASTER=`tail -1 ${OUTPUTM}/indexing/avg_mb_per_second.json | cut -d , -f 2`
IDX_GOPATH=`tail -1 ${OUTPUT}/indexing/avg_mb_per_second.json | cut -d , -f 2`
IDX_CHANGE=`awk -v t1="${IDX_MASTER}" -v t2="${IDX_GOPATH}" 'BEGIN{printf "%.2f", (t2-t1)/t1 * 100}'`
echo "avg_mb_per_second\t\t${IDX_MASTER}\t${IDX_GOPATH}\t${IDX_CHANGE}%"

QHI_MASTER=`tail -1 ${OUTPUTM}/querying/term-query-hi.json | cut -d , -f 2`
QHI_GOPATH=`tail -1 ${OUTPUT}/querying/term-query-hi.json | cut -d , -f 2`
QHI_CHANGE=`awk -v t1="${QHI_MASTER}" -v t2="${QHI_GOPATH}" 'BEGIN{printf "%.2f", (t2-t1)/t1 * 100}'`
echo "term-query-hi\t\t\t${QHI_MASTER}\t${QHI_GOPATH}\t${QHI_CHANGE}%"

QLO_MASTER=`tail -1 ${OUTPUTM}/querying/term-query-low.json | cut -d , -f 2`
QLO_GOPATH=`tail -1 ${OUTPUT}/querying/term-query-low.json | cut -d , -f 2`
QLO_CHANGE=`awk -v t1="${QLO_MASTER}" -v t2="${QLO_GOPATH}" 'BEGIN{printf "%.2f", (t2-t1)/t1 * 100}'`
echo "term-query-hi\t\t\t${QLO_MASTER}\t${QLO_GOPATH}\t${QLO_CHANGE}%"
