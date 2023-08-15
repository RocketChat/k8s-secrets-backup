FROM golang:1.20.4

RUN mkdir /app

COPY ./main.go ./go.sum ./go.mod ./env.go /app/

WORKDIR /app

RUN GOOS=linux go build -o k8s-secrets-backup . 

CMD ["/app/k8s-secrets-backup"]
