==Conventions==
We follow all rules of shellcheck, and we follow https://google.github.io/styleguide/shellguide.html as close as possible.
Mainly that means:
* no scripts bigger than 100 lines
* indent with groups of 'two spaces' (instead of tabs, 4 spaces, etc.)
* No trailing spaces
* A lot more. Please read url and expand.

==Develop==
Run `make test` as much as possible. It will check bash scripts using shellcheck which greatly enhances quality.
Before commiting changes:
* add all your changes with `git add`
* run `make format` to see if it changes anything.
* check all changes with `git diff .`
* Make sure they are as you hoped. Fix if necessary and run `make format` again
* run `make test` just to be sure
* commit

==Work in progress==
format.sh is work in progress. We need to add as much rules as possible from
https://google.github.io/styleguide/shellguide.html
which is 'work in progress'.
Everytime you touch this folder, also apply one extra rule.
