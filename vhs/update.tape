# serra update - refresh prices from Scryfall
#
# A full update pulls the entire Scryfall bulk file and imports every
# printing, which is far too long for a README gif, so the recording is cut
# short once the interesting part - the download and import - is on screen.
Output ../imgs/update.gif
Source common.tape
Set Height 300
# The per-set progress bars are a fixed 50 cells wide plus the set name and
# counters - at the default font size they wrap and turn into a mess.
Set FontSize 13
Source setup.tape

Type "./serra update"
Enter
Sleep 38s
Ctrl+C
Sleep 2s
