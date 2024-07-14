#!/bin/bash

cd $UTIL_ROOT
mkdir $UTIL_TEST_DIR
cd $UTIL_TEST_DIR
echo "123" > numbers.txt
echo "abc" > letters.txt
mkdir sub1
cd sub1
echo "a file in a sub-directory" > subfile.txt
echo "another file in a sub-directory" > anothersubfile.txt