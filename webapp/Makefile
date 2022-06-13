alp:
	tail -n 300 /tmp/nginx/log/access.log | alp ltsv -m '/image/.*,/js/.*,/posts/.*' -o count,1xx,2xx,3xx,4xx,5xx,method,uri,min,avg,max,sum --sort sum -r --format markdown

up:
	: > /tmp/mysql/slow.log
	docker compose up --build -d

down:
	docker compose down -v --rmi local

logs:
	docker compose logs -f

attach-mysql:
	docker exec -it webapp-mysql-1 mysql -uroot -proot -Disuconp

attach-nginx:
	docker exec -it webapp-nginx-1 bash

attach-app:
	docker exec -it webapp-app-1 bash
