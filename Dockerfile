FROM golang:1.9 as build

RUN cd / && go get -v github.com/disintegration/gift github.com/nsf/termbox-go
COPY main.go /main.go
RUN cd / && \
    CGO_ENABLED=0 GOOS=linux go build -a -tags "netgo static_build" -installsuffix netgo -ldflags "-w -s" -o invaders main.go

FROM scratch
LABEL maintainer "Sau Sheong Chang <sausheong@gmail.com>"

WORKDIR /
CMD ["/invaders"]
COPY imgs /imgs
COPY --from=build /invaders /
