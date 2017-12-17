#!/bin/sh
grep -Rn 'package \(\w\+\)$' types/*.pb.go | cut -d':' -f1 | while read F;do backup="$F.bak";sed "s/package \([a-zA-Z]\{1,\}\)$/\/\/nolint: gas<NEWLINE_HEREXXXX>&/1" "$F" | awk -F"<NEWLINE_HEREXXXX>" '{ if (NF >= 2) { print $1"\n"$2 } else { print $1 } }' > "$backup";mv "$backup" "$F";done
