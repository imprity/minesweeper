#!/bin/bash

xdg-open http://127.0.0.1:6969
http-server -c-1 -p6969 ./web_build/
