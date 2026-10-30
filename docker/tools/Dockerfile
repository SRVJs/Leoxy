from golang:1.27.3-alpine

workdir /app

COPY ./tools/go.mod ./tools/go.sum ./

RUN go mod download

copy . .

RUN CGO_ENABLED=0 GOOS=linux go build -o /out/lx-2 ./tools/cmd/

COPY --from=builder /out/lx-2 /usr/local/bin/leoxy
 
CMD ["./usr/local/bin/lx-2"]
