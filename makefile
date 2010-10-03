all:
	make -C network install
	make -C client clean all

test:
	make -C network test

clean:
	make -C network clean
	make -C client clean

format:
	gofmt -w .
