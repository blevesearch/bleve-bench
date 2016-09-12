#!/bin/sh

# $1 file to process
# $2 name of config to add

awk -v newcol="$2" -F ' ' '
BEGIN {
  FS = ",";
  OFS=","
}
;
{
  if(NR==1) {
    print $0,newcol
  } else {
    printf("%s,,\n",$0)
  }
}
' $1
