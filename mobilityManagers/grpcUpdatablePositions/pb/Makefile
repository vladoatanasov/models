TARGETS=service.pb.go

all: $(TARGETS) Makefile

clean:
	rm -f $(TARGETS)

%.pb.go : %.proto
	protoc --go_out=plugins=grpc:. $<
