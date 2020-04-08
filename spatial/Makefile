SDK_VERSION=14.3.0

run: ../*.go
	go build ../cmd/server && ./server

.PHONY: client
client:
	go build ../cmd/client && cd .. && ./spatial/client

test: *.go lib/*.go
	go test ./...

.PHONY: start_spatial
start_spatial:
	spatial alpha local launch --launch_config launch_configs/dev.json

.PHONY: schema
schema:
	./bin/schema_compiler --schema_path=./schema/ ./schema/superspatial.schema --descriptor_set_out=./schema/bin/schema.descriptor --bundle_out=./schema/bin/schema.bundle
	
.PHONY: snapshot
snapshot:
	./bin/snapshot_converter convert-json snapshot.json json snapshots/default.snapshot binary schema/bin/schema.bundle

#Linux
.PHONY: setup
setup:
	spatial package get tools schema_compiler-x86_64-linux ${SDK_VERSION} ./bin --unzip
	spatial package get tools snapshot_converter-x86_64-linux ${SDK_VERSION} ./bin --unzip
	spatial package get --force --unzip schema standard_library ${SDK_VERSION} ./schema

#Windows
.PHONY: setup_win
setup_win:
	spatial package get tools schema_compiler-x86_64-win32 ${SDK_VERSION} ./bin --unzip
	spatial package get tools snapshot_converter-x86_64-win32 ${SDK_VERSION} ./bin --unzip
	spatial package get --force --unzip schema standard_library ${SDK_VERSION} ./schema

.PHONY: clean
clean:
	rm -rf lib/c_sdk libworker.so worker.dll ./bin/ || true

.PHONY: build-container
build-container:
	docker built -t xgo-bash .
	
ROOT_DIR:=$(shell dirname $(realpath $(firstword $(MAKEFILE_LIST))))

.PHONY: deploy
deploy: export MSYS_NO_PATHCONV=1
deploy:
	rm -rf build/LauncherClient
	rm -rf build/balancer
	mkdir -p build/balancer
	mkdir -p build/LauncherClient
	cp launch.sh build/balancer/
	cd ..; go build -o spatial/build/LauncherClient/client.exe cmd/client/main.go
	cp ../../sos/improbable_worker.dll build/LauncherClient
	cp -r ../assets build/LauncherClient/
	cp -r launcher_client_config.json build/LauncherClient/
	#docker run -v $(ROOT_DIR)/../../:/work -ti xgo-bash bash -c "cd /work/superspatial; go build -o spatial/build/balancer/server cmd/server/main.go; go build -o spatial/build/balancer/balancer cmd/balancer/main.go; go build -o spatial/build/balancer/bot cmd/bot/main.go"
	docker run -v "/c/work:/work" -ti xgo-bash bash -c "cd /work/superspatial; go build -o spatial/build/balancer/server cmd/server/main.go; go build -o spatial/build/balancer/balancer cmd/balancer/main.go; go build -o spatial/build/balancer/bot cmd/bot/main.go"
	cp ../../sos/libimprobable_worker.so build/balancer
	spatial alpha cloud upload -a superspatial --force
	spatial alpha cloud launch -d superspatial -a superspatial 

