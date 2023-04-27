#!/usr/bin/env bash

# This script replaces a normal card with a foil card I needed this because the
# tracking of foils was only added in version 3.5.0 of serra

# give set code as $1 like "dmr"
: ${1:?}

while true;  do
  read -p "$1> " card 
  serra add --foil ${1}/${card}
  serra remove ${1}/${card}
done
