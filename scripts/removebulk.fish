#!/usr/bin/env fish
set SET $argv[1]


# export and remove colors
serra cards --set $SET --min-count 2 --sort value  | gsed 's/\x1B[@A-Z\\\]^_]\|\x1B\[[0-9:;<=>?]*[-!"#$%&'"'"'()*+,.\/]*[][\\@A-Z^_`a-z{|}~]//g' > $SET 

# edit
nvim $SET

# remove formatting
cat $SET | gsed 's/^\* //' | gsed 's/x.*(/ /' | gsed 's/).*//' | grep -v "Total Value" > {$SET}.txt

# delete everything from serra
for x in (cat {$SET}.txt) ; echo serra remove -c  $x ; end

# cleanup 
rm $SET
rm {$SET}.txt
