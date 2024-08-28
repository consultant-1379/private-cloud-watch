GENDIR=../gen/cruxgen
GEN2DIR=../gen/ruckgen

PROTOS := $(basename $(wildcard *.proto))
PROTOS_GRPC := $(filter srv_%,$(PROTOS))

PB_GO := $(addsuffix .pb.go,$(PROTOS))
GRPC_GO := $(addsuffix .grpc.go,$(PROTOS_GRPC))
ALL_GO := $(addprefix $(GENDIR)/,$(PB_GO) monocle.pb.go monocle.pb.gw.go) $(addprefix $(GEN2DIR)/,$(GRPC_GO))

protos: $(ALL_GO)

# monocle.pb.go - Not built like other proto's. Avoiding crux generated goo since
# this is an end-user accessible API.
# monocle.pb.gw.go - the Swagger gateway/proxy code for monocle grpcs

$(GENDIR)/monocle.pb.go: monocle.proto
	mkdir -p $(GENDIR)
	protoc $< -I. -I$(GOPATH)/src \
		-I../vendor/github.com/grpc-ecosystem/grpc-gateway/third_party/googleapis \
		--grpc-gateway_out=logtostderr=true:$(GENDIR) \
		--go_out=plugins=grpc:$(GENDIR)

$(GENDIR)/monocle.pb.gw.go: $(GENDIR)/monocle.pb.go

$(GENDIR)/%.pb.go: %.proto
	mkdir -p $(GENDIR)
	protoc $< --go_out=plugins=grpc:$(GENDIR)

$(GEN2DIR)/%.grpc.go: %.proto
	mkdir -p $(GEN2DIR)
	server -o $(GEN2DIR) $*.proto

clean:
	rm -f $(ALL_GO)
