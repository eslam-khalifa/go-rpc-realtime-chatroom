go run server.go & 
sleep 1

konsole -e /bin/bash --rcfile <(echo "go run client.go") & 

konsole -e /bin/bash --rcfile <(echo "go run client.go") & 

konsole -e /bin/bash --rcfile <(echo "go run client.go") & 

konsole -e /bin/bash --rcfile <(echo "go run client.go") & 

sleep 60

echo "thanks"

kill $(jobs -p)
kill $(ps aux | grep 'exe/server' | awk '{print $2}')
