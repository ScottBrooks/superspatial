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

.PHONY: start_spatial_old
start_spatial_old:
	cd old_config && spatial build build-config && spatial local launch

.PHONY: schema
schema:
	./bin/schema_compiler --schema_path=./schema/ ./schema/superspatial.schema --descriptor_set_out=./schema/bin/schema.descriptor --bundle_out=./schema/bin/schema.bundle
	
.PHONY: snapshot
snapshot:
	./bin/snapshot_converter convert-json snapshot.json json snapshots/default.snapshot binary schema/bin/schema.bundle

#Linux
.PHONY: setup
setup:
	spatial package get worker_sdk c-dynamic-x86_64-gcc510-linux ${SDK_VERSION} ./lib/c_sdk --unzip
	spatial package get worker_sdk c_headers ${SDK_VERSION} ./lib/c_sdk --unzip
	spatial package get tools schema_compiler-x86_64-linux ${SDK_VERSION} ./bin --unzip
	spatial package get tools snapshot_converter-x86_64-linux ${SDK_VERSION} ./bin --unzip
	spatial package get --force --unzip schema standard_library ${SDK_VERSION} ./schema
	cp lib/c_sdk/libimprobable_worker.so ./

.PHONY: clean
clean:
	rm -rf lib/c_sdk libworker.so worker.dll ./bin/ || true
	

.PHONY: deploy
deploy:
	rm -rf build/server
	mkdir -p build/server
	cp config/launch.sh build/server/
	go build -o build/server/server ./cmd/server
	cp lib/c_sdk/libworker.so build/server
	spatial alpha cloud upload -a boids --force
	spatial alpha cloud launch -d boids -a boids

