#!/usr/bin/env sh

BINARY=/furyad/${BINARY:-cosmovisor}
ID=${ID:-0}
LOG=${LOG:-furyad.log}

if ! [ -f "${BINARY}" ]; then
	echo "The binary $(basename "${BINARY}") cannot be found. Please add the binary to the shared folder. Please use the BINARY environment variable if the name of the binary is not 'furyad'"
	exit 1
fi

BINARY_CHECK="$(file "$BINARY" | grep 'ELF 64-bit LSB executable, x86-64')"

if [ -z "${BINARY_CHECK}" ]; then
	echo "Binary needs to be OS linux, ARCH amd64"
	exit 1
fi

export FURYAD_HOME="/furyad/node${ID}/furyad"

if [ -d "$(dirname "${FURYAD_HOME}"/"${LOG}")" ]; then
    "${BINARY}" run "$@" --home "${FURYAD_HOME}" | tee "${FURYAD_HOME}/${LOG}"
else
    "${BINARY}" run "$@" --home "${FURYAD_HOME}"
fi