#!/bin/bash
### This is just an example of what it might look like
### Implement rules to meet as much of https://google.github.io/styleguide/shellguide.html as possible
exit 0
find . -type f | while read -r FILE; do
  dos2unix "$FILE"
  # We should be indenting with 2 spaces, not with tabs.
	sed -i 's/\t/  /g' "$FILE"
  # We should not have trailing spaces
	sed -i 's/ +$//' "$FILE"
  # Files should not be longer than 100 lines:
  if [ "$(wc -l < "$FILE")" -gt 100 ]; then
    echo "$FILE has more than 100 lines. Dont use bash."
    exit 1
  fi
done
