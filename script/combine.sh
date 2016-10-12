id=0
while read -r line || [[ -n $line ]]; do
	if [[ -n $line ]]; then
		id=$((id+1))
		echo $line | dot -Tpng -o "trie${id}.png";
	fi
done

s=
for ((j=1;j<=id;j++))
do
	s="$s $(printf "trie${j}.png")"
done

# do combine
/usr/local/bin/gm convert +append $s trie.png

# cleanup
find . -name "trie[0-9].png" -exec rm '{}' +;

open "trie.png";
