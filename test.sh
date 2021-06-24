#!/bin/bash
# This will be replaced with a better guide soon.
go build
rm -rf ~/commons-output/*
./commons -head="/home/matto/commons/frag/head.htm" -header="/home/matto/commons/frag/header.htm" -tprefix="example.com - " ~/commons/content ~/commons-output
