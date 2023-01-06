#!/bin/bash
for((;;))
do
   git push origin main
   if [ $? -eq 0 ]
   then
      exit
   fi
  
done
 
