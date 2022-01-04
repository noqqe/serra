#!/usr/bin/env fish

function serra_add ()
    ./serra add $argv[1]/$argv[2]
    echo ./serra add $argv[1]/$argv[2] >> add.log
end

while true;
  read -l c
  serra_add $argv[1] $c
end
