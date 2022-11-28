#!/usr/bin/env fish

function serra_add ()

    # split input args
    set -l args (string split " " $argv)

    # check if user wants to exit
    if [ "$args[2]" = "exit" ]
      exit 0
    end

    # if usage contains a number at the end, add multiple ones
    # if not, add a single one
    if test -n "$args[3]"
      serra add $args[1]/$args[2] --count=$args[3] -u
      echo serra add $args[1]/$args[2] --count=$args[3] >> add.log
    else
      serra add $args[1]/$args[2] -u
      echo serra add $args[1]/$args[2] >> add.log
    end
end

while true;
  read -l c
  serra_add $argv[1] $c
end
