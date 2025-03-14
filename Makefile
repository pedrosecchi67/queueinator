CC=go build
TARGET=queueinator

TEST=test/
TEST_SCPT=test.sh

BIN=$(shell dirname $$(which ls))

.PHONY: compile
all: compile

.PHONY: clean
clean:
	rm $(TARGET)

compile:
	$(CC) $(TARGET)

install:
	@echo "Binary directory: $(BIN)"

	cp $(TARGET) $(BIN)

uninstall:
	@echo "Removing $(BIN)/$(TARGET)..."

	rm $(BIN)/$(TARGET)

.PHONY: test
test:
	cd $(TEST); bash $(TEST_SCPT); cd ..

