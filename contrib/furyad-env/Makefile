all: furyad-upgrade-env

furyad-upgrade-env:
	docker build --platform linux/amd64 --no-cache --build-arg entrypoint=cosmovisor-entrypoint.sh --tag furya/furyad-upgrade-env .

.PHONY: all furyad-upgrade-env
