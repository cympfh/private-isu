bench: bin/benchmarker
	./bin/benchmarker -t "http://localhost:80" -u /home/cympfh/solved/isucon/private-isu/benchmarker/userdata --debug | jq '{pass,score,success,fail,messages:.messages[:4]}' -cM

bin/benchmarker: *.go checker/*.go
	go build -o bin/benchmarker

ping:
	curl -I localhost:80

mysqldumpslow:
	mysqldumpslow /tmp/mysql/slow.log -s t -t 5

alp:
	cat /tmp/nginx/log/access.log | alp ltsv -m '/image/.*jpg,/image/.*png,/image/.*gif,/js/.*,/posts/.*,/@.*' -o count,1xx,2xx,3xx,4xx,5xx,method,uri,min,avg,max,sum --sort sum -r --format markdown
