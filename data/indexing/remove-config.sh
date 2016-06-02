#!/bin/sh

# $1 file to process
# $2 name of config to remove

awk -v col="$2" -F ' ' '
BEGIN {
  FS = ",";
  OFS=","
}
;
{
  if(NR==1) {
    coli = 0
    for (i=1;i<=NF;i++) {
      if ($i == col) {
        coli = i
      }
    }
    if (coli==0) {
      printf("unable to find config named %s\n", col)
      exit 1
    }
    for (i=1;i<=NF;i++) {
      if (i == coli) {
        continue
      }
      if (i != 1) {
        printf("%s",OFS)
      }
      printf("%s",$i)
    }
    printf("\n")
  } else {
    for (i=1;i<=NF;i++) {
      if (i == ((coli-1)*2)) {
        continue
      }
      if (i == ((coli-1)*2)+1) {
        continue
      }
      if (i != 1) {
        printf("%s",OFS)
      }
      printf("%s",$i)
    }
    printf("\n")
  }
}
' $1
