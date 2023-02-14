FROM golang:buster

RUN mkdir /app
COPY . /app
WORKDIR /app

RUN apt-get update && apt-get upgrade -y \
  && apt-get install --no-install-recommends -y \
    bash \
    netcat \
# Cleaning cache:
   && apt-get purge -y --auto-remove -o APT::AutoRemove::RecommendsImportant=false \
   && apt-get clean -y && rm -rf /var/lib/apt/lists/*
RUN go mod download && go build -o /app/orders /app/cmd/orders

#ENTRYPOINT ["/app/orders"]
