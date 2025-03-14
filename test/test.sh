
# run a server that receives the text of lorem ipsum (located within a folder),
# greps it and calculates the number of occurrences of the preposition "in".
# Outputs to stdio.
# True answer is 7
../queueinator serve 'cat fold/lipsum.txt | grep -o "in" | wc -l' 8080 -n 3 &
PID_SERVER=$!

GROUND_TRUTH=7

sleep 2

PID_CLIENTS=()
for fold in `ls -d job*`; do
	cd $fold
	../../queueinator run localhost 8080 &
	PID=$!
	PID_CLIENTS+=("$PID")
	cd ..
done

for PID in ${PID_CLIENTS[@]}; do
	wait $PID
done

kill -9 $PID_SERVER

for fold in `ls -d job*`; do
	ANSWER=`cat $fold/queueinator.log`

	if [[ "$ANSWER" != "$GROUND_TRUTH" ]]; then
		echo "WARNING: ground truth not recovered in lorem ipsum test"
		exit 1
	fi
done

echo "Tests ran successfully!"

