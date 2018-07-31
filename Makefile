GOPKGS ?= ./...

fmt: goimports

goimports:
	@goimports -w --local github.com/supergiant/capacity cmd pkg

get-tools:
	go get -u github.com/golang/dep/cmd/dep@v0.5.0
	go get -u github.com/alecthomas/gometalinter
	gometalinter --install

lint: gometalinter

gometalinter:
	@gometalinter --deadline=50s --vendor \
	    --cyclo-over=50 --dupl-threshold=100 \
	    --disable-all \
	    --enable=vet \
	    --enable=deadcode \
	    --enable=golint \
	    --enable=vetshadow \
	    --enable=gocyclo \
	    --enable=misspell \
	    --skip=test \
	    --skip=bindata \
	    --skip=vendor \
	    --tests \
	    $(GOPKGS)

