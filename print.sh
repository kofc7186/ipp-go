#!/bin/sh
#

TOPIC="print_queue"

function portable_base64_encode() {
  if [[ "$OSTYPE" == "darwin"* ]]; then
    # macOS
    cat $1 | base64 | tr -d '\n'
  else
    # Linux (assuming GNU coreutils or compatible)
    cat $1 | base64 -w0
  fi
}

if ! [ -d $1 ]; then
  echo "one command line argument required; must be a name of a directory" && exit 1
fi

for file in `ls $1/`;
do
  B64=$(portable_base64_encode $1/$file)
  SON=$(echo $1/$file|tr -cd '0-9')
  parity="odd"
  if [ $((SON & 1)) -eq 0 ]; then
    parity="even"
  fi
  gcloud pubsub topics publish $TOPIC --message "${B64}" --attribute="son=${SON},parity=${parity},reprint=false" --project fishfry2021
  sleep 1
done
