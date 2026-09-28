# Demo image: the Spring Boot app AND the Go sidecar in ONE container.
# (Fine for a local demo; in Kubernetes they would be two containers in one pod.)
#
# Before building, copy your Spring Boot jar here as ./app.jar

# ---- Stage 1: compile the Go sidecar ----
FROM golang:1.27-alpine AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY *.go ./
RUN CGO_ENABLED=0 go build -o /sidecar .

# ---- Stage 2: Java runtime + jar + sidecar ----
FROM eclipse-temurin:21-jre
WORKDIR /app
COPY app.jar /app/app.jar
COPY --from=build /sidecar /app/sidecar
COPY start.sh /app/start.sh
RUN chmod +x /app/start.sh /app/sidecar

EXPOSE 8080 8090
ENTRYPOINT ["/app/start.sh"]
