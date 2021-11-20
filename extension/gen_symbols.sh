#!/bin/bash

pkg=$1
shift

basepkg=`basename $pkg`
../cmd/extract/extract -name extension -tag $basepkg $pkg

file=`echo $pkg | tr ./ _-`.go
while test $# != 0; do
        case $1 in
                -fixlog)
                sed -i 's/logLogger/log.Logger/' $file
                shift
                ;;
                -extramsg)
                extrapkg=$pkg/$basepkg
                extrafile=`echo $extrapkg | tr ./ _-`.go
                ../cmd/extract/extract -name extension -tag ${basepkg}msg $extrapkg
                sed -n '/func init/,$p' $extrafile >> $file
                shift
                ;;
                *)
                shift
        esac
done

gopls format -w $file