#!/bin/bash

# Configuration MySQL
export REMOTESHELL_MYSQL_ENABLED=true
export REMOTESHELL_MYSQL_HOST=ol35063-001.eu.clouddb.ovh.net
export REMOTESHELL_MYSQL_PORT=35177
export REMOTESHELL_MYSQL_USER=tms
export REMOTESHELL_MYSQL_PASSWORD=ADrylDigit59
export REMOTESHELL_MYSQL_DATABASE=tms

# Configuration serveur
export REMOTESHELL_SERVER_HOST=0.0.0.0
export REMOTESHELL_SERVER_PORT=8080

# Lancer le serveur
./server

