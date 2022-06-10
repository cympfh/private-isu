all: bin/benchmarker
	./bin/benchmarker -t "http://localhost:80" -u /home/cympfh/solved/isucon/private-isu/benchmarker/userdata --debug

bin/benchmarker: *.go checker/*.go
	go build -o bin/benchmarker
