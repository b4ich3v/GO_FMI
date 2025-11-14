#!/bin/bash

go run . usernames.txt > out.csv
column -t -s';' out.csv
