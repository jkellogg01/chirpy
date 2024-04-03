BIN_DIR := bin
TARGET := $(BIN_DIR)/chirpy

dev: build
	$(TARGET) --dev

run: build
	$(TARGET)

build:
	go build -o $(TARGET)

clean:
	$(RM) $(BIN_DIR)/*
