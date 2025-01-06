sudo docker stop jgapi
sudo docker rm -f jgapi
sudo docker rmi -f jgapi:0.1
rm -f api
mv cmd api
sudo docker build -t jgapi:0.1 .
sudo docker run -d --restart always --name jgapi jgapi:0.1
sudo docker logs -f jgapi
