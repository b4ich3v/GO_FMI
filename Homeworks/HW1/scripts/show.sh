#!/bin/bash

go run . usernames.txt > web/out.csv
column -t -s';' web/out.csv
