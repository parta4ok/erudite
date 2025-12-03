FROM golang:1.25-alpine AS builder

WORKDIR /common

RUN go install github.com/golangci/golangci-lint/v2/cmd/golangci-lint@v2.1.6
COPY ./.golangci.yml ./

COPY go.mod go.sum ./

RUN go mod download

COPY . .

RUN golangci-lint run -c /common/.golangci.yml
RUN go test ./...

FROM builder AS question_builder
WORKDIR /common/question
RUN go build -o ./cmd/question_service/question ./cmd/question_service/main.go

FROM builder AS auth_builder
WORKDIR /common/auth
RUN go build -o ./cmd/app/auth ./cmd/app/main.go

FROM builder AS notificationhub_builder
WORKDIR /common/notificationhub
RUN go build -o ./cmd/app/notificationhub ./cmd/app/main.go

FROM builder AS reporting_builder
WORKDIR /common/reporting
RUN go build -o ./cmd/app/reporting ./cmd/app/main.go

FROM alpine:latest AS question
WORKDIR /app
COPY --from=question_builder /common/question/cmd/question_service/question .
COPY --from=question_builder /common/deployment/question.yaml .
CMD ["./question", "--config=question.yaml"]

FROM alpine:latest AS auth
WORKDIR /app
COPY --from=auth_builder /common/auth/cmd/app/auth .
COPY --from=auth_builder /common/deployment/auth.yaml .
CMD ["./auth", "--config=auth.yaml"]

FROM alpine:latest AS notificationhub
WORKDIR /app
COPY --from=notificationhub_builder /common/notificationhub/cmd/app/notificationhub .
COPY --from=notificationhub_builder /common/deployment/notificationhub.yaml .
CMD ["./notificationhub", "--config=notificationhub.yaml"]

FROM alpine:latest AS reporting
WORKDIR /app
COPY --from=reporting_builder /common/reporting/cmd/app/reporting .
COPY --from=reporting_builder /common/deployment/reporting.yaml .
CMD ["./reporting", "--config=reporting.yaml"]
