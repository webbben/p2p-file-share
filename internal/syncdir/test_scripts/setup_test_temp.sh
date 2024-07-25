#!/bin/bash

cd $SYNCDIR_WD
mkdir $SYNCDIR_TEMP
cd $SYNCDIR_TEMP

dd if=/dev/zero of=largefile bs=1G count=1

mkdir a
cd a
dd if=/dev/zero of=file_a bs=1G count=1
dd if=/dev/zero of=file_aa bs=1G count=1

mkdir b
cd b
dd if=/dev/zero of=file_b bs=1G count=1
dd if=/dev/zero of=file_bb bs=1G count=2