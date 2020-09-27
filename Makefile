build:
	go build -o mondane

clean:
	go clean
	rm -rf mondane

.PHONY: all build $(SERVICES) $(CLIENTS)
