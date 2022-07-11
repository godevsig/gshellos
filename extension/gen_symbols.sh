#!/bin/bash

pkg=$1
shift

basepkg=`basename $pkg`
../cmd/extract/extract -name extension -tag $basepkg $pkg

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
                ../cmd/extract/extract -name extension -tag ${basepkg}msg $extrapkg
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