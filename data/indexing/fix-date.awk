BEGIN {
  FS = ",";
  OFS=","
}
;
{
  if(NR!=1) {
    $1=sprintf("%s 00:00:00",$1)
  }
  print $0
}
