FROM golang:1.25.5-alpine AS build
WORKDIR /app
COPY ./src .
RUN go build -o /weather-app ./

FROM alpine:3.18
COPY --from=build /weather-app /weather-app
EXPOSE 8080
ENTRYPOINT ["/weather-app"]
