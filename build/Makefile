# Build path
BUILD_BIN_PATH=../../../../../bin

# Binary 
LCND=${BUILD_BIN_PATH}/lcnd
GOBIN_LCND=${GOBIN}/lcnd

#CLI=${BUILD_BIN_PATH}/cli
#GOBIN_CLI=${GOBIN}/cli

TESTS=${BUILD_BIN_PATH}/tests

# Install path
INSTALL_DIR=./bin/

all:
	go install -tags=embed ../cmd/...
	@mkdir -p bin

	@if test -f ${LCND}; then \
		mv ${LCND} ${CLI} ${INSTALL_DIR} ;  \
		echo "make done, mv the lcnd cli to ${INSTALL_DIR}";  \
	elif test -f ${GOBIN_LCND}; then \
		mv ${GOBIN_LCND} ${GOBIN_CLI} ${INSTALL_DIR} ; \
		echo "make done, mv the ${GOBIN_LCND} to ${INSTALL_DIR}"; \
	else \
		echo "make fail, not mv lcnd to ${INSTALL_DIR}";\
		fi

tests:
	go install -tags=embed ../tests/...
	mv ${TESTS} ${INSTALL_DIR}

.PHONY:all tests
