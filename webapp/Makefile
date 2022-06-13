alp:
	tail -n 300 /tmp/nginx/log/access.log | alp ltsv -m '/image/.*,/js/.*,/posts/.*' -o count,1xx,2xx,3xx,4xx,5xx,method,uri,min,avg,max,sum --sort sum -r
