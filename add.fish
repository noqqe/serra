#!/usr/bin/env fish

function serra_add ()
    echo ./serra add $argv[1]/$argv[2]
end

while true;
  read -l c
  serra_add $argv[1] $c
end
