#!/bin/bash
cd $SYNCDIR_WD
mkdir $SYNCDIR_COPYDIR
cd $SYNCDIR_COPYDIR

echo "a" > a.txt
echo "b" > b.txt
mkdir x
cd x
echo "c" > c.txt
mkdir y
cd y
echo "d" > d.txt