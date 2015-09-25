# `git-domain`

`git-domain` is a `git` subcommand that lets you figure out whose domain you are
stepping into. To guess the suitability of ownership both historic (commit)
information and current (blame) information are calculated per author as shares
of the total stats and summed using weights that can be tweaked.

## TODO

[ ] If arg is filename just check that file, if the arg is a folder name check stats
  for each file in the folder (and recurse if flag is set)
[ ] On folder check also try to figure out who owns the folder
[ ] Recursive flag
[ ] Move everything to proper structs (methods, printers etc)
[ ] Parallelize current/historic checks
