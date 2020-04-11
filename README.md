# StabbyShips

Was originally SuperSpatial, an attempt to re-create subspace, but ended up just making ships that run into each other.

*This is a proof of concept, and the code quality is roughly "mash the keyboard until it works"*

## How to run it:

1. Clone this repo
2. Clone this repo github.com/ScottBrooks/sos beside this one.
3. From ../sos run `make setup_win` and `make setup`.  For mac you'll need to edit the makefile and adjust as needed to get your platform binaries.
You may need a gcc version to build with cgo on windows.
4. From the spatial directory run `make setup` or `make setup_win` to get the schema compiler, and snapshot converter.
5. Run `make schema` to compile the schema(needs to be done the first time, and then after touching any of the .schema files)
6. Run `make snapshot` to prepare the initial snapshot.  Needs to be done the first time, and then ever editing the snapshot json file)
7. Run `make start_spatial` to start a local server.
8. From the root run `make balancer` to build and run the balancer worker
9. Run `go run cmd/client/main.go` to run the local client.




