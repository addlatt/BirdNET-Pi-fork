#!/usr/bin/env bash

source /etc/birdnet/birdnet.conf
cd /home/$BIRDNET_USER/BirdNET-Pi

# Use the virtual environment's Python which has birdnet package installed
./birdnet/bin/python3 -c 'from birdnet.helpers import set_label_file; set_label_file()'

cd -
