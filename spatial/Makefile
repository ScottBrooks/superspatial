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
	spatial alpha local launch

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
deploy:
	rm -rf build/client
	rm -rf build/server
	mkdir -p build/server
	mkdir -p build/client
	cp launch.sh build/server/
	cd ..; go build -o spatial/build/client/client.exe cmd/client/main.go
	cp ../../sos/improbable_worker.dll build/client
	cp -r ../assets build/client/
	cp -r launcher_client_config.json build/client/
	docker run -v $(ROOT_DIR)/../../:/work -ti xgo-bash bash -c "cd /work/superspatial; go build -o spatial/build/server/server cmd/server/main.go"
	cp ../../sos/libimprobable_worker.so build/server
	chmod +x build/server/server
	chmod +x build/server/libimprobable_worker.so
	spatial alpha cloud upload -a superspatial --force
	spatial alpha cloud launch -d superspatial -a superspatial

