#!/bin/sh

# kill prev copy if exists
pkill -e -f go-whatsapp-proxy


# start a new one
./go-whatsapp-proxy &


