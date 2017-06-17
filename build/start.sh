#/bin/bash
killall lcnd
for i in 1 2 3 4
do
 	./bin/lcnd --config=$i.yaml &
done
