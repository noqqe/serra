#!/usr/bin/env fish

function serra_add ()
    set -l args (string split " " $argv)
    if test -n "$args[3]"
      ./serra add $args[1]/$args[2] --count=$args[3]
      echo ./serra add $args[1]/$args[2] --count=$args[3] >> add.log
    else
      ./serra add $args[1]/$args[2]
      echo ./serra add $args[1]/$args[2] >> add.log
    end
end

while true;
  read -l c
  serra_add $argv[1] $c
end
