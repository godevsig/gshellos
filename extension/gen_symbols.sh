#!/bin/bash

test "$1" = "-tag" && tag=$2 && shift 2
pkg=$1 && shift
basepkg=`basename $pkg`
: ${tag:=$basepkg}

file=`echo $pkg | tr ./ _-`.go
../cmd/extract/extract -name extension -tag $tag $pkg
sed -i 's/Symbols\[/BuiltinSymbols\[/g' $file

while test $# != 0; do
        case $1 in
        -fixlog)
                sed -i 's/logLogger/log.Logger/' $file
                shift
                ;;
        -extramsg)
                extrapkg=$pkg/$basepkg
                extrafile=`echo $extrapkg | tr ./ _-`.go
                ../cmd/extract/extract -name extension -tag ${tag}msg $extrapkg
                sed -i 's/Symbols\[/BuiltinSymbols\[/g' $extrafile
                sed -n '/func init/,$p' $extrafile >> $file
                shift
                ;;
        *)
                shift
        esac
done
