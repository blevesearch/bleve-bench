#!/bin/sh

HIGH_HIGH=0.9
HIGH_MED=0.09
LOW_MED=0.002
LOW_LOW=0.0005

tmpfile=$(mktemp /tmp/td-$1.XXXXXX)
echo "tmpfile for collation is $tmpfile"


echo "Processing $1 file..."
bleve_dump -dictionary $2 -index $1 | awk '{print $1,$3}' >> $tmpfile

# sort by col1
echo "sorting into $tmpfile.sorted"
sort -t ' ' -k1,1 $tmpfile > ${tmpfile}.sorted

HH=$(bc <<<"scale=0;$HIGH_HIGH*$3")
HM=$(bc <<<"scale=0;$HIGH_MED*$3")
LM=$(bc <<<"scale=0;$LOW_MED*$3")
LL=$(bc <<<"scale=0;$LOW_LOW*$3")

# find hi,med,low terms
echo "getting high ($HM <= x < $HH) - ${tmpfile}.hi"
awk -v hh="$HH" -v hm="$HM" -F ' ' '{
  if(($2 < hh) && ($2 >= hm)) {
    printf("%s %d\n",$1,$2)
  }
}' ${tmpfile}.sorted > ${tmpfile}.hi

echo "getting med ($LM <= x < $HM) - ${tmpfile}.med"
awk -v hm="$HM" -v lm="$LM" -F ' ' '{
  if(($2 < hm) && ($2 >= lm)) {
    printf("%s %d\n",$1,$2)
  }
}' ${tmpfile}.sorted > ${tmpfile}.med

echo "getting low ($LL <= x < $LM) - ${tmpfile}.low"
awk -v lm="$LM" -v ll="$LL" -F ' ' '{
  if(($2 < lm) && ($2 >= ll)) {
    printf("%s %d\n",$1,$2)
  }
}' ${tmpfile}.sorted > ${tmpfile}.low

# check too high
awk -v hh="$HH" -F ' ' 'BEGIN{
  count = 0
}{
  if($2 >= hh) {
    count++
  }
}
END{
  printf("too high: %d\n", count)
}' ${tmpfile}.sorted

# check too low
awk -v ll="$LL" -F ' ' 'BEGIN{
  count = 0
}{
  if($2 < ll) {
    count++
  }
}
END{
  printf("too low: %d\n", count)
}' ${tmpfile}.sorted
