# Hidden shell preparation shared by every serra tape.
#
# Sourced after the Set commands and before the first visible Type. It
# points serra at the disposable demo database from demo-db.sh, so a
# recording never reads - let alone writes - a real collection.

Hide
# serra resolves ./sounds, ./templates and ./assets relative to the working
# directory, so recording has to happen from the repo root.
Type "cd $(git rev-parse --show-toplevel)"
Enter
Type "export MONGODB_URI=$(vhs/demo-db.sh uri) SERRA_CURRENCY=EUR"
Enter
Type "export PS1='> '"
Enter
Type "clear"
Enter
Show
