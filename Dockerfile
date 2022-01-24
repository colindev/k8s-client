FROM golang as builder

COPY . /src/
WORKDIR /src
RUN CGO_ENABLED=0 \
    go build -a -mod vendor -o /bin/k8sclient .

FROM google/cloud-sdk:slim

COPY --from=builder /bin/k8sclient /bin/

ENTRYPOINT ["k8sclient"]
