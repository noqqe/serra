# serra add - add cards to the collection
#
# Recorded near the end of a run because it writes: remove.tape undoes
# exactly these additions so the demo database ends where it started.
Output ../imgs/add.gif
Source common.tape
Set Height 200
Source setup.tape

Type "./serra add mh3/38"
Enter
Sleep 3s
Type "./serra add --count 4 --foil one/23"
Enter
Sleep 3s
Type "./serra add --unique usg/321"
Enter
Sleep 4s
