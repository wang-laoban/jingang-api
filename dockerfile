FROM alpine:latest
MAINTAINER haibo

WORKDIR /home/app
COPY ./cmd /home/app/
RUN ls -l /home/app
RUN chmod +x /home/app/cmd
ENTRYPOINT ["/home/app/cmd"]
