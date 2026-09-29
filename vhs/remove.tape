# serra remove - take cards back out of the collection
#
# Undoes what add.tape added, so recording the full set of tapes leaves the
# demo database unchanged.
Output ../imgs/remove.gif
Source common.tape
Set Height 180
Source setup.tape

Type "./serra remove mh3/38"
Enter
Sleep 3s
Type "./serra remove --count 4 --foil one/23"
Enter
Sleep 4s
