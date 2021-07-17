#! /usr/bin/sh

read -p "This will remove the database files and logfiles. Do you want to continue? (y/N) " -n 1 -r confirmation || exit
confirmation=${confirmation:N}
echo
if [[ $confirmation =~ ^[Yy]$ ]]
then
    echo "Deleting the listed files"
    rm -rf bin/* db/* logs/*
else
    echo "Aborting"
fi
