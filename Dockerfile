FROM golang:1.20 AS go_builder

WORKDIR /app

# pre-copy/cache go.mod for pre-downloading dependencies and only redownloading them in subsequent builds if they change
COPY go.mod go.sum ./
RUN go mod download && go mod verify

COPY . .
RUN go build 

FROM node:20-alpine AS tailwind_builder
COPY --from=go_builder /app /app
WORKDIR /app

RUN npm install
RUN npx tailwindcss -i ./web/style.css -o ./web/static/style.css

FROM golang:1.20
COPY --from=tailwind_builder /app /app
WORKDIR /app

ENTRYPOINT ["./websearch"]
