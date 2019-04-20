list:
	@echo "sync-codes: 同步代码"
	@echo "build: 编译可执行文件"
	@echo "test: 测试"

sync-codes:
	git pull

build:
	PATH=/tmp/govendor/bin:$(PATH)
	GOPATH=/tmp/govendor/:$(GOPATH)
	CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -ldflags '-w' -o server ./main.go

deps:
	mkdir -p /tmp/govendor/src/gitlab.wallstcn.com/wscnbackend
	mkdir -p /tmp/govendor/src/gitlab.wallstcn.com/baoer/flash
	@if [ "$(CI_COMMIT_REF_NAME)" = "master" ]; then\
		echo "checkout ivankastd:master";\
		git clone git@gitlab.wallstcn.com:wscnbackend/govendor.git /tmp/govendor_temp;\
		git clone git@gitlab.wallstcn.com:wscnbackend/ivankastd.git /tmp/govendor/src/gitlab.wallstcn.com/wscnbackend/ivankastd;\
		git clone git@gitlab.wallstcn.com:wscnbackend/ivankaprotocol.git /tmp/govendor/src/gitlab.wallstcn.com/wscnbackend/ivankaprotocol;\
		git clone git@gitlab.wallstcn.com:wscnbackend/witsstd.git /tmp/govendor/src/gitlab.wallstcn.com/wscnbackend/witsstd; \
		git clone git@gitlab.wallstcn.com:wscnbackend/witsgateway.git /tmp/govendor/src/gitlab.wallstcn.com/wscnbackend/witsgateway; \
		git clone git@gitlab.wallstcn.com:wscnbackend/ivankaway.git /tmp/govendor/src/gitlab.wallstcn.com/wscnbackend/ivankaway; \
		git clone git@gitlab.wallstcn.com:wscnbackend/wscnpay.git /tmp/govendor/src/gitlab.wallstcn.com/wscnbackend/wscnpay; \
		git clone git@gitlab.wallstcn.com:wscnbackend/ddcstd.git /tmp/govendor/src/gitlab.wallstcn.com/wscnbackend/ddcstd;\
		git clone git@gitlab.wallstcn.com:wscnbackend/wows.git  /tmp/govendor/src/gitlab.wallstcn.com/wscnbackend/wows;\
        git clone git@gitlab.wallstcn.com:baoer/xgbbackend.git /tmp/govendor/src/gitlab.wallstcn.com/baoer/xgbbackend;\
        git clone git@gitlab.wallstcn.com:baoer/backend.git /tmp/govendor/src/gitlab.wallstcn.com/baoer/backend;\
        git clone git@gitlab.wallstcn.com:baoer/flash/flashstd.git /tmp/govendor/src/gitlab.wallstcn.com/baoer/flash/flashstd;\
        git clone git@gitlab.wallstcn.com:baoer/flash/flashcommon.git /tmp/govendor/src/gitlab.wallstcn.com/baoer/flash/flashcommon;\
	elif [ "$(CI_COMMIT_REF_NAME)" = "stage" ]; then\
		echo "checkout ivankastd:stage";\
		git clone git@gitlab.wallstcn.com:wscnbackend/govendor.git /tmp/govendor_temp;\
		git clone -b stage git@gitlab.wallstcn.com:wscnbackend/ivankastd.git /tmp/govendor/src/gitlab.wallstcn.com/wscnbackend/ivankastd;\
		git clone -b stage git@gitlab.wallstcn.com:wscnbackend/ivankaprotocol.git /tmp/govendor/src/gitlab.wallstcn.com/wscnbackend/ivankaprotocol;\
		git clone git@gitlab.wallstcn.com:wscnbackend/witsstd.git /tmp/govendor/src/gitlab.wallstcn.com/wscnbackend/witsstd; \
		git clone git@gitlab.wallstcn.com:wscnbackend/witsgateway.git /tmp/govendor/src/gitlab.wallstcn.com/wscnbackend/witsgateway; \
		git clone git@gitlab.wallstcn.com:wscnbackend/ivankaway.git /tmp/govendor/src/gitlab.wallstcn.com/wscnbackend/ivankaway; \
		git clone git@gitlab.wallstcn.com:wscnbackend/wscnpay.git /tmp/govendor/src/gitlab.wallstcn.com/wscnbackend/wscnpay; \
		git clone git@gitlab.wallstcn.com:wscnbackend/ddcstd.git /tmp/govendor/src/gitlab.wallstcn.com/wscnbackend/ddcstd;\
        git clone git@gitlab.wallstcn.com:wscnbackend/wows.git  /tmp/govendor/src/gitlab.wallstcn.com/wscnbackend/wows;\
		git clone git@gitlab.wallstcn.com:baoer/xgbbackend.git /tmp/govendor/src/gitlab.wallstcn.com/baoer/xgbbackend;\
		git clone git@gitlab.wallstcn.com:baoer/backend.git /tmp/govendor/src/gitlab.wallstcn.com/baoer/backend;\
		git clone git@gitlab.wallstcn.com:baoer/flash/flashstd.git /tmp/govendor/src/gitlab.wallstcn.com/baoer/flash/flashstd;\
		git clone git@gitlab.wallstcn.com:baoer/flash/flashcommon.git /tmp/govendor/src/gitlab.wallstcn.com/baoer/flash/flashcommon;\
	else\
		echo "checkout ivankastd:sit";\
		git clone -b sit git@gitlab.wallstcn.com:wscnbackend/govendor.git /tmp/govendor_temp;\
		git clone -b sit git@gitlab.wallstcn.com:wscnbackend/ivankastd.git /tmp/govendor/src/gitlab.wallstcn.com/wscnbackend/ivankastd;\
		git clone -b sit git@gitlab.wallstcn.com:wscnbackend/ivankaprotocol.git /tmp/govendor/src/gitlab.wallstcn.com/wscnbackend/ivankaprotocol;\
		git clone -b sit git@gitlab.wallstcn.com:wscnbackend/witsstd.git /tmp/govendor/src/gitlab.wallstcn.com/wscnbackend/witsstd; \
		git clone -b sit git@gitlab.wallstcn.com:wscnbackend/witsgateway.git /tmp/govendor/src/gitlab.wallstcn.com/wscnbackend/witsgateway; \
		git clone -b sit git@gitlab.wallstcn.com:wscnbackend/ivankaway.git /tmp/govendor/src/gitlab.wallstcn.com/wscnbackend/ivankaway; \
		git clone -b sit git@gitlab.wallstcn.com:wscnbackend/wscnpay.git /tmp/govendor/src/gitlab.wallstcn.com/wscnbackend/wscnpay; \
		git clone -b sit git@gitlab.wallstcn.com:wscnbackend/ddcstd.git /tmp/govendor/src/gitlab.wallstcn.com/wscnbackend/ddcstd;\
		git clone -b sit git@gitlab.wallstcn.com:wscnbackend/wows.git  /tmp/govendor/src/gitlab.wallstcn.com/wscnbackend/wows;\
		git clone -b sit git@gitlab.wallstcn.com:baoer/xgbbackend.git /tmp/govendor/src/gitlab.wallstcn.com/baoer/xgbbackend;\
		git clone -b sit git@gitlab.wallstcn.com:baoer/backend.git /tmp/govendor/src/gitlab.wallstcn.com/baoer/backend;\
		git clone -b sit git@gitlab.wallstcn.com:baoer/flash/flashstd.git /tmp/govendor/src/gitlab.wallstcn.com/baoer/flash/flashstd;\
		git clone -b sit git@gitlab.wallstcn.com:baoer/flash/flashcommon.git /tmp/govendor/src/gitlab.wallstcn.com/baoer/flash/flashcommon;\
	fi
	cp -r /tmp/govendor_temp/vendor/* /tmp/govendor/src
	mkdir -p /tmp/govendor/bin
	mkdir -p /go/src/gitlab.wallstcn.com/$(CI_PROJECT_NAMESPACE)/
	cp -R "/builds/$(CI_PROJECT_NAMESPACE)/$(SERVICE_NAME)" "/go/src/gitlab.wallstcn.com/$(CI_PROJECT_NAMESPACE)/$(SERVICE_NAME)/"

istio:
	wget -q -P /tmp/govendor/bin/ https://wallstreetcn-1252820405.cossh.myqcloud.com/e237c04d-a078-4c5c-a9c4-516140bc2543.protoc-gen-go
	mv /tmp/govendor/bin/e237c04d-a078-4c5c-a9c4-516140bc2543.protoc-gen-go /tmp/govendor/bin/protoc-gen-go
	chmod +x /tmp/govendor/bin/protoc-gen-go
	GOPATH=/tmp/govendor/ sh /tmp/govendor/src/gitlab.wallstcn.com/wscnbackend/witsstd/protobuf.sh
	GOPATH=/tmp/govendor/ sh /tmp/govendor/src/gitlab.wallstcn.com/wscnbackend/ivankaprotocol/protobuf.sh
	GOPATH=/tmp/govendor/ sh /tmp/govendor/src/gitlab.wallstcn.com/baoer/flash/flashcommon/protobuf.sh
	GOPATH=/tmp/govendor/ sh /tmp/govendor/src/gitlab.wallstcn.com/wscnbackend/wows/wowsstd/protobuf.sh
	GOPATH=/tmp/govendor/ sh /tmp/govendor/src/gitlab.wallstcn.com/wscnbackend/ddcstd/protobuf.sh

test:
	go test
