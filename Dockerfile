# ---------------------------------------------------------------------
#  The first stage container, for building the application
# ---------------------------------------------------------------------
FROM golang:1.15.0 as builder

# Add the keys
# ARG user
# ENV user=$user
# ARG personal_access_token
# ENV personal_access_token=$personal_access_token

WORKDIR $GOPATH/src/github.com/wooshot/grpcTest

COPY . .

# RUN git config \
#     --global \
#     url."https://${user}:${personal_access_token}@privategitlab.com".insteadOf \
#     "https://privategitlab.com"

RUN GIT_TERMINAL_PROMPT=1 \
    GOARCH=amd64 \
    GOOS=linux \
    CGO_ENABLED=0 \
    go build -v --installsuffix cgo --ldflags="-s" -o main

# ---------------------------------------------------------------------
#  The second stage container, for running the application
# ---------------------------------------------------------------------
FROM alpine:3.8
COPY --from=builder ./main /bin

WORKDIR /bin

ENTRYPOINT ["./main"]