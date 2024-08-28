INSTANCE=prometheus

CAS=$(INSTANCE)-root-ca

HOSTS=prometheus

CLIENTS=promuser

FILES=$(addsuffix .crt,$(CAS)) $(addsuffix .key,$(CAS)) \
      $(addsuffix -host.crt,$(HOSTS)) $(addsuffix -host.key,$(HOSTS)) \
      $(addsuffix -client.crt,$(CLIENTS)) $(addsuffix -client.key,$(CLIENTS))

all: $(FILES)

#.SECONDARY: # un-comment to save intermediate files (like *.req)

%.crt: %.req $(INSTANCE)-root-ca.key
	./sign-cert-req $(INSTANCE)-root-ca $*

%.req %.key:
	./req-host-cert $*

%-root-ca.crt %-root-ca.key:
	./make-ca-root "$*"

%.der: %.crt
	openssl x509 -in $< -outform DER -out $@

%.pfx: %.crt %.key
	touch $*.pem && chmod 600 $*.pem && \
		cat $*.crt $*.key >>$*.pem && \
	openssl pkcs12 -export -aes128 -in $*.pem -aes128 -out $@
	rm -f $*.pem

%.show: %.crt
	openssl x509 -in $< -noout -text

clean:
	rm -f *-host.key *-host.crt *-host.req
	rm -f *-client.key *-client.crt *-client.req *-client.pfx

clean-all:
	rm -f *.srl *.key *.crt *.req *.pfx
