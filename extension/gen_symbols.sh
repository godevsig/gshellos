#!/bin/bash

test "$1" = "-tag" && tag=$2 && shift 2
pkg=$1 && shift
basepkg=`basename $pkg`
: ${tag:=$basepkg}

../cmd/extract/extract -name extension -tag $tag $pkg

file=`echo $pkg | tr ./ _-`.go
mv $file $file.raw
while test $# != 0; do
        case $1 in
        -fixlog)
                sed -i 's/logLogger/log.Logger/' $file.raw
                shift
                ;;
        -extramsg)
                extrapkg=$pkg/$basepkg
                extrafile=`echo $extrapkg | tr ./ _-`.go
                ../cmd/extract/extract -name extension -tag ${tag}msg $extrapkg
                sed -n '/func init/,$p' $extrafile >> $file.raw
                shift
                ;;
        *)
                shift
        esac
done

head -n 2 $file.raw > $file
tail -n +2 $file.raw > fmt-$file
gopls format -w fmt-$file
cat fmt-$file >> $file
rm -f $file.raw fmt-$file